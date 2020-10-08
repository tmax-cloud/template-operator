package controller

import (
	"github.com/tmax-cloud/template-operator/pkg/controller/templateinstance"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, templateinstance.Add)
}
