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

	tmplv1 "github.com/tmax-cloud/template-operator/api/v1"
)

// ClusterTemplateClaimReconciler reconciles a ClusterTemplateClaim object
type ClusterTemplateClaimReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const (
	claimFinalizer = "clustertemplateclaims.tmax.io/finalizer"
	claimLabel     = "clustertemplateclaims.tmax.io/claim"
)

// +kubebuilder:rbac:groups=tmax.io,resources=clustertemplateclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=clustertemplateclaims/status,verbs=get;update;patch

func (r *ClusterTemplateClaimReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("clustertemplateclaim", req.NamespacedName)
	logger.Info("Reconciling ClusterTemplateClaim")

	// Get the ClusterTemplateClaim instance
	claim := &tmplv1.ClusterTemplateClaim{}
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
		status := &tmplv1.ClusterTemplateClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             fmt.Sprintf("ClusterTemplate %s already exist", claim.Spec.ResourceName),
			Status:             tmplv1.Rejected,
			Handled:            false,
		}
		return r.updateClusterTemplateClaimStatus(claim, status)
	}

	exist, template := r.getTemplateIfExist(claim)
	if !exist {
		status := &tmplv1.ClusterTemplateClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             fmt.Sprintf("Fail to get template %s", claim.Spec.TemplateName),
			Status:             tmplv1.Error,
			Handled:            false,
		}
		return r.updateClusterTemplateClaimStatus(claim, status)
	}

	switch claim.Status.Status {
	case "":
		status := &tmplv1.ClusterTemplateClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             "Waiting for admin permission",
			Status:             tmplv1.Awating,
			Handled:            false,
		}
		return r.updateClusterTemplateClaimStatus(claim, status)
	case tmplv1.Approved:
		if err := r.createClusterTemplate(claim, template); err != nil {
			status := &tmplv1.ClusterTemplateClaimStatus{
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             "Error occurs while creating cluster template",
				Status:             tmplv1.Error,
				Handled:            false,
			}
			return r.updateClusterTemplateClaimStatus(claim, status)
		}

		status := &tmplv1.ClusterTemplateClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             "Succeed to create cluster template",
			Status:             tmplv1.Approved,
			Handled:            true,
		}
		return r.updateClusterTemplateClaimStatus(claim, status)
	case tmplv1.Rejected:
		rejectReason := claim.Status.Reason
		if len(rejectReason) == 0 {
			rejectReason = "Rejected by admin"
		}
		status := &tmplv1.ClusterTemplateClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             rejectReason,
			Status:             tmplv1.Rejected,
			Handled:            false,
		}
		return r.updateClusterTemplateClaimStatus(claim, status)
	}

	return ctrl.Result{}, nil
}

func (r *ClusterTemplateClaimReconciler) checkClusterTemplateExist(claim *tmplv1.ClusterTemplateClaim) bool {
	ct := &tmplv1.ClusterTemplate{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: "",
		Name:      claim.Spec.ResourceName,
	}, ct); err != nil && errors.IsNotFound(err) {
		return false
	}
	return true
}

func (r *ClusterTemplateClaimReconciler) createClusterTemplate(claim *tmplv1.ClusterTemplateClaim, template *tmplv1.Template) error {
	ct := &tmplv1.ClusterTemplate{}
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

func (r *ClusterTemplateClaimReconciler) getTemplateIfExist(claim *tmplv1.ClusterTemplateClaim) (bool, *tmplv1.Template) {
	template := &tmplv1.Template{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: claim.Namespace,
		Name:      claim.Spec.TemplateName,
	}, template); err != nil {
		return false, nil
	}
	return true, template
}

func (r *ClusterTemplateClaimReconciler) updateClusterTemplateClaimStatus(
	claim *tmplv1.ClusterTemplateClaim, status *tmplv1.ClusterTemplateClaimStatus) (ctrl.Result, error) {
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
		For(&tmplv1.ClusterTemplateClaim{}).
		Complete(r)
}
