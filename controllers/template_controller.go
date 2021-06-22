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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmaxiov1 "github.com/tmax-cloud/template-operator/api/v1"
)

// TemplateReconciler reconciles a Template object
type TemplateReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tmax.io,resources=templates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=templates/status,verbs=get;update;patch

func (r *TemplateReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling Template")

	// Fetch the Template instance
	instance := &tmaxiov1.Template{}
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
	if len(instance.Status.Status) != 0 {
		reqLogger.Info("already handled template")
		return ctrl.Result{}, nil
	}

	err = setTemplateSpecDefaultField(instance)
	if err != nil {
		reqLogger.Info("Updating Template Default Field is failed")
	}
	updateInstance := instance.DeepCopy()

	// add kind to objectKinds fields
	objectKinds := make([]string, 0)
	for _, obj := range instance.Objects {
		var in runtime.Object
		var scope conversion.Scope // While not actually used within the function, need to pass in
		if err = runtime.Convert_runtime_RawExtension_To_runtime_Object(&obj, &in, scope); err != nil {
			reqLogger.Error(err, "cannot decode object")
			templateStatus := &tmaxiov1.TemplateStatus{
				Message: "cannot decode object",
				Status:  tmaxiov1.TemplateError,
			}
			return r.updateTemplateStatus(instance, templateStatus)
		}

		unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(in)
		if err != nil {
			reqLogger.Error(err, "cannot decode object")
			templateStatus := &tmaxiov1.TemplateStatus{
				Message: "cannot decode object",
				Status:  tmaxiov1.TemplateError,
			}
			return r.updateTemplateStatus(instance, templateStatus)
		}

		unstr := unstructured.Unstructured{Object: unstrObj}
		reqLogger.Info(fmt.Sprintf("kind: %s", unstr.GetKind()))
		objectKinds = append(objectKinds, unstr.GetKind())
	}
	updateInstance.ObjectKinds = objectKinds
	reqLogger.Info(fmt.Sprintf("%v", objectKinds))

	if err = r.Client.Patch(context.TODO(), updateInstance, client.MergeFrom(instance)); err != nil {
		reqLogger.Error(err, "cannot update template")
		templateStatus := &tmaxiov1.TemplateStatus{
			Message: "cannot update template",
			Status:  tmaxiov1.TemplateError,
		}
		return r.updateTemplateStatus(instance, templateStatus)
	}

	templateStatus := &tmaxiov1.TemplateStatus{
		Message: "update success",
		Status:  tmaxiov1.TemplateSuccess,
	}
	return r.updateTemplateStatus(instance, templateStatus)
}

func (r *TemplateReconciler) updateTemplateStatus(
	template *tmaxiov1.Template, status *tmaxiov1.TemplateStatus) (ctrl.Result, error) {
	reqLogger := r.Log.WithName("update template status")

	updatedTemplate := template.DeepCopy()
	updatedTemplate.Status = *status

	if err := r.Client.Status().Patch(context.TODO(), updatedTemplate, client.MergeFrom(template)); err != nil {
		reqLogger.Error(err, "could not update Template status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func setTemplateSpecDefaultField(template *tmaxiov1.Template) error {
	if template.ShortDescription == "" {
		template.ShortDescription = template.ObjectMeta.Name
	}

	if template.ImageUrl == "" {
		template.ImageUrl = "https://folo.co.kr/img/gm_noimage.png"
	}
	if template.LongDescription == "" {
		template.LongDescription = template.ObjectMeta.Name
	}

	if template.MarkDownDescription == "" {
		template.MarkDownDescription = template.ObjectMeta.Name
	}

	if template.Provider == "" {
		template.Provider = "tmax"
	}
	return nil
}

func (r *TemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmaxiov1.Template{}).
		Complete(r)
}
