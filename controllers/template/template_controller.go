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

package template

import (
	"context"
	"fmt"
	"github.com/tmax-cloud/template-operator/internal"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmplv1 "github.com/tmax-cloud/template-operator/api/v1"
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

	// Fetch the Template
	template := &tmplv1.Template{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, template)
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
	if len(template.Status.Status) != 0 {
		reqLogger.Info("already handled template")
		return ctrl.Result{}, nil
	}

	// copy reconciling template from original
	updateTemplate := template.DeepCopy()

	templateResolver := internal.NewTemplateResolver(updateTemplate.GetObjectMeta().GetName(), updateTemplate.TemplateSpec)
	templateResolver.SetTemplateDefaultFields()
	templateResolver.SetParameterDefaultFields()
	if err := templateResolver.SetObjectKinds(); err != nil {
		reqLogger.Error(err, "cannot decode object")
		templateStatus := &tmplv1.TemplateStatus{
			Message: "cannot decode object",
			Status:  tmplv1.TemplateError,
		}
		return r.updateTemplateStatus(template, templateStatus)
	}

	updateTemplate.TemplateSpec = templateResolver.Get()

	reqLogger.Info(fmt.Sprintf("object kinds: %v", updateTemplate.ObjectKinds))

	// Patch reconciled template
	if err = r.Client.Patch(context.TODO(), updateTemplate, client.MergeFrom(template)); err != nil {
		reqLogger.Error(err, "cannot update template")
		templateStatus := &tmplv1.TemplateStatus{
			Message: "cannot update template",
			Status:  tmplv1.TemplateError,
		}
		return r.updateTemplateStatus(template, templateStatus)
	}

	// update status when succeed
	templateStatus := &tmplv1.TemplateStatus{
		Message: "update success",
		Status:  tmplv1.TemplateSuccess,
	}
	return r.updateTemplateStatus(template, templateStatus)
}

func (r *TemplateReconciler) updateTemplateStatus(
	template *tmplv1.Template, status *tmplv1.TemplateStatus) (ctrl.Result, error) {
	reqLogger := r.Log.WithName("update template status")

	updatedTemplate := template.DeepCopy()
	updatedTemplate.Status = *status

	if err := r.Client.Status().Patch(context.TODO(), updatedTemplate, client.MergeFrom(template)); err != nil {
		reqLogger.Error(err, "could not update Template status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *TemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmplv1.Template{}).
		Complete(r)
}
