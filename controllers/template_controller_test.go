package controllers

import (
	"context"
	"testing"

	tmaxiov1 "github.com/tmax-cloud/template-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestTemplateController(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	var (
		name      = "test"
		namespace = "template-test"
	)

	// Template object with metadata and Objects Info
	template := &tmaxiov1.Template{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TemplateSpec: tmaxiov1.TemplateSpec{
			Objects: []runtime.RawExtension{
				{Raw: []byte(`{"kind": "Deployment"}`)},
				{Raw: []byte(`{"kind": "Service"}`)},
				{Raw: []byte(`{"kind": "Secret"}`)},
			},
		},
	}

	objs := []runtime.Object{template}

	s := scheme.Scheme
	s.AddKnownTypes(tmaxiov1.SchemeBuilder.GroupVersion, template)

	cl := fake.NewFakeClient(objs...)

	r := &TemplateReconciler{
		Client: cl,
		Log:    logf.Log.WithName("test-logger"),
		Scheme: s,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}

	if _, err := r.Reconcile(req); err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	// Check if the correct value is added to the ObjectKinds field
	tp := &tmaxiov1.Template{}
	if err := r.Client.Get(context.TODO(), req.NamespacedName, tp); err != nil {
		t.Fatalf("get template: (%v)", err)
	}

	ok := tp.ObjectKinds
	if len(ok) == 0 {
		t.Error("ObjectKinds is empty")
	}
	if ok[0] != "Deployment" || ok[1] != "Service" || ok[2] != "Secret" {
		t.Error("ObjectKinds have unexpected value")
	}
}
