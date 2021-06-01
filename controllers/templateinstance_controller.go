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
	instanceParameters := []tmaxiov1.ParamSpec{}
	instanceWithTemplate := instance.DeepCopy()

	if instance.Spec.ClusterTemplate == nil && instance.Spec.Template == nil {
		err := errors.NewBadRequest("cannot find any template info in instance spec")
		reqLogger.Error(err, "")
		return r.updateTemplateInstanceStatus(instance, err)
	} else if instance.Spec.ClusterTemplate != nil && instance.Spec.Template != nil {
		err := errors.NewBadRequest("you should insert either template or clustertemplate")
		reqLogger.Error(err, "")
		return r.updateTemplateInstanceStatus(instance, err)
	} else if instance.Spec.ClusterTemplate != nil {
		err = r.Client.Get(context.TODO(), types.NamespacedName{
			Name: instance.Spec.ClusterTemplate.Metadata.Name,
		}, refClusterTemplate)
		if err != nil {
			reqLogger.Error(err, "clusterTemplate not found")
			return r.updateTemplateInstanceStatus(instance, err)
		}
		objectInfo.Metadata.Name = instance.Spec.ClusterTemplate.Metadata.Name
		objectInfo.Objects = refClusterTemplate.Objects
		objectInfo.Parameters = refClusterTemplate.Parameters[0:]
		instanceParameters = instance.Spec.ClusterTemplate.Parameters[0:]
		instanceWithTemplate.Spec.ClusterTemplate = objectInfo
	} else {
		err = r.Client.Get(context.TODO(), types.NamespacedName{
			Namespace: instance.Namespace,
			Name:      instance.Spec.Template.Metadata.Name,
		}, refTemplate)
		if err != nil {
			reqLogger.Error(err, "template not found")
			return r.updateTemplateInstanceStatus(instance, err)
		}
		objectInfo.Metadata.Name = instance.Spec.Template.Metadata.Name
		objectInfo.Objects = refTemplate.Objects
		objectInfo.Parameters = refTemplate.Parameters[0:]
		instanceParameters = instance.Spec.Template.Parameters[0:]
		instanceWithTemplate.Spec.Template = objectInfo
	}

	//parameter map
	params := make(map[string]intstr.IntOrString)

	// make instance parameter map
	for _, param := range instanceParameters {
		params[param.Name] = param.Value
	}

	// make real parameter with instance and default parameter
	for idx, param := range objectInfo.Parameters {
		if val, ok := params[param.Name]; ok {
			// if a instance param was given, change instance param value
			objectInfo.Parameters[idx].Value = val
		} else if param.Required || param.Value.Size() == 0 {
			// if param not found && (the param was required or default value was not set)
			err := errors.NewBadRequest("parameter: " + param.Name + " must be included")
			reqLogger.Error(err, "error occurs while setting parameters")
			return r.updateTemplateInstanceStatus(instance, err)
		}
		if (len(objectInfo.Parameters[idx].ValueType) == 0 || objectInfo.Parameters[idx].ValueType == "string") && objectInfo.Parameters[idx].Value.Type == 0 {
			if objectInfo.Parameters[idx].Value.IntValue() == 0 {
				objectInfo.Parameters[idx].Value = intstr.IntOrString{Type: 1, IntVal: 0, StrVal: ""}
			}
		}
		//set param value
		params[param.Name] = objectInfo.Parameters[idx].Value
	}

	//replace parameter name to value in object and check exist k8s object
	for i := range objectInfo.Objects {
		if err = r.replaceParamsWithValue(&(objectInfo.Objects[i]), &params); err != nil {
			reqLogger.Error(err, "error occurs while replacing parameters")
			return r.updateTemplateInstanceStatus(instance, err)
		}
		if err = r.existK8sObject(&(objectInfo.Objects[i]), instance); err != nil {
			reqLogger.Error(err, "exist resource")
			return r.updateTemplateInstanceStatus(instance, err)
		}
	}

	//create k8s object
	for i := range objectInfo.Objects {
		if err = r.createK8sObject(&(objectInfo.Objects[i]), instance); err != nil {
			reqLogger.Error(err, "error occurs while create k8s object")
			return r.updateTemplateInstanceStatus(instance, err)
		}
	}

	// finally, update template instance
	res, e := r.updateTemplateInstanceStatus(instanceWithTemplate, nil)
	if err = r.Client.Patch(context.TODO(), instanceWithTemplate, client.MergeFrom(instance)); err != nil {
		reqLogger.Error(err, "could not update template instance")
		return r.updateTemplateInstanceStatus(instance, err)
	}
	return res, e
}

func (r *TemplateInstanceReconciler) replaceParamsWithValue(obj *runtime.RawExtension, params *map[string]intstr.IntOrString) error {
	reqLogger := r.Log.WithName("replace k8s object")
	objYamlStr := string(obj.Raw)
	reqLogger.Info("original object: " + objYamlStr)
	for key, value := range *params {
		reqLogger.Info("key: " + key + " value: " + value.String())
		if value.Type == 0 {
			objYamlStr = strings.Replace(objYamlStr, "\"${"+key+"}\"", value.String(), -1)
		}
		objYamlStr = strings.Replace(objYamlStr, "${"+key+"}", value.String(), -1)
	}
	reqLogger.Info("replaced object: " + objYamlStr)

	obj.Raw = []byte(objYamlStr)
	return nil
}

func (r *TemplateInstanceReconciler) existK8sObject(obj *runtime.RawExtension, owner *tmaxiov1.TemplateInstance) error {
	unstr, err := internal.BytesToUnstructuredObject(obj)
	if err != nil {
		return err
	}
	unstr.SetNamespace(owner.Namespace)
	// check if the object already exist
	check := unstr.DeepCopy()
	if err = r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: check.GetNamespace(),
		Name:      check.GetName(),
	}, check); err == nil {
		return errors.NewAlreadyExists(schema.GroupResource{
			Group:    check.GroupVersionKind().Group,
			Resource: check.GetKind()}, "namespace: "+check.GetNamespace()+" name: "+check.GetName())
	}
	return nil
}

func (r *TemplateInstanceReconciler) createK8sObject(obj *runtime.RawExtension, owner *tmaxiov1.TemplateInstance) error {
	//reqLogger := r.Log.WithName("replace createK8sObject")
	// get unstructured object
	unstr, err := internal.BytesToUnstructuredObject(obj)
	if err != nil {
		return err
	}

	// set namespace if not exist
	//if len(unstr.GetNamespace()) == 0 {
	unstr.SetNamespace(owner.Namespace)
	//}

	// set owner reference
	isController := false
	blockOwnerDeletion := true

	//Get 하고 추가
	ownerRefs := unstr.GetOwnerReferences()
	//reqLogger.Info("before: " + fmt.Sprintf("%+v\n", unstr.GetOwnerReferences()))
	ownerRef := v1.OwnerReference{
		APIVersion:         owner.APIVersion,
		Kind:               owner.Kind,
		Name:               owner.Name,
		UID:                owner.UID,
		Controller:         &isController,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
	ownerRefs = append(ownerRefs, ownerRef)
	unstr.SetOwnerReferences(ownerRefs)
	//reqLogger.Info("after: " + fmt.Sprintf("%+v\n", unstr.GetOwnerReferences()))
	// create object
	if err = r.Client.Create(context.TODO(), unstr); err != nil {
		return err
	}

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
