package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmplv1 "github.com/tmax-cloud/template-operator/api/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestTemplateInstanceController(t *testing.T) {
	var (
		templateName = "test-template"
		instanceName = "test-instance"
		objectName   = "test-object"
		namespace    = "test-ns"
	)

	template := &tmplv1.Template{
		ObjectMeta: metav1.ObjectMeta{
			Name:      templateName,
			Namespace: namespace,
		},
		TemplateSpec: tmplv1.TemplateSpec{
			Objects: []runtime.RawExtension{
				{Raw: []byte(`{"kind": "Deployment", "apiVersion": "apps/v1", "metadata": { "name": "${NAME}"},
				"spec": { "replicas": "${REPLICAS}"}}`)},
				{Raw: []byte(`{"kind": "Service", "apiVersion": "v1", "metadata": { "name": "${NAME}"}}`)},
			},
			Parameters: []tmplv1.ParamSpec{
				{Name: "NAME", ValueType: "string"},
				{Name: "REPLICAS", ValueType: "number"},
			},
		},
	}

	instance := &tmplv1.TemplateInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName,
			Namespace: namespace,
		},
		Spec: tmplv1.TemplateInstanceSpec{
			Template: &tmplv1.ObjectInfo{
				Metadata: tmplv1.MetadataSpec{
					Name: templateName,
				},
				Parameters: []tmplv1.ParamSpec{
					{Name: "NAME", Value: intstr.IntOrString{Type: intstr.String, StrVal: objectName}},
					// {Name: "REPLICAS", Value: intstr.IntOrString{Type: intstr.String, StrVal: "2"}},  // original case (Regex 추가 후 controller.go:143 변경 확인)
					{Name: "REPLICAS", Value: intstr.IntOrString{Type: intstr.Int, IntVal: 2}},
				},
			},
		},
	}

	objs := []runtime.Object{template, instance}

	s := scheme.Scheme
	s.AddKnownTypes(tmplv1.SchemeBuilder.GroupVersion, template)
	s.AddKnownTypes(tmplv1.SchemeBuilder.GroupVersion, instance)

	cl := fake.NewFakeClient(objs...)

	r := &TemplateInstanceReconciler{
		Client: cl,
		Log:    logf.Log.WithName("test-logger"),
		Scheme: s,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      instanceName,
			Namespace: namespace,
		},
	}

	_, err := r.Reconcile(req)
	require.NoError(t, err)

	TestDeploy := &appsv1.Deployment{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: objectName, Namespace: namespace}, TestDeploy)
	require.NoError(t, err)

	assert.Equal(t, *TestDeploy.Spec.Replicas, int32(2))
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: objectName, Namespace: namespace}, &corev1.Service{})
	require.NoError(t, err)

}
