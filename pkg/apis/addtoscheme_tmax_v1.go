package apis

import (
	v1 "github.com/youngind/hypercloud-operator/pkg/apis/tmax/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1.SchemeBuilder.AddToScheme)
}
