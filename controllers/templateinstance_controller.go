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
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmaxiov1 "github.com/tmax-cloud/template-operator/api/v1"
	"github.com/tmax-cloud/template-operator/internal"
)

// TemplateInstanceReconciler reconciles a TemplateInstance object
type TemplateInstanceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tmax.io,resources=templateinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=templateinstances/status,verbs=get;update;patch

func (r *TemplateInstanceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling TemplateInstance")

	// Fetch the TemplateInstance instance
	instance := &tmaxiov1.TemplateInstance{}
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

	// if status field is not nil, end reconcile
	if len(instance.Status.Conditions) != 0 {
		reqLogger.Info("already handled instance")
		return ctrl.Result{}, nil
	}

	// Get the template it refers
	refTemplate := &tmaxiov1.Template{}
	refClusterTemplate := &tmaxiov1.ClusterTemplate{}
	objectInfo := &tmaxiov1.ObjectInfo{}
	instanceWithTemplate := instance.DeepCopy()

	if instance.Spec.ClusterTemplate.Name == "" && instance.Spec.Template.Name == "" {
		err := errors.NewBadRequest("cannot find any template info in instance spec")
		reqLogger.Error(err, "")
		return r.updateTemplateInstanceStatus(instance, err)
	} else if instance.Spec.ClusterTemplate.Name != "" && instance.Spec.Template.Name != "" {
		err := errors.NewBadRequest("you should insert either template or clustertemplate")
		reqLogger.Error(err, "")
		return r.updateTemplateInstanceStatus(instance, err)
	} else if instance.Spec.ClusterTemplate.Name != "" {
		err = r.Client.Get(context.TODO(), types.NamespacedName{
			Name: instance.Spec.ClusterTemplate.Name,
		}, refClusterTemplate)
		if err != nil {
			reqLogger.Error(err, "clusterTemplate not found")
			return r.updateTemplateInstanceStatus(instance, err)
		}
		objectInfo.ObjectMeta = instance.Spec.ClusterTemplate.ObjectMeta
		objectInfo.Objects = refClusterTemplate.Objects
		objectInfo.Parameters = instance.Spec.ClusterTemplate.Parameters[0:]
		instanceWithTemplate.Spec.ClusterTemplate = *objectInfo
	} else {
		err = r.Client.Get(context.TODO(), types.NamespacedName{
			Namespace: instance.Namespace,
			Name:      instance.Spec.Template.Name,
		}, refTemplate)
		if err != nil {
			reqLogger.Error(err, "template not found")
			return r.updateTemplateInstanceStatus(instance, err)
		}
		objectInfo.ObjectMeta = instance.Spec.Template.ObjectMeta
		objectInfo.Objects = refTemplate.Objects
		objectInfo.Parameters = instance.Spec.Template.Parameters[0:]
		instanceWithTemplate.Spec.Template = *objectInfo
	}

	// make parameter map
	params := make(map[string]intstr.IntOrString)
	for _, param := range objectInfo.Parameters {
		params[param.Name] = param.Value
	}

	for i := range objectInfo.Objects {
		if err = r.replaceParamsWithValue(&(objectInfo.Objects[i]), &params); err != nil {
			reqLogger.Error(err, "error occurs while replacing parameters")
			return r.updateTemplateInstanceStatus(instance, err)
		}

		if err = r.createK8sObject(&(objectInfo.Objects[i]), instance); err != nil && !errors.IsAlreadyExists(err) {
			reqLogger.Error(err, "error occurs while create k8s object")
			return r.updateTemplateInstanceStatus(instance, err)
		}
	}

	// finally, update template instance
	if err = r.Client.Patch(context.TODO(), instanceWithTemplate, client.MergeFrom(instance)); err != nil {
		reqLogger.Error(err, "could not update template instance")
		return r.updateTemplateInstanceStatus(instance, err)
	}
	return r.updateTemplateInstanceStatus(instanceWithTemplate, nil)
}

func (r *TemplateInstanceReconciler) replaceParamsWithValue(obj *runtime.RawExtension, params *map[string]intstr.IntOrString) error {
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

func (r *TemplateInstanceReconciler) createK8sObject(obj *runtime.RawExtension, owner *tmaxiov1.TemplateInstance) error {
	reqLogger := r.Log.WithName("create k8s object")
	// get unstructured object
	unstr, err := internal.BytesToUnstructuredObject(obj)
	if err != nil {
		return err
	}

	// set namespace if not exist
	//if len(unstr.GetNamespace()) == 0 {
	unstr.SetNamespace(owner.Namespace)
	//}

	// check if the object already exist
	check := unstr.DeepCopy()
	if err = r.Client.Get(context.TODO(), types.NamespacedName{
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
	if err = r.Client.Create(context.TODO(), unstr); err != nil {
		return err
	}
	reqLogger.Info("Group: " + check.GroupVersionKind().Group + " kind: " + check.GetKind() + " Name: " + check.GetName() + " Namespace: " + check.GetNamespace())
	return nil
}

func (r *TemplateInstanceReconciler) updateTemplateInstanceStatus(instance *tmaxiov1.TemplateInstance, err error) (ctrl.Result, error) {
	reqLogger := r.Log.WithName("update template instance status")
	// set condition depending on the error
	instanceWithStatus := instance.DeepCopy()

	var cond tmaxiov1.ConditionSpec
	if err == nil {
		cond.Message = "succeed to create instances"
		cond.Status = "Success"
	} else {
		cond.Message = err.Error()
		cond.Reason = "error occurs while create instance"
		cond.Status = "Error"
	}

	// set status
	instanceWithStatus.Status = tmaxiov1.TemplateInstanceStatus{
		Conditions: []tmaxiov1.ConditionSpec{
			cond,
		},
		Objects: nil,
	}

	if errUp := r.Client.Status().Patch(context.TODO(), instanceWithStatus, client.MergeFrom(instance)); errUp != nil {
		reqLogger.Error(errUp, "could not update template instance")
		return ctrl.Result{}, errUp
	}

	reqLogger.Info("succeed to create template instance status")
	return ctrl.Result{}, err
}

func (r *TemplateInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmaxiov1.TemplateInstance{}).
		Complete(r)
}
