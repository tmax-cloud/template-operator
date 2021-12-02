package internal

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"

	tmaxiov1 "github.com/tmax-cloud/template-operator/api/v1"
)

func BytesToUnstructuredObject(obj *runtime.RawExtension) (*unstructured.Unstructured, error) {
	var in runtime.Object
	var scope conversion.Scope // While not actually used within the function, need to pass in
	if err := runtime.Convert_runtime_RawExtension_To_runtime_Object(obj, &in, scope); err != nil {
		return nil, err
	}

	unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(in)
	if err != nil {
		//reqLogger.Error(err, "cannot decode object")
		return nil, err
	}

	return &unstructured.Unstructured{Object: unstrObj}, nil
}

func SetNamespace(obj *runtime.RawExtension, owner *tmaxiov1.TemplateInstance) error {
	unstr, err := BytesToUnstructuredObject(obj)
	if err != nil {
		return err
	}

	unstr.SetNamespace(owner.Namespace)
	raw, err := json.Marshal(unstr)
	if err != nil {
		return err
	}
	obj.Raw = raw

	return nil
}
