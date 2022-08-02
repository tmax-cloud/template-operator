package template

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmplv1 "github.com/tmax-cloud/template-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestTemplateController(t *testing.T) {
	var (
		name      = "test"
		namespace = "template-test"
	)

	// Template object with metadata and Objects Info
	template := &tmplv1.Template{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TemplateSpec: tmplv1.TemplateSpec{
			Objects: []runtime.RawExtension{
				{Raw: []byte(`{"kind": "Deployment"}`)},
				{Raw: []byte(`{"kind": "Service"}`)},
				{Raw: []byte(`{"kind": "Secret"}`)},
			},
		},
	}

	objs := []runtime.Object{template}

	s := scheme.Scheme
	s.AddKnownTypes(tmplv1.SchemeBuilder.GroupVersion, template)

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

	_, err := r.Reconcile(req)
	require.NoError(t, err)

	// Check if the correct value is added to the ObjectKinds field
	tp := &tmplv1.Template{}
	err = r.Client.Get(context.TODO(), req.NamespacedName, tp)
	require.NoError(t, err)

	ok := tp.ObjectKinds
	assert.NotEqual(t, 0, len(ok), "ObjectKinds is empty")

	assert.Equal(t, "Deployment", ok[0], "ObjectKinds have unexpected value")
	assert.Equal(t, "Service", ok[1], "ObjectKinds have unexpected value")
	assert.Equal(t, "Secret", ok[2], "ObjectKinds have unexpected value")
}
