package internal

import (
	"bytes"
	"text/template"

	"github.com/ghodss/yaml"
	tmplv1 "github.com/tmax-cloud/template-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("Go template")

func TemplateExec(tmpl *tmplv1.ClusterTemplate, param map[string]intstr.IntOrString) (result []runtime.RawExtension, err error) {

	tmplParam := make(map[string]interface{})
	for k, v := range param {
		if v.Type == 0 { // valueType : integer
			tmplParam[k] = v.IntVal
		} else {
			tmplParam[k] = v.StrVal
		}
	}

	t := template.New("object template")
	cache := runtime.RawExtension{}
	for _, tp := range tmpl.Object {
		t, err = t.Parse(tp)
		if err != nil {
			log.Error(err, "parsing error")
			return nil, err
		}

		buf := new(bytes.Buffer)
		err = t.Execute(buf, tmplParam)
		if err != nil {
			log.Error(err, "template executing error")
			return nil, err
		}

		cache.Raw, _ = yaml.YAMLToJSON(buf.Bytes())
		result = append(result, cache)
	}
	return result, nil
}
