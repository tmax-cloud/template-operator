package internal

import (
	"fmt"
	"regexp"
	"strconv"

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
			// [TODO : 아래 경우는 무조건 0으로 받아지기 때문에 필요 없음 / 문제생기는지 체크 필요]
			// if param.ValueType == numberType && val.Type == intstr.String {
			// 	convertedVal = intstr.IntOrString{Type: intstr.Int, IntVal: int32(val.IntValue())}
			// }
			if param.ValueType == stringType && val.Type == intstr.Int {
				convertedVal = intstr.IntOrString{Type: intstr.String, StrVal: val.String()}
			}
			param.Value = convertedVal
		}
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

	regCheckParam := make(map[string]string)
	m := "Regex Validation succeeded"

	// converse all param values to string type
	for key, val := range checkParamAsMap {
		if val.Type == intstr.Int {
			regCheckParam[key] = strconv.Itoa(int(val.IntVal))
		} else {
			regCheckParam[key] = val.StrVal
		}
	}

	for _, param := range paramSpec {
		name := param.Name
		if matched, _ := regexp.MatchString(param.Regex, regCheckParam[name]); !matched {
			m = fmt.Sprintf("parameter:%s value:%s doesn't match with given regex", name, regCheckParam[name])
			return matched, m
		}
	}

	return true, m
}

func (p *ParamHandler) GetTemplateParameters() []tmplv1.ParamSpec {
	return p.templateParameters
}
