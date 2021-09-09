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
	"encoding/json"
	"reflect"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	tmaxiov1 "github.com/tmax-cloud/template-operator/api/v1"
	"github.com/tmax-cloud/template-operator/internal"
)

// TemplateInstanceReconciler reconciles a TemplateInstance object
type TemplateInstanceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const (
	stringType = "string"
	numberType = "number"
)

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
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// template/clustertemplate both empty or inserted
	if (instance.Spec.ClusterTemplate == nil) == (instance.Spec.Template == nil) {
		err := errors.NewBadRequest("You should insert either template or clustertemplate")
		reqLogger.Error(err, "Error occurs while get template info")
		return r.updateTemplateInstanceStatus(instance, err)
	}

	objectInfo := &tmaxiov1.ObjectInfo{}
	instanceParameters := []tmaxiov1.ParamSpec{}
	updateInstance := instance.DeepCopy()

	if instance.Spec.ClusterTemplate != nil { // instance with clustertemplate
		instanceParameters = instance.Spec.ClusterTemplate.Parameters
		if updateInstance.Status.ClusterTemplate == nil { // initial apply of instance
			updateInstance.Status.ClusterTemplate = objectInfo
			// Get the clustertemplate info
			template := &tmaxiov1.ClusterTemplate{}
			if err = r.Client.Get(context.TODO(), types.NamespacedName{
				Name: instance.Spec.ClusterTemplate.Metadata.Name,
			}, template); err != nil {
				reqLogger.Error(err, "Error occurs while get clustertemplate")
				return r.updateTemplateInstanceStatus(instance, err)
			}

			objectInfo.Metadata.Name = instance.Spec.ClusterTemplate.Metadata.Name
			objectInfo.Objects = template.Objects
			objectInfo.Parameters = template.Parameters

		} else {
			objectInfo = updateInstance.Status.ClusterTemplate
		}
	}
	if instance.Spec.Template != nil { // instance with template
		instanceParameters = instance.Spec.Template.Parameters
		if updateInstance.Status.Template == nil { // initial apply of instance
			updateInstance.Status.Template = objectInfo

			// Get the template info
			template := &tmaxiov1.Template{}
			if err = r.Client.Get(context.TODO(), types.NamespacedName{
				Namespace: instance.Namespace,
				Name:      instance.Spec.Template.Metadata.Name,
			}, template); err != nil {
				reqLogger.Error(err, "Error occurs while get template")
				return r.updateTemplateInstanceStatus(instance, err)
			}

			objectInfo.Metadata.Name = instance.Spec.Template.Metadata.Name
			objectInfo.Objects = template.Objects
			objectInfo.Parameters = template.Parameters

		} else {
			objectInfo = updateInstance.Status.Template
		}
	}

	tempObjectInfo := objectInfo.DeepCopy()

	// make instance parameter map
	instanceParams := make(map[string]intstr.IntOrString)
	for _, param := range instanceParameters {
		instanceParams[param.Name] = param.Value
	}

	// make real parameter with instance and default parameter
	for idx, param := range tempObjectInfo.Parameters {
		// reflect a given instance parameter
		if val, exist := instanceParams[param.Name]; exist {
			convertedVal := val
			if param.ValueType == numberType && val.Type == intstr.String {
				convertedVal = intstr.IntOrString{Type: intstr.Int, IntVal: int32(val.IntValue())}
			}
			if param.ValueType == stringType && val.Type == intstr.Int {
				convertedVal = intstr.IntOrString{Type: intstr.String, StrVal: val.String()}
			}
			param.Value = convertedVal
		}
		// If the required field has no value
		if param.Required && param.Value.Size() == 0 {
			err := errors.NewBadRequest(param.Name + "must have a value")

			reqLogger.Error(err, "Required parameter has no value")
			return r.updateTemplateInstanceStatus(instance, err)
		}

		// Set default value for not required parameter
		if param.Value.Size() == 0 {
			if len(param.ValueType) == 0 || param.ValueType == stringType {
				param.Value = intstr.IntOrString{Type: intstr.String, StrVal: ""}
			}
			if param.ValueType == numberType {
				param.Value = intstr.IntOrString{Type: intstr.Int, IntVal: 0}
			}
		}
		tempObjectInfo.Parameters[idx] = param
	}

	//replace parameter name to value in object and check exist k8s object
	totalParam := make(map[string]intstr.IntOrString)
	for _, param := range tempObjectInfo.Parameters {
		totalParam[param.Name] = param.Value
	}

	if instance.Status.ClusterTemplate == nil && instance.Status.Template == nil {
		for idx := range tempObjectInfo.Objects {
			if err = r.replaceParamsWithValue(&(tempObjectInfo.Objects[idx]), totalParam); err != nil {
				reqLogger.Error(err, "error occurs while replace parameters")
				return r.updateTemplateInstanceStatus(instance, err)
			}
			if err = r.checkObjectExist(&(tempObjectInfo.Objects[idx]), instance.Namespace); err != nil {
				reqLogger.Error(err, "exist resource")
				return r.updateTemplateInstanceStatus(instance, err)
			}
		}

		//create k8s object
		for idx := range tempObjectInfo.Objects {
			if err = r.createObject(&(tempObjectInfo.Objects[idx]), instance); err != nil {
				reqLogger.Error(err, "error occurs while create k8s object")
				return r.updateTemplateInstanceStatus(instance, err)
			}
		}

		if res, err := r.updateTemplateInstanceStatus(updateInstance, nil); err != nil {
			return res, err
		}
	}
	if instance.Status.ClusterTemplate != nil || instance.Status.Template != nil {
		for idx := range tempObjectInfo.Objects {
			if err = r.replaceParamsWithValue(&(tempObjectInfo.Objects[idx]), totalParam); err != nil {
				reqLogger.Error(err, "error occurs while replace parameters")
				return r.updateTemplateInstanceStatus(instance, err)
			}
		}

		//update k8s object
		for idx := range tempObjectInfo.Objects {
			if err = r.updateObject(&(tempObjectInfo.Objects[idx]), instance.Namespace); err != nil {
				reqLogger.Error(err, "error occurs while update k8s object")
				return r.updateTemplateInstanceStatus(instance, err)
			}
		}
	}

	if err := r.Client.Status().Patch(context.TODO(), updateInstance, client.MergeFrom(instance)); err != nil {
		reqLogger.Error(err, "could not update template instance status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *TemplateInstanceReconciler) createObject(obj *runtime.RawExtension, owner *tmaxiov1.TemplateInstance) error {
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

// Apply changed parameters on existing k8s objects which are populated by templateinstance.
// Get k8s obejcts as unstructured type and transform to []byte for applying parameters.
func (r *TemplateInstanceReconciler) updateObject(obj *runtime.RawExtension, ns string) error {
	updateUnstr, err := internal.BytesToUnstructuredObject(obj)
	if err != nil {
		return err
	}
	updateUnstr.SetNamespace(ns)
	unstr := updateUnstr.DeepCopy()

	// get already existing k8s object as unstructured type
	if err = r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: updateUnstr.GetNamespace(),
		Name:      updateUnstr.GetName(),
	}, unstr); err != nil {
		return err
	}

	bytedUnstr, _ := unstr.MarshalJSON()
	bytedUpdateUnstr, _ := updateUnstr.MarshalJSON()
	patchedByte, _ := jsonpatch.MergePatch(bytedUnstr, bytedUpdateUnstr)

	finalPatch := make(map[string]interface{})
	if err := json.Unmarshal(patchedByte, &finalPatch); err != nil {
		return err
	}
	unstr.SetUnstructuredContent(finalPatch)

	if err = r.Client.Update(context.TODO(), unstr); err != nil {
		return err
	}

	return nil
}

