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

package clustertemplate

import (
	"context"
	"fmt"
	"github.com/tmax-cloud/template-operator/internal"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	tmplv1 "github.com/tmax-cloud/template-operator/api/v1"
)

// ClusterTemplateReconciler reconciles a ClusterTemplate object
type ClusterTemplateReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tmax.io,resources=clustertemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=clustertemplates/status,verbs=get;update;patch

func (r *ClusterTemplateReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling ClusterTemplate")

	// Fetch the ClusterTemplate instance
	template := &tmplv1.ClusterTemplate{}
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

	// if ClusterTemplate was created by claim, claim status must be updated when the ClusterTemplate is deleted
	if template.GetDeletionTimestamp() != nil && controllerutil.ContainsFinalizer(template, internal.ClaimFinalizer) {
		if err := r.updateClaimStatus(reqLogger, template); err != nil {
			reqLogger.Error(err, "fail to update claim status")
		}
		controllerutil.RemoveFinalizer(template, internal.ClaimFinalizer)
		if err := r.Client.Update(context.TODO(), template); err != nil {
			reqLogger.Error(err, "fail to update template")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
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
		return r.updateClusterTemplateStatus(template, templateStatus)
	}

	updateTemplate.TemplateSpec = templateResolver.Get()

	reqLogger.Info(fmt.Sprintf("object kinds: %v", updateTemplate.ObjectKinds))

	// Patch reconciled template
	if err = r.Client.Patch(context.TODO(), updateTemplate, client.MergeFrom(template)); err != nil {
		reqLogger.Error(err, "cannot update clustertemplate")
		templateStatus := &tmplv1.TemplateStatus{
			Message: "cannot update clustertemplate",
			Status:  tmplv1.TemplateError,
		}
		return r.updateClusterTemplateStatus(template, templateStatus)
	}

	// update status when succeed
	templateStatus := &tmplv1.TemplateStatus{
		Message: "update success",
		Status:  tmplv1.TemplateSuccess,
	}
	return r.updateClusterTemplateStatus(template, templateStatus)
}

func (r *ClusterTemplateReconciler) updateClaimStatus(reqLogger logr.Logger, ct *tmplv1.ClusterTemplate) error {
	claim := &tmplv1.ClusterTemplateClaim{}

	claimInfo := strings.Split(ct.ObjectMeta.Labels[internal.ClaimLabel], ".")
	claimNamespacedName := types.NamespacedName{
		Namespace: claimInfo[1],
		Name:      claimInfo[0],
	}
	if err := r.Client.Get(context.TODO(), claimNamespacedName, claim); err != nil {
		reqLogger.Error(err, "Failed to get claim")
		return err
	}

	updatedClaim := claim.DeepCopy()
	updatedClaim.Status = tmplv1.ClusterTemplateClaimStatus{
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "ClusterTemplate was deleted",

		Status:  tmplv1.ClusterTemplateDeleted,
		Handled: true,
	}

	if err := r.Client.Status().Patch(context.TODO(), updatedClaim, client.MergeFrom(claim)); err != nil {
		reqLogger.Error(err, "Error occurs while updating ClusterTemplateClaim status")
		return err
	}

	reqLogger.Info("Successfully finalized clustertemplate")
	return nil
}

func (r *ClusterTemplateReconciler) updateClusterTemplateStatus(
	template *tmplv1.ClusterTemplate, status *tmplv1.TemplateStatus) (ctrl.Result, error) {
	reqLogger := r.Log.WithName("update clustertemplate status")

	updatedTemplate := template.DeepCopy()
	updatedTemplate.Status = *status

	if err := r.Client.Status().Patch(context.TODO(), updatedTemplate, client.MergeFrom(template)); err != nil {
		reqLogger.Error(err, "could not update clusterTemplate status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ClusterTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmplv1.ClusterTemplate{}).
		Complete(r)
}
