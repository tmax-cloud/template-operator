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

package templateinstance

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	tmplv1 "github.com/tmax-cloud/template-operator/api/v1"
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

	globalTemplate := &tmplv1.ClusterTemplate{}
	// Fetch the TemplateInstance instance
	instance := &tmplv1.TemplateInstance{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 다른 namespace에 생성된 resource는 finalizer 통해서 삭제
	if instance.GetDeletionTimestamp() != nil {
		if err := r.removeDependents(instance); err != nil {
			reqLogger.Error(err, "failed to remove dependents")
		}
		return ctrl.Result{}, nil
	}

	// template/clustertemplate both empty or inserted
	if (instance.Spec.ClusterTemplate == nil) == (instance.Spec.Template == nil) {
		err := errors.NewBadRequest("You should insert either template or clustertemplate")
		reqLogger.Error(err, "Error occurs while get template info")
		return r.updateTemplateInstanceStatus(instance, err)
	}

	objectInfo := &tmplv1.ObjectInfo{}
	instanceParameters := []tmplv1.ParamSpec{}
	updateInstance := instance.DeepCopy()

	if instance.Spec.ClusterTemplate != nil { // instance with clustertemplate
		instanceParameters = instance.Spec.ClusterTemplate.Parameters

		if updateInstance.Status.ClusterTemplate == nil { // initial apply of instance
			updateInstance.Status.ClusterTemplate = objectInfo
			// Get the clustertemplate info
			template := &tmplv1.ClusterTemplate{}
			if err = r.Client.Get(context.TODO(), types.NamespacedName{
				Name: instance.Spec.ClusterTemplate.Metadata.Name,
			}, template); err != nil {
				reqLogger.Error(err, "Error occurs while get clustertemplate")
				return r.updateTemplateInstanceStatus(instance, err)
			}

			globalTemplate = template
			objectInfo.Metadata.Name = instance.Spec.ClusterTemplate.Metadata.Name
			objectInfo.Objects = template.Objects
			objectInfo.Object = template.Object
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
			template := &tmplv1.Template{}
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
	paramHandler := NewParamHandler(tempObjectInfo.Parameters, instanceParameters)

	if err := paramHandler.ReviseParam(); err != nil {
		reqLogger.Error(err, "Required parameter has no value")
		return r.updateTemplateInstanceStatus(instance, err)
	}

	totalParam := GetParamAsMap(paramHandler.templateParameters)
	// Regex validating parameter values
	if matched, m := RegexValidate(totalParam, objectInfo.Parameters); !matched {
		reqLogger.Error(err, "error occurs while checking parameter matches regex")
		return r.updateTemplateInstanceStatus(instance, fmt.Errorf(m))
	}

	//////////////////////////test//////////////////////////
	if len(objectInfo.Object) != 0 {
		tempObjectInfo.Objects, err = TemplateExec(globalTemplate, totalParam)
		if err != nil {
			reqLogger.Error(err, "error occurs while executing go template")
			return r.updateTemplateInstanceStatus(instance, err)
		}
	}
	//////////////////////////test//////////////////////////

	for key, val := range totalParam {
		reqLogger := r.Log.WithName("replace k8s object")
		reqLogger.Info("key: " + key + " value: " + val.String())
	}

	// gitops options
	//if instance.Annotations["gitops"] == "enable" {
	//	// Push template obejcts to given repo
	//	for idx := range tempObjectInfo.Objects {
	//		if err = r.replaceParamsWithValue(&(tempObjectInfo.Objects[idx]), totalParam); err != nil {
	//			reqLogger.Error(err, "error occurs while replace parameters")
	//			return r.updateTemplateInstanceStatus(instance, err)
	//		}
	//
	//		if err = internal.SetNamespace(&(tempObjectInfo.Objects[idx]), instance); err != nil {
	//			reqLogger.Error(err, "error occurs while update namespace")
	//			return r.updateTemplateInstanceStatus(instance, err)
	//		}
	//
	//		if err = internal.PushToGivenRepo(instance, tempObjectInfo.Objects[idx], r.Client); err != nil {
	//			reqLogger.Error(err, "error occurs while push objects")
	//			return r.updateTemplateInstanceStatus(instance, err)
	//		}
	//	}
	//
	//	// set template instance status
	//	if res, err := r.updateTemplateInstanceStatus(updateInstance, nil); err != nil {
	//		return res, err
	//	}
	//
	//	if err := r.Client.Status().Patch(context.TODO(), updateInstance, client.MergeFrom(instance)); err != nil {
	//		reqLogger.Error(err, "could not update template instance status")
	//		return ctrl.Result{}, err
	//	}
	//
	//	return ctrl.Result{}, nil
	//}

	// normal case (do not use gitops option)
	if instance.Status.ClusterTemplate == nil && instance.Status.Template == nil {
		for idx := range tempObjectInfo.Objects {
			if err = replaceParamsWithValue(&(tempObjectInfo.Objects[idx]), totalParam); err != nil {
				reqLogger.Error(err, "error occurs while replace parameters")
				return r.updateTemplateInstanceStatus(instance, err)
			}
			if err = r.checkObjectExist(&(tempObjectInfo.Objects[idx])); err != nil {
				reqLogger.Error(err, "exist resource")
				return r.updateTemplateInstanceStatus(instance, err)
			}
		}

		cacheUnstr := []*unstructured.Unstructured{} // cache for case of error
		finalizers := instance.GetFinalizers()

		//create k8s object
		for idx := range tempObjectInfo.Objects {
			cache := &unstructured.Unstructured{}

			var finalizer string

			if cache, finalizer, err = r.createObject(&(tempObjectInfo.Objects[idx]), updateInstance, finalizers); err != nil {
				reqLogger.Error(err, "error occurs while create k8s object")
				for _, cacheObj := range cacheUnstr {
					r.Client.Delete(context.TODO(), cacheObj) // when error occurs during create objects, delete already created objects
					reqLogger.Info("Object: " + cacheObj.GetKind() + " is deleted")
				}
				return r.updateTemplateInstanceStatus(instance, err)
			}

			finalizers = append(finalizers, finalizer)
			cacheUnstr = append(cacheUnstr, cache)
		}

		if res, err := r.updateTemplateInstanceStatus(updateInstance, nil); err != nil {
			return res, err
		}
	}

	if instance.Status.ClusterTemplate != nil || instance.Status.Template != nil {
		for idx := range tempObjectInfo.Objects {
			if err = replaceParamsWithValue(&(tempObjectInfo.Objects[idx]), totalParam); err != nil {
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

func (r *TemplateInstanceReconciler) removeDependents(instance *tmplv1.TemplateInstance) (err error) {

	var kind string
	var APIVersion string
	var ns string
	var name string
	var finalizersToBeDeleted []string

	// finalizer 분리해서 kind / ns / name 추출
	for _, finalizer := range instance.GetFinalizers() {
		dependentSignature := strings.Split(finalizer, ".-.")
		APIVersion = dependentSignature[0]
		kind = dependentSignature[1]
		ns = dependentSignature[2]
		name = dependentSignature[3]

		unstr := &unstructured.Unstructured{}
		unstr.SetAPIVersion(APIVersion)
		unstr.SetKind(kind)
		unstr.SetNamespace(ns)
		unstr.SetName(name)

		if err = r.Client.Delete(context.TODO(), unstr); err != nil {
			r.Log.Error(err, finalizer+" is not deleted")
			finalizersToBeDeleted = append(finalizersToBeDeleted, finalizer) // 누군가가 임의로 리소스 삭제해도 finalizer에서는 삭제
		} else {
			r.Log.Info(finalizer + " is deleted")
			finalizersToBeDeleted = append(finalizersToBeDeleted, finalizer)
		}
	}

	for _, finalizer := range finalizersToBeDeleted {
		// instance의 finalizer 삭제 및 업데이트
		controllerutil.RemoveFinalizer(instance, finalizer)
		if err := r.Client.Update(context.TODO(), instance); err != nil {
			r.Log.Error(err, "fail to update instance finalizer")
			return err
		}
	}

	return nil

}

func (r *TemplateInstanceReconciler) createObject(obj *runtime.RawExtension, owner *tmplv1.TemplateInstance, finalizers []string) (cache *unstructured.Unstructured, finalizer string, err error) {

	instanceWithFinalizer := owner.DeepCopy()

	// get unstructured object
	unstr, err := BytesToUnstructuredObject(obj)
	if err != nil {
		return nil, "", err
	}

	// namespace 설정을 안해주면 owner의 네임스페이스 설정 및 onwerRef 추가
	if len(unstr.GetNamespace()) == 0 {
		unstr.SetNamespace(owner.Namespace)

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
	} else { // namespace 설정이 있을 시, 해당 APIversion, kind, ns, name으로 finalizer 설정. 이 정보 가지고 추후 gc 진행
		finalizer = unstr.GetAPIVersion() + ".-." + unstr.GetKind() + ".-." + unstr.GetNamespace() + ".-." + unstr.GetName()

		var filtered []string // 빈 element가 하나 생겨서 일단 한번 필터링 해줌
		for _, str := range finalizers {
			if str != "" {
				filtered = append(filtered, str)
			}
		}

		var finalizerArray []string
		finalizerArray = append(filtered, finalizer)

		instanceWithFinalizer.SetFinalizers(finalizerArray)

		if errUp := r.Client.Patch(context.TODO(), instanceWithFinalizer, client.MergeFrom(owner)); errUp != nil {
			r.Log.Error(errUp, "could not create template instance")
			return nil, "", errUp
		}

		label := make(map[string]string)
		label["owner"] = owner.Kind + "-" + owner.Name
		unstr.SetLabels(label)
	}

	//reqLogger.Info("after: " + fmt.Sprintf("%+v\n", unstr.GetOwnerReferences()))
	// create object
	if err = r.Client.Create(context.TODO(), unstr); err != nil {
		return nil, "", err
	}
	r.Log.Info(unstr.GetKind() + " is created")
	return unstr, finalizer, nil
}

// Apply changed parameters on existing k8s objects which are populated by templateinstance.
// Get k8s obejcts as unstructured type and transform to []byte for applying parameters.
func (r *TemplateInstanceReconciler) updateObject(obj *runtime.RawExtension, ns string) error {
	updateUnstr, err := BytesToUnstructuredObject(obj)
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

func (r *TemplateInstanceReconciler) checkObjectExist(obj *runtime.RawExtension) error {
	unstr, err := BytesToUnstructuredObject(obj)
	if err != nil {

		return err
	}
	// unstr.SetNamespace(ns)
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

func (r *TemplateInstanceReconciler) updateTemplateInstanceStatus(instance *tmplv1.TemplateInstance, err error) (ctrl.Result, error) {
	reqLogger := r.Log.WithName("update template instance status")
	// set condition depending on the error
	instanceWithStatus := instance.DeepCopy()

	var cond tmplv1.ConditionSpec
	if err == nil {
		cond.Message = "succeed to create instances"
		cond.Status = "Succeeded"
	} else {
		cond.Message = err.Error()
		cond.Reason = "error occurs while create instance"
		cond.Status = "Error"
	}

	// set status
	instanceWithStatus.Status = tmplv1.TemplateInstanceStatus{
		Conditions: []tmplv1.ConditionSpec{
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
			oldSpec := e.ObjectOld.(*tmplv1.TemplateInstance).DeepCopy().Spec
			newSpec := e.ObjectNew.(*tmplv1.TemplateInstance).DeepCopy().Spec
			oldMeta := e.MetaOld.GetDeletionTimestamp()
			newMeta := e.MetaNew.GetDeletionTimestamp()
			return !reflect.DeepEqual(oldSpec, newSpec) || !reflect.DeepEqual(oldMeta, newMeta)
		},
	}
}

// func ignoreFinalizerUpdate() predicate.Predicate {
// 	return predicate.Funcs{
// 		UpdateFunc: func(e event.UpdateEvent) bool {
// 			// Ignore to call reconcile loop when TemplateInstanceStatus is updated
// 			oldFinalizer := e.ObjectOld.(*tmplv1.TemplateInstance).DeepCopy().ObjectMeta.Finalizers
// 			newFinalizer := e.ObjectNew.(*tmplv1.TemplateInstance).DeepCopy().ObjectMeta.Finalizers
// 			return !reflect.DeepEqual(oldFinalizer, newFinalizer)
// 		},
// 	}
// }

func (r *TemplateInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmplv1.TemplateInstance{}).
		WithEventFilter(ignoreStatusUpdate()).
		Complete(r)
}
