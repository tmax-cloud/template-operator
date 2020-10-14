package template

import (
	"context"
	"fmt"

	tmaxv1 "github.com/tmax-cloud/template-operator/pkg/apis/tmax/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_template")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Template Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileTemplate{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("template-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Template
	err = c.Watch(&source.Kind{Type: &tmaxv1.Template{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Template
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &tmaxv1.Template{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileTemplate implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileTemplate{}

// ReconcileTemplate reconciles a Template object
type ReconcileTemplate struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Template object and makes changes based on the state read
// and what is in the Template.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileTemplate) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Template")

	// Fetch the Template instance
	instance := &tmaxv1.Template{}
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

	// if status field is not nil, end reconcile
	if len(instance.Status.Status) != 0 {
		reqLogger.Info("already handled template")
		return reconcile.Result{}, nil
	}

	updateInstance := instance.DeepCopy()

	// add kind to objectKinds fields
	objectKinds := make([]string, 0)
	for _, obj := range instance.Objects {
		var in runtime.Object
		var scope conversion.Scope // While not actually used within the function, need to pass in
		if err = runtime.Convert_runtime_RawExtension_To_runtime_Object(&obj, &in, scope); err != nil {
			reqLogger.Error(err, "cannot decode object")
			templateStatus := &tmaxv1.TemplateStatus{
				Message: "cannot decode object",
				Status:  tmaxv1.TemplateError,
			}
			return r.updateTemplateStatus(instance, templateStatus)
		}

		unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(in)
		if err != nil {
			reqLogger.Error(err, "cannot decode object")
			templateStatus := &tmaxv1.TemplateStatus{
				Message: "cannot decode object",
				Status:  tmaxv1.TemplateError,
			}
			return r.updateTemplateStatus(instance, templateStatus)
		}

		unstr := unstructured.Unstructured{Object: unstrObj}
		reqLogger.Info(fmt.Sprintf("kind: %s", unstr.GetKind()))
		objectKinds = append(objectKinds, unstr.GetKind())
	}
	updateInstance.ObjectKinds = objectKinds
	reqLogger.Info(fmt.Sprintf("%v", objectKinds))

	if err = r.client.Patch(context.TODO(), updateInstance, client.MergeFrom(instance)); err != nil {
		reqLogger.Error(err, "cannot update template")
		templateStatus := &tmaxv1.TemplateStatus{
			Message: "cannot update template",
			Status:  tmaxv1.TemplateError,
		}
		return r.updateTemplateStatus(instance, templateStatus)
	}

	templateStatus := &tmaxv1.TemplateStatus{
		Message: "update success",
		Status:  tmaxv1.TemplateSuccess,
	}
	return r.updateTemplateStatus(instance, templateStatus)
}

func (r *ReconcileTemplate) updateTemplateStatus(
	template *tmaxv1.Template, status *tmaxv1.TemplateStatus) (reconcile.Result, error) {
	reqLogger := log.WithName("update template status")

	updatedTemplate := template.DeepCopy()
	updatedTemplate.Status = *status

	if err := r.client.Status().Patch(context.TODO(), updatedTemplate, client.MergeFrom(template)); err != nil {
		reqLogger.Error(err, "could not update Template status")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
