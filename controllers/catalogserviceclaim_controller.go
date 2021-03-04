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
	"time"

	"github.com/go-logr/logr"
	tmaxiov1 "github.com/tmax-cloud/template-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	instance := &tmaxiov1.CatalogServiceClaim{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	if instance.Status.Handled == true {
		reqLogger.Info("already handled claim")
		return ctrl.Result{}, nil
	}

	switch instance.Status.Status {
	case tmaxiov1.ClaimSuccess, tmaxiov1.ClaimError:
	case "": // if status empty
		cscStatus := &tmaxiov1.CatalogServiceClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Message:            "wait for admin permission",
			Status:             tmaxiov1.ClaimAwating,
			Handled:            false,
		}
		return r.updateCatalogServiceClaimStatus(instance, cscStatus)
	case tmaxiov1.ClaimApprove:
		ct := &tmaxiov1.ClusterTemplate{}
		ct.TypeMeta = metav1.TypeMeta{
			APIVersion: "tmax.io/v1",
			Kind:       "ClusterTemplate",
		}
		ct.ObjectMeta = instance.Spec.ObjectMeta
		ct.TemplateSpec = instance.Spec.TemplateSpec
		if err = r.createTemplateIfNotExist(ct, instance); err != nil {
			cscStatus := &tmaxiov1.CatalogServiceClaimStatus{
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Message:            "error occurs while creating cluster template",
				Reason:             err.Error(),
				Status:             tmaxiov1.ClaimError,
				Handled:            true,
			}
			return r.updateCatalogServiceClaimStatus(instance, cscStatus)
		}

		cscStatus := &tmaxiov1.CatalogServiceClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Message:            "succeed to create cluster template",
			Status:             tmaxiov1.ClaimSuccess,
			Handled:            true,
		}

		return r.updateCatalogServiceClaimStatus(instance, cscStatus)
	case tmaxiov1.ClaimReject:
		cscStatus := &tmaxiov1.CatalogServiceClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Message:            "reject from admin",
			Status:             tmaxiov1.ClaimReject,
			Handled:            true,
		}
		return r.updateCatalogServiceClaimStatus(instance, cscStatus)
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

func (r *CatalogServiceClaimReconciler) createTemplateIfNotExist(
	template *tmaxiov1.ClusterTemplate, owner *tmaxiov1.CatalogServiceClaim) error {
	foundTemplate := &tmaxiov1.ClusterTemplate{}

	// check if it exists
	if err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: template.Namespace,
		Name:      template.Name,
	}, foundTemplate); err == nil {
		return errors.NewAlreadyExists(schema.GroupResource{
			Group:    foundTemplate.GroupVersionKind().Group,
			Resource: foundTemplate.Kind}, "resource already exist")
	}

	// set owner reference
	/*
		isController := true
		blockOwnerDeletion := true
		ownerRef := []metav1.OwnerReference{
			{
				APIVersion:         owner.APIVersion,
				Kind:               owner.Kind,
				Name:               owner.Name,
				UID:                owner.UID,
				Controller:         &isController,
				BlockOwnerDeletion: &blockOwnerDeletion,
			},
		}
		template.SetOwnerReferences(ownerRef)
	*/

	// if not exists, create template
	if err := r.Client.Create(context.TODO(), template); err != nil {
		return errors.NewInternalError(err)
	}

	return nil
}

func (r *CatalogServiceClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmaxiov1.CatalogServiceClaim{}).
		Complete(r)
}
