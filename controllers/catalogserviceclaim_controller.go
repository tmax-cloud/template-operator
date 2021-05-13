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
	tmaxiov1 "github.com/tmax-cloud/template-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CatalogServiceClaimReconciler reconciles a CatalogServiceClaim object
type CatalogServiceClaimReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tmax.io,resources=catalogserviceclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=catalogserviceclaims/status,verbs=get;update;patch

func (r *CatalogServiceClaimReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling CatalogServiceClaim")

	// Fetch the CatalogServiceClaim instance
	claim := &tmaxiov1.CatalogServiceClaim{}
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

	if claim.Status.Handled == true {
		reqLogger.Info("already handled claim")
		return ctrl.Result{}, nil
	}

	if r.checkClusterTemplateExist(claim) {
		cscStatus := &tmaxiov1.CatalogServiceClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Message:            fmt.Sprintf("clustertemplate %s already exist", claim.Spec.ResourceName),
			Status:             tmaxiov1.ClaimReject,
			Handled:            true,
		}
		return r.updateCatalogServiceClaimStatus(claim, cscStatus)
	}

	exist, template := r.getTemplateIfExist(claim)
	if !exist {
		cscStatus := &tmaxiov1.CatalogServiceClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Message:            fmt.Sprintf("template %s is not exist", claim.Spec.TemplateName),
			Status:             tmaxiov1.ClaimError,
			Handled:            true,
		}
		return r.updateCatalogServiceClaimStatus(claim, cscStatus)
	}

	switch claim.Status.Status {
	case tmaxiov1.ClaimSuccess, tmaxiov1.ClaimError:
	case "": // if status empty
		cscStatus := &tmaxiov1.CatalogServiceClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Message:            "wait for admin permission",
			Status:             tmaxiov1.ClaimAwating,
			Handled:            false,
		}
		return r.updateCatalogServiceClaimStatus(claim, cscStatus)
	case tmaxiov1.ClaimApprove:
		if err := r.createClusterTemplate(claim, template); err != nil {
			cscStatus := &tmaxiov1.CatalogServiceClaimStatus{
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Message:            "error occurs while creating cluster template",
				Reason:             err.Error(),
				Status:             tmaxiov1.ClaimError,
				Handled:            true,
			}
			return r.updateCatalogServiceClaimStatus(claim, cscStatus)
		}

		cscStatus := &tmaxiov1.CatalogServiceClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Message:            "succeed to create cluster template",
			Status:             tmaxiov1.ClaimSuccess,
			Handled:            true,
		}
		return r.updateCatalogServiceClaimStatus(claim, cscStatus)
	case tmaxiov1.ClaimReject:
		cscStatus := &tmaxiov1.CatalogServiceClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Message:            "reject from admin",
			Status:             tmaxiov1.ClaimReject,
			Handled:            true,
		}
		return r.updateCatalogServiceClaimStatus(claim, cscStatus)
	}

	return ctrl.Result{}, nil
}

func (r *CatalogServiceClaimReconciler) updateCatalogServiceClaimStatus(
	claim *tmaxiov1.CatalogServiceClaim, status *tmaxiov1.CatalogServiceClaimStatus) (ctrl.Result, error) {
	reqLogger := r.Log.WithName("update catalog service claim status")

	updatedClaim := claim.DeepCopy()
	updatedClaim.Status = *status

	if err := r.Client.Status().Patch(context.TODO(), updatedClaim, client.MergeFrom(claim)); err != nil {
		reqLogger.Error(err, "could not update CatalogServiceClaim status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *CatalogServiceClaimReconciler) checkClusterTemplateExist(claim *tmaxiov1.CatalogServiceClaim) bool {
	ct := &tmaxiov1.ClusterTemplate{}
	// check if it exists
	if err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: "",
		Name:      claim.Spec.ResourceName,
	}, ct); err != nil && errors.IsNotFound(err) {
		return false
	}
	return true
}

func (r *CatalogServiceClaimReconciler) getTemplateIfExist(claim *tmaxiov1.CatalogServiceClaim) (bool, *tmaxiov1.Template) {
	template := &tmaxiov1.Template{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: claim.Namespace,
		Name:      claim.Spec.TemplateName,
	}, template); err != nil && errors.IsNotFound(err) {
		return false, nil
	}
	return true, template
}

func (r *CatalogServiceClaimReconciler) createClusterTemplate(claim *tmaxiov1.CatalogServiceClaim, template *tmaxiov1.Template) error {
	ct := &tmaxiov1.ClusterTemplate{}
	ct.TypeMeta = metav1.TypeMeta{
		APIVersion: "tmax.io/v1",
		Kind:       "ClusterTemplate",
	}
	ct.ObjectMeta = metav1.ObjectMeta{
		Name: claim.Spec.ResourceName,
	}
	ct.TemplateSpec = template.TemplateSpec

	return r.Client.Create(context.TODO(), ct)
}

func (r *CatalogServiceClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmaxiov1.CatalogServiceClaim{}).
		Complete(r)
}
