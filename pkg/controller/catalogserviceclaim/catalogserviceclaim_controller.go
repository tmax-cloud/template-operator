package catalogserviceclaim

import (
	"context"

	tmaxv1 "github.com/youngind/hypercloud-operator/pkg/apis/tmax/v1"

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

	crdapi "github.com/kubernetes-client/go/kubernetes/client"
	"github.com/kubernetes-client/go/kubernetes/config"
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

	Event := request.Event
	claimName := request.Name
	claimNamespace := request.Namespace
	resourceName := instance.Spec.Metadata.Name
	catalogNamespace := "default"

	switch Event {
	case "Create": // case 1: wait for admin permission -> pathStatus ( Event == Create)
		instance.Status.Status = "Awaiting"
		instance.Status.Reason = "wait for admin permission"

		if err = r.client.Status().Update(context.TODO(), instance); err != nil {
			return reconcile.Result{}, err
		}
	case "Update":
		status := instance.Status.Status

		if status == "Success" && templateAlreadyExist(resourceName, catalogNamespace) {
			// case 2: approved by admin and template update -> patchStatus ( Event == Update && template cr != null)
			instance.Status.Status = "Success"
			instance.Status.Reason = "template update success."

			if err = r.client.Status().Update(context.TODO(), instance); err != nil {
				return reconcile.Result{}, err
			}
		} else if status == "Success" && !templateAlreadyExist(resourceName, catalogNamespace) {
			// case 3: approved by admin and template create -> createCustomObject & patchStatus ( Event == Update && template cr = null)
			err := createTemplate(instance, claimName, claimNamespace)

			if err != nil {
				panic("===[ Template Create Error ] : " + err.Error())
			}
			instance.Status.Status = "Success"
			instance.Status.Reason = "template create success."

			if err = r.client.Status().Update(context.TODO(), instance); err != nil {
				return reconcile.Result{}, err
			}
		}

	case "Delete":
		// Nothing to do
	}
	return reconcile.Result{}, nil
}

func templateAlreadyExist(name string, namespace string) bool {

	c, err := config.LoadKubeConfig()
	if err != nil {
		panic("===[ Error ] : " + err.Error())
	}
	clientset := crdapi.NewAPIClient(c)

	cr, _, err := clientset.CustomObjectsApi.GetNamespacedCustomObject(context.Background(), "tmax.io", "v1", namespace, "templates", name)
	if err != nil {
		return false
	} else if cr == nil {
		return false
	}
	return true
}

func createTemplate(claim *tmaxv1.CatalogServiceClaim, name string, namespace string) error {

	c, err := config.LoadKubeConfig()
	if err != nil {
		panic("===[ Error ] : " + err.Error())
	}
	clientset := crdapi.NewAPIClient(c)

	var bodyObj interface{}
	bodyObj = claim.Spec

	response, _, err := clientset.CustomObjectsApi.CreateNamespacedCustomObject(context.Background(), "tmax.io", "v1", namespace, "templates", bodyObj, nil)

	if err != nil && response == nil {
		if errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
