package controller

import (
	"github.com/jitaeyun/template-operator/pkg/controller/template"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, template.Add)
}
