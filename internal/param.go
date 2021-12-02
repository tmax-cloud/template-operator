package internal

import (
	"fmt"
	"regexp"
	"strconv"

	tmplv1 "github.com/tmax-cloud/template-operator/api/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

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
