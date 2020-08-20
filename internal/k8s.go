package internal

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
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
