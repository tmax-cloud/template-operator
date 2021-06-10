package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	tmaxiov1 "github.com/tmax-cloud/template-operator/api/v1"
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

	template := &tmaxiov1.Template{
		ObjectMeta: metav1.ObjectMeta{
			Name:      templateName,
			Namespace: namespace,
		},
		TemplateSpec: tmaxiov1.TemplateSpec{
			Objects: []runtime.RawExtension{
				{Raw: []byte(`{"kind": "Pod", "apiVersion": "v1", "metadata": { "name": "${NAME}"}}`)},
				{Raw: []byte(`{"kind": "Service", "apiVersion": "v1", "metadata": { "name": "${NAME}"}}`)},
			},
			Parameters: []tmaxiov1.ParamSpec{
				{Name: "NAME", ValueType: "string"},
			},
		},
	}

	instance := &tmaxiov1.TemplateInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName,
			Namespace: namespace,
		},
		Spec: tmaxiov1.TemplateInstanceSpec{
			Template: &tmaxiov1.ObjectInfo{
				Metadata: tmaxiov1.MetadataSpec{
					Name: templateName,
				},
				Parameters: []tmaxiov1.ParamSpec{
					{Name: "NAME", Value: intstr.IntOrString{Type: 1, StrVal: objectName}},
				},
			},
		},
	}

	objs := []runtime.Object{template, instance}

	s := scheme.Scheme
	s.AddKnownTypes(tmaxiov1.SchemeBuilder.GroupVersion, template)
	s.AddKnownTypes(tmaxiov1.SchemeBuilder.GroupVersion, instance)

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

	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: objectName, Namespace: namespace}, &corev1.Pod{})
	require.NoError(t, err)
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: objectName, Namespace: namespace}, &corev1.Service{})
	require.NoError(t, err)
}
