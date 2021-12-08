// Deprecated at 211208

package internal

// import (
// 	"encoding/json"
// 	"fmt"

// 	"github.com/tmax-cloud/template-operator/schemas"
// 	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
// 	"k8s.io/apimachinery/pkg/runtime"

// 	tmplv1 "github.com/tmax-cloud/template-operator/api/v1"
// )

// var (
// 	appAPIVersion = "argoproj.io/v1alpha1"
// 	appKind       = "Application"
// 	argoDefaultNs = "argocd"
// )

// // Template일 경우도 추가
// func CreateApplicationAsUnstr(instance *tmplv1.TemplateInstance) (*unstructured.Unstructured, error) {
// 	// Set Default Application values from template instance
// 	app := &schemas.Application{}
// 	app.APIVersion = appAPIVersion
// 	app.Kind = appKind
// 	app.Name = instance.Name + "-" + instance.Spec.ClusterTemplate.Metadata.Name
// 	app.ObjectMeta.Namespace = argoDefaultNs

// 	app.Spec.Source.RepoURL = MutateRepoURL(instance.Spec.Gitops.SourceGitRepo)
// 	app.Spec.Source.Path = MutateRepoPath(instance.Spec.Gitops.Path)
// 	app.Spec.Source.TargetRevision = "HEAD"

// 	app.Spec.Destination.Server = instance.Spec.Gitops.Destination
// 	app.Spec.Destination.Namespace = instance.Namespace

// 	syncPolicy := &schemas.SyncPolicyAutomated{
// 		Prune: true,
// 	}
// 	app.Spec.SyncPolicy = &schemas.SyncPolicy{
// 		Automated: syncPolicy,
// 	}

// 	byteApp, err := json.Marshal(app)
// 	if err != nil {
// 		fmt.Println(err, "error occurs while marshal into byteApp")
// 		return nil, err
// 	}

// 	objApp := runtime.RawExtension{Raw: byteApp}
// 	unstrApp, err := BytesToUnstructuredObject(&objApp)
// 	if err != nil {
// 		fmt.Println(err, "error occurs while transform into unstrApp")
// 		return nil, err
// 	}

// 	isController := false
// 	blockOwnerDeletion := true

// 	ownerRefs := unstrApp.GetOwnerReferences()
// 	ownerRef := v1.OwnerReference{
// 		APIVersion:         instance.APIVersion,
// 		Kind:               instance.Kind,
// 		Name:               instance.Name,
// 		UID:                instance.UID,
// 		Controller:         &isController,
// 		BlockOwnerDeletion: &blockOwnerDeletion,
// 	}
// 	ownerRefs = append(ownerRefs, ownerRef)
// 	unstrApp.SetOwnerReferences(ownerRefs)

// 	return unstrApp, nil
// }

// Controller logic

// // Create Application CR through unstructrued type
// unstrApp, err := internal.CreateApplicationAsUnstr(instance)
// if err != nil {
// 	reqLogger.Error(err, "error occurs while get unstrApp")
// 	return r.updateTemplateInstanceStatus(instance, err)
// }

// if err := r.Client.Get(context.TODO(), types.NamespacedName{
// 	Namespace: argoDefaultNs,
// 	Name:      instance.Name + "-" + templateName,
// }, unstrApp); err != nil {
// 	if err = r.Client.Create(context.TODO(), unstrApp); err != nil {
// 		return r.updateTemplateInstanceStatus(instance, err)
// 	}
// }
