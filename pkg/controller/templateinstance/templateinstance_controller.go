package templateinstance

import (
	"context"
	"fmt"
	"github.com/jwkim1993/hypercloud-operator/internal"
	tmaxv1 "github.com/jwkim1993/hypercloud-operator/pkg/apis/tmax/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_templateinstance")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new TemplateInstance Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileTemplateInstance{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("templateinstance-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource TemplateInstance
	err = c.Watch(&source.Kind{Type: &tmaxv1.TemplateInstance{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner TemplateInstance
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &tmaxv1.TemplateInstance{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileTemplateInstance implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileTemplateInstance{}

// ReconcileTemplateInstance reconciles a TemplateInstance object
type ReconcileTemplateInstance struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a TemplateInstance object and makes changes based on the state read
// and what is in the TemplateInstance.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileTemplateInstance) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling TemplateInstance")

	// Fetch the TemplateInstance instance
	instance := &tmaxv1.TemplateInstance{}
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

	// Get the template it refers
	refTemplate := &tmaxv1.Template{}
	err = r.client.Get(context.TODO(), types.NamespacedName{
		Namespace: request.Namespace,
		Name:      instance.Spec.Template.Name,
	}, refTemplate)
	if err != nil {
		reqLogger.Error(err, "template not found")
		return reconcile.Result{}, err
	}

	// make parameter map
	params := make(map[string]intstr.IntOrString)
	for _, param := range instance.Spec.Template.Parameters {
		params[param.Name] = param.Value
	}

	for i := range refTemplate.Objects {
		if err = r.replaceParamsWithValue(&(refTemplate.Objects[i]), &params); err != nil {
			reqLogger.Error(err, "error occurs while replacing parameters")
			return reconcile.Result{}, err
		}

		if err = r.createK8sObject(&(refTemplate.Objects[i]), refTemplate.Namespace); err != nil {
			reqLogger.Error(err, "error occurs while create k8s object")
			return reconcile.Result{}, err
		}
	}

	// finally, update template instance
	instance.Spec.Template = *refTemplate
	if err = r.client.Update(context.TODO(), instance); err != nil {
		reqLogger.Error(err, "error occurs while update template instance")
		return reconcile.Result{}, err
	}

	reqLogger.Info("succeed to create all resources")
	return reconcile.Result{}, nil
}

func (r *ReconcileTemplateInstance) replaceParamsWithValue(obj *runtime.RawExtension, params *map[string]intstr.IntOrString) error {
	objYamlStr := string(obj.Raw)
	for key, value := range *params {
		objYamlStr = strings.Replace(objYamlStr, fmt.Sprintf("${%s}", key), value.String(), -1)
	}

	if strings.Index(objYamlStr, "${") != -1 {
		return errors.NewBadRequest("not enough parameters exist")
	}

	obj.Raw = []byte(objYamlStr)
	return nil
}

func (r *ReconcileTemplateInstance) createK8sObject(obj *runtime.RawExtension, ns string) error {
	// get unstructured object
	unstr, err := internal.BytesToUnstructuredObject(obj)
	if err != nil {
		return err
	}

	// set namespace if not exist
	if len(unstr.GetNamespace()) == 0 {
		unstr.SetNamespace(ns)
	}

	// create object
	if err = r.client.Create(context.TODO(), unstr); err != nil {
		return err
	}

	return nil
}
