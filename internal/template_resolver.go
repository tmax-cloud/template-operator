package internal

import (
	tmplv1 "github.com/tmax-cloud/template-operator/api/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type TemplateResolver struct {
	templateName string
	spec         tmplv1.TemplateSpec
}

func NewTemplateResolver(templateName string, spec tmplv1.TemplateSpec) *TemplateResolver {
	return &TemplateResolver{
		templateName,
		spec,
	}
}

func (r *TemplateResolver) SetTemplateDefaultFields() {
	if r.spec.ShortDescription == "" {
		r.spec.ShortDescription = r.templateName
	}

	if r.spec.ImageUrl == "" {
		r.spec.ImageUrl = "https://folo.co.kr/img/gm_noimage.png"
	}

	if r.spec.LongDescription == "" {
		r.spec.LongDescription = r.templateName
	}

	if r.spec.MarkDownDescription == "" {
		r.spec.MarkDownDescription = r.templateName
	}

	if r.spec.Provider == "" {
		r.spec.Provider = "tmax"
	}
}

func (r *TemplateResolver) SetParameterDefaultFields() {
	var newParams []tmplv1.ParamSpec

	for _, param := range r.spec.Parameters {
		if len(param.ValueType) == 0 {
			param.ValueType = "string"
		}

		// IntOrString default value{0 0 }    for util.go:44
		if param.ValueType == "string" && len(param.Value.StrVal) == 0 {
			param.Value = intstr.IntOrString{Type: intstr.String, StrVal: ""}
		}

		newParams = append(newParams, param)
	}
	r.spec.Parameters = newParams
}

func (r *TemplateResolver) SetObjectKinds() error {
	objectKinds := make([]string, 0)
	for _, obj := range r.spec.Objects {
		var in runtime.Object
		var scope conversion.Scope
		if err := runtime.Convert_runtime_RawExtension_To_runtime_Object(&obj, &in, scope); err != nil {
			return err
		}

		if unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(in); err != nil {
			return err
		} else {
			unstr := unstructured.Unstructured{Object: unstrObj}
			objectKinds = append(objectKinds, unstr.GetKind())
		}
	}

	r.spec.ObjectKinds = objectKinds
	return nil
}

func (r *TemplateResolver) Get() tmplv1.TemplateSpec {
	return r.spec
}
