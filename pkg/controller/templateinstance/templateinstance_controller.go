package templateinstance

import (
	"context"
	"fmt"
	"strings"

	"github.com/jwkim1993/hypercloud-operator/internal"
	tmaxv1 "github.com/jwkim1993/hypercloud-operator/pkg/apis/tmax/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

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

	// if status field is not nil, end reconcile
	if len(instance.Status.Conditions) != 0 {
		reqLogger.Info("already handled instance")
		return reconcile.Result{}, nil
	}

	// Get the template it refers

	refTemplate := &tmaxv1.Template{}
	refClusterTemplate := &tmaxv1.ClusterTemplate{}
	var objects []runtime.RawExtension
	var parameters []tmaxv1.ParamSpec

	if instance.Spec.ClusterTemplate.Name == "" && instance.Spec.Template.Name == "" {
		err := errors.NewBadRequest("cannot find any template info in instance spec")
		reqLogger.Error(err, "")
		return r.updateTemplateInstanceStatus(instance, err)
	} else if instance.Spec.ClusterTemplate.Name != "" && instance.Spec.Template.Name != "" {
		err := errors.NewBadRequest("you should insert either template or clustertemplate")
		reqLogger.Error(err, "")
		return r.updateTemplateInstanceStatus(instance, err)
	} else if instance.Spec.ClusterTemplate.Name != "" {
		err = r.client.Get(context.TODO(), types.NamespacedName{
			Name: instance.Spec.ClusterTemplate.Name,
		}, refClusterTemplate)
		if err != nil {
			reqLogger.Error(err, "clusterTemplate not found")
			return r.updateTemplateInstanceStatus(instance, err)
		}
		objects = refClusterTemplate.Objects[0:]
		refClusterTemplate.Parameters = instance.Spec.ClusterTemplate.Parameters
		parameters = refClusterTemplate.Parameters[0:]
	} else {
		err = r.client.Get(context.TODO(), types.NamespacedName{
			Namespace: instance.Namespace,
			Name:      instance.Spec.Template.Name,
		}, refTemplate)
		if err != nil {
			reqLogger.Error(err, "template not found")
			return r.updateTemplateInstanceStatus(instance, err)
		}
		objects = refTemplate.Objects[0:]
		refTemplate.Parameters = instance.Spec.Template.Parameters
		parameters = refTemplate.Parameters[0:]
	}

	// make parameter map
	params := make(map[string]intstr.IntOrString)
	for _, param := range parameters {
		params[param.Name] = param.Value
	}

	for i := range objects {
		if err = r.replaceParamsWithValue(&(objects[i]), &params); err != nil {
			reqLogger.Error(err, "error occurs while replacing parameters")
			return r.updateTemplateInstanceStatus(instance, err)
		}

		if err = r.createK8sObject(&(objects[i]), instance); err != nil && !errors.IsAlreadyExists(err) {
			reqLogger.Error(err, "error occurs while create k8s object")
			return r.updateTemplateInstanceStatus(instance, err)
		}
	}

	// finally, update template instance
	instanceWithTemplate := instance.DeepCopy()
	instanceWithTemplate.Spec.Template = *refTemplate
	instanceWithTemplate.Spec.ClusterTemplate = *refClusterTemplate
	if err = r.client.Patch(context.TODO(), instanceWithTemplate, client.MergeFrom(instance)); err != nil {
		reqLogger.Error(err, "could not update template instance")
		return r.updateTemplateInstanceStatus(instance, err)
	}
	return r.updateTemplateInstanceStatus(instanceWithTemplate, nil)
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

func (r *ReconcileTemplateInstance) createK8sObject(obj *runtime.RawExtension, owner *tmaxv1.TemplateInstance) error {
	// get unstructured object
	unstr, err := internal.BytesToUnstructuredObject(obj)
	if err != nil {
		return err
	}

	// set namespace if not exist
	if len(unstr.GetNamespace()) == 0 {
		unstr.SetNamespace(owner.Namespace)
	}

	// check if the object already exist
	check := unstr.DeepCopy()
	if err = r.client.Get(context.TODO(), types.NamespacedName{
		Namespace: check.GetNamespace(),
		Name:      check.GetName(),
	}, check); err == nil {
		return errors.NewAlreadyExists(schema.GroupResource{
			Group:    check.GroupVersionKind().Group,
			Resource: check.GetKind()}, "resource already exist")
	}

	// set owner reference
	isController := true
	blockOwnerDeletion := true
	ownerRef := []v1.OwnerReference{
		{
			APIVersion:         owner.APIVersion,
			Kind:               owner.Kind,
			Name:               owner.Name,
			UID:                owner.UID,
			Controller:         &isController,
			BlockOwnerDeletion: &blockOwnerDeletion,
		},
	}
	unstr.SetOwnerReferences(ownerRef)

	// create object
	if err = r.client.Create(context.TODO(), unstr); err != nil {
		return err
	}

	return nil
}

func (r *ReconcileTemplateInstance) updateTemplateInstanceStatus(instance *tmaxv1.TemplateInstance, err error) (reconcile.Result, error) {
	reqLogger := log.WithName("update template instance status")
	// set condition depending on the error
	instanceWithStatus := instance.DeepCopy()

	var cond tmaxv1.ConditionSpec
	if err == nil {
		cond.Message = "succeed to create instances"
		cond.Status = "Success"
	} else {
		cond.Message = err.Error()
		cond.Reason = "error occurs while create instance"
		cond.Status = "Error"
	}

	// set status
	instanceWithStatus.Status = tmaxv1.TemplateInstanceStatus{
		Conditions: []tmaxv1.ConditionSpec{
			cond,
		},
		Objects: nil,
	}

	if errUp := r.client.Status().Patch(context.TODO(), instanceWithStatus, client.MergeFrom(instance)); errUp != nil {
		reqLogger.Error(errUp, "could not update template instance")
		return reconcile.Result{}, errUp
	}

	reqLogger.Info("succeed to create template instance status")
	return reconcile.Result{}, err
}
