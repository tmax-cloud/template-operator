/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	tmaxiov1 "github.com/tmax-cloud/template-operator/api/v1"
)

// ClusterTemplateClaimReconciler reconciles a ClusterTemplateClaim object
type ClusterTemplateClaimReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const claimFinalizer = "clustertemplateclaims.tmax.io/finalizer"
const claimLabel = "clustertemplateclaims.tmax.io/claim"

// +kubebuilder:rbac:groups=tmax.io,resources=clustertemplateclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=clustertemplateclaims/status,verbs=get;update;patch

func (r *ClusterTemplateClaimReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("clustertemplateclaim", req.NamespacedName)
	logger.Info("Reconciling ClusterTemplateClaim")

	// Get the ClusterTemplateClaim instance
	claim := &tmaxiov1.ClusterTemplateClaim{}
	if err := r.Client.Get(context.TODO(), req.NamespacedName, claim); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	if claim.Status.Handled {
		logger.Info("Already handled claim")
		return ctrl.Result{}, nil
	}

	if r.checkClusterTemplateExist(claim) {
		status := &tmaxiov1.ClusterTemplateClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             fmt.Sprintf("ClusterTemplate %s already exist", claim.Spec.ResourceName),
			Status:             tmaxiov1.Rejected,
			Handled:            false,
		}
		return r.updateClusterTemplateClaimStatus(claim, status)
	}

	exist, template := r.getTemplateIfExist(claim)
	if !exist {
		status := &tmaxiov1.ClusterTemplateClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             fmt.Sprintf("Fail to get template %s", claim.Spec.TemplateName),
			Status:             tmaxiov1.Error,
			Handled:            false,
		}
		return r.updateClusterTemplateClaimStatus(claim, status)
	}

	switch claim.Status.Status {
	case "":
		status := &tmaxiov1.ClusterTemplateClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             "Waiting for admin permission",
			Status:             tmaxiov1.Awating,
			Handled:            false,
		}
		return r.updateClusterTemplateClaimStatus(claim, status)
	case tmaxiov1.Approved:
		if err := r.createClusterTemplate(claim, template); err != nil {
			status := &tmaxiov1.ClusterTemplateClaimStatus{
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             "Error occurs while creating cluster template",
				Status:             tmaxiov1.Error,
				Handled:            false,
			}
			return r.updateClusterTemplateClaimStatus(claim, status)
		}

		status := &tmaxiov1.ClusterTemplateClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             "Succeed to create cluster template",
			Status:             tmaxiov1.Approved,
			Handled:            true,
		}
		return r.updateClusterTemplateClaimStatus(claim, status)
	case tmaxiov1.Rejected:
		status := &tmaxiov1.ClusterTemplateClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             "Rejected by admin",
			Status:             tmaxiov1.Rejected,
			Handled:            false,
		}
		return r.updateClusterTemplateClaimStatus(claim, status)
	}

	return ctrl.Result{}, nil
}

func (r *ClusterTemplateClaimReconciler) checkClusterTemplateExist(claim *tmaxiov1.ClusterTemplateClaim) bool {
	ct := &tmaxiov1.ClusterTemplate{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: "",
		Name:      claim.Spec.ResourceName,
	}, ct); err != nil && errors.IsNotFound(err) {
		return false
	}
	return true
}

func (r *ClusterTemplateClaimReconciler) createClusterTemplate(claim *tmaxiov1.ClusterTemplateClaim, template *tmaxiov1.Template) error {
	ct := &tmaxiov1.ClusterTemplate{}
	ct.TypeMeta = metav1.TypeMeta{
		APIVersion: "tmax.io/v1",
		Kind:       "ClusterTemplate",
	}

	ct.ObjectMeta = metav1.ObjectMeta{
		Name:   claim.Spec.ResourceName,
		Labels: map[string]string{claimLabel: claim.Name + "." + claim.Namespace},
	}

	ct.TemplateSpec = template.TemplateSpec

	// Add finalizer to update claim status when ClusterTemplate is deleted
	controllerutil.AddFinalizer(ct, claimFinalizer)

	return r.Client.Create(context.TODO(), ct)
}

func (r *ClusterTemplateClaimReconciler) getTemplateIfExist(claim *tmaxiov1.ClusterTemplateClaim) (bool, *tmaxiov1.Template) {
	template := &tmaxiov1.Template{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: claim.Namespace,
		Name:      claim.Spec.TemplateName,
	}, template); err != nil {
		return false, nil
	}
	return true, template
}

func (r *ClusterTemplateClaimReconciler) updateClusterTemplateClaimStatus(
	claim *tmaxiov1.ClusterTemplateClaim, status *tmaxiov1.ClusterTemplateClaimStatus) (ctrl.Result, error) {
	logger := r.Log.WithName("Update ClusterTemplateClaim status")

	updatedClaim := claim.DeepCopy()
	updatedClaim.Status = *status

	if err := r.Client.Status().Patch(context.TODO(), updatedClaim, client.MergeFrom(claim)); err != nil {
		logger.Error(err, "Error occurs while updating ClusterTemplateClaim status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ClusterTemplateClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmaxiov1.ClusterTemplateClaim{}).
		Complete(r)
}
