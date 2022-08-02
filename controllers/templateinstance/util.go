package templateinstance

import (
	"bytes"
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"regexp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/yaml"
	"strconv"
	"strings"
	"text/template"

	"k8s.io/apimachinery/pkg/api/errors"

	tmplv1 "github.com/tmax-cloud/template-operator/api/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	stringType = "string"
	numberType = "number"
)

type ParamHandler struct {
	templateParameters []tmplv1.ParamSpec
	instanceParameters []tmplv1.ParamSpec
}

func NewParamHandler(templateParameters, instanceParameters []tmplv1.ParamSpec) *ParamHandler {
	return &ParamHandler{
		templateParameters,
		instanceParameters,
	}
}

func (p *ParamHandler) ReviseParam() error {
	instanceParams := GetParamAsMap(p.instanceParameters)

	for idx, param := range p.templateParameters {
		if val, exist := instanceParams[param.Name]; exist {
			convertedVal := val
			if param.ValueType == numberType && val.Type == intstr.String {
				convertedVal = intstr.IntOrString{Type: intstr.Int, IntVal: int32(val.IntValue())}
			}
			if param.ValueType == stringType && val.Type == intstr.Int {
				convertedVal = intstr.IntOrString{Type: intstr.String, StrVal: val.String()}
			}
			// in case of Service Instance has no value
			if param.ValueType == stringType && val.Type == intstr.String && len(val.StrVal) == 0 {
				convertedVal = param.Value
			}
			param.Value = convertedVal
		}
		// [TODO]: UI 변경되면 확인해야함 (tsb create Template Instance 부분)
		// If the required field has no value
		if param.Required && param.Value.Type == 1 && len(param.Value.StrVal) == 0 {
			err := errors.NewBadRequest(param.Name + "must have a value")
			return err
		}

		// // Set default value for not required parameter
		// if param.Value.Size() == 0 {
		// 	if len(param.ValueType) == 0 || param.ValueType == stringType {
		// 		param.Value = intstr.IntOrString{Type: intstr.String, StrVal: ""}
		// 	}
		// 	if param.ValueType == numberType {
		// 		param.Value = intstr.IntOrString{Type: intstr.Int, IntVal: 0}
		// 	}
		// }

		p.templateParameters[idx] = param
	}
	return nil
}

func GetParamAsMap(parameters []tmplv1.ParamSpec) (resultParam map[string]intstr.IntOrString) {
	resultParam = make(map[string]intstr.IntOrString)
	for _, param := range parameters {
		resultParam[param.Name] = param.Value
	}
	return resultParam
}

func RegexValidate(checkParamAsMap map[string]intstr.IntOrString, paramSpec []tmplv1.ParamSpec) (matched bool, msg string) {

	m := "Regex Validation succeeded"

	for _, param := range paramSpec {
		intOrStrVal := checkParamAsMap[param.Name]
		var stringVal string
		if intOrStrVal.Type == intstr.Int {
			stringVal = strconv.Itoa(int(intOrStrVal.IntVal))
		} else {
			stringVal = intOrStrVal.StrVal
		}
		if matched, _ := regexp.MatchString(param.Regex, stringVal); !matched {
			m = fmt.Sprintf("parameter:%s value:%s doesn't match with given regex", param.Name, stringVal)
			return matched, m
		}
	}

	return true, m
}

func replaceParamsWithValue(obj *runtime.RawExtension, params map[string]intstr.IntOrString) error {
	reqLogger := ctrl.Log.WithName("replace k8s object")
	objStr := string(obj.Raw)
	reqLogger.Info("original object: " + objStr)
	for key, value := range params {
		// reqLogger.Info("key: " + key + " value: " + value.String())
		if value.Type == intstr.Int {
			objStr = strings.Replace(objStr, "\"${"+key+"}\"", value.String(), -1)
			objStr = strings.Replace(objStr, "${"+key+"}", value.String(), -1)
		} else {
			objStr = strings.Replace(objStr, "${"+key+"}", value.String(), -1)
		}
	}
	reqLogger.Info("replaced object: " + objStr)

	obj.Raw = []byte(objStr)
	return nil
}

func TemplateExec(tmpl *tmplv1.ClusterTemplate, param map[string]intstr.IntOrString) (result []runtime.RawExtension, err error) {
	log := ctrl.Log.WithName("Go template")
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