func (r *TemplateInstanceReconciler) checkObjectExist(obj *runtime.RawExtension, ns string) error {
	unstr, err := internal.BytesToUnstructuredObject(obj)
	if err != nil {
		return err
	}
	unstr.SetNamespace(ns)
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

func (r *TemplateInstanceReconciler) replaceParamsWithValue(obj *runtime.RawExtension, params map[string]intstr.IntOrString) error {
	reqLogger := r.Log.WithName("replace k8s object")
	objStr := string(obj.Raw)
	reqLogger.Info("original object: " + objStr)
	for key, value := range params {
		reqLogger.Info("key: " + key + " value: " + value.String())
		if value.Type == intstr.Int {
			objStr = strings.Replace(objStr, "\"${"+key+"}\"", value.String(), -1)
			objStr = strings.Replace(objStr, "${"+key+"}", value.String(), -1)
		} else {
			objStr = strings.Replace(objStr, "${"+key+"}", value.String(), -1)
		}
	}
	reqLogger.Info("replaced object: " + objStr)

	obj.Raw = []byte(objStr)
	return nil
}

func (r *TemplateInstanceReconciler) updateTemplateInstanceStatus(instance *tmaxiov1.TemplateInstance, err error) (ctrl.Result, error) {
	reqLogger := r.Log.WithName("update template instance status")
	// set condition depending on the error
	instanceWithStatus := instance.DeepCopy()

	var cond tmaxiov1.ConditionSpec
	if err == nil {
		cond.Message = "succeed to create instances"
		cond.Status = "Succeeded"
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
		Objects:         nil,
		ClusterTemplate: instance.Status.ClusterTemplate,
		Template:        instance.Status.Template,
	}

	if errUp := r.Client.Status().Patch(context.TODO(), instanceWithStatus, client.MergeFrom(instance)); errUp != nil {
		reqLogger.Error(errUp, "could not create template instance")
		return ctrl.Result{}, errUp
	}

	reqLogger.Info("succeed to create template instance status")
	return ctrl.Result{}, err
}

func ignoreStatusUpdate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore to call reconcile loop when TemplateInstanceStatus is updated
			oldSpec := e.ObjectOld.(*tmaxiov1.TemplateInstance).DeepCopy().Spec
			newSpec := e.ObjectNew.(*tmaxiov1.TemplateInstance).DeepCopy().Spec
			return !reflect.DeepEqual(oldSpec, newSpec)
		},
	}
}

func (r *TemplateInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmaxiov1.TemplateInstance{}).
		WithEventFilter(ignoreStatusUpdate()).
		Complete(r)
}
