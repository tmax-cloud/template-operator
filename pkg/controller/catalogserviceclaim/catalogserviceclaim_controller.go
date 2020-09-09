package catalogserviceclaim

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	tmaxv1 "github.com/jitaeyun/template-operator/pkg/apis/tmax/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_catalogserviceclaim")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new CatalogServiceClaim Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCatalogServiceClaim{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("catalogserviceclaim-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CatalogServiceClaim
	err = c.Watch(&source.Kind{Type: &tmaxv1.CatalogServiceClaim{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner CatalogServiceClaim
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &tmaxv1.CatalogServiceClaim{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileCatalogServiceClaim implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCatalogServiceClaim{}

// ReconcileCatalogServiceClaim reconciles a CatalogServiceClaim object
type ReconcileCatalogServiceClaim struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a CatalogServiceClaim object and makes changes based on the state read
// and what is in the CatalogServiceClaim.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCatalogServiceClaim) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling CatalogServiceClaim")

	// Fetch the CatalogServiceClaim instance
	instance := &tmaxv1.CatalogServiceClaim{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	switch instance.Status.Status {
	case tmaxv1.ClaimSuccess, tmaxv1.ClaimReject, tmaxv1.ClaimError:
	case "": // if status empty
		cscStatus := &tmaxv1.CatalogServiceClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Message:            "wait for admin permission",
			Status:             tmaxv1.ClaimAwating,
		}
		return r.updateCatalogServiceClaimStatus(instance, cscStatus)
	case tmaxv1.ClaimApprove:
		if err = r.createTemplateIfNotExist(&instance.Spec, instance); err != nil {
			cscStatus := &tmaxv1.CatalogServiceClaimStatus{
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Message:            "error occurs while creating cluster template",
				Reason:             err.Error(),
				Status:             tmaxv1.ClaimError,
			}
			return r.updateCatalogServiceClaimStatus(instance, cscStatus)
		}

		cscStatus := &tmaxv1.CatalogServiceClaimStatus{
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Message:            "succeed to create cluster template",
			Status:             tmaxv1.ClaimSuccess,
		}

		return r.updateCatalogServiceClaimStatus(instance, cscStatus)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileCatalogServiceClaim) updateCatalogServiceClaimStatus(
	claim *tmaxv1.CatalogServiceClaim, status *tmaxv1.CatalogServiceClaimStatus) (reconcile.Result, error) {
	reqLogger := log.WithName("update catalog service claim status")

	updatedClaim := claim.DeepCopy()
	updatedClaim.Status = *status

	if err := r.client.Status().Patch(context.TODO(), updatedClaim, client.MergeFrom(claim)); err != nil {
		reqLogger.Error(err, "could not update CatalogServiceClaim status")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileCatalogServiceClaim) createTemplateIfNotExist(
	template *tmaxv1.ClusterTemplate, owner *tmaxv1.CatalogServiceClaim) error {
	foundTemplate := &tmaxv1.ClusterTemplate{}

	// check if it exists
	if err := r.client.Get(context.TODO(), types.NamespacedName{
		Namespace: template.Namespace,
		Name:      template.Name,
	}, foundTemplate); err == nil {
		return errors.NewAlreadyExists(schema.GroupResource{
			Group:    foundTemplate.GroupVersionKind().Group,
			Resource: foundTemplate.Kind}, "resource already exist")
	}

	// set owner reference
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

	// if not exists, create template
	if err := r.client.Create(context.TODO(), template); err != nil {
		return errors.NewInternalError(err)
	}

	return nil
}
