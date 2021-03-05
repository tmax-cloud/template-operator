/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type TemplateStatusType string

const (
	TemplateSuccess TemplateStatusType = "Success"
	TemplateError   TemplateStatusType = "Error"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:validation:XPreserveUnknownFields
type PlanSpec struct {
	Id                     string          `json:"id,omitempty"`
	Name                   string          `json:"name"`
	Description            string          `json:"description,omitempty"`
	Metadata               PlanMetadata    `json:"metadata,omitempty"`
	Free                   bool            `json:"free,omitempty"`
	Bindable               bool            `json:"bindable,omitempty"`
	PlanUpdateable         bool            `json:"plan_updateable,omitempty"`
	Schemas                Schemas         `json:"schemas,omitempty"`
	MaximumPollingDuration int             `json:"maximum_polling_duration,omitempty"`
	MaintenanceInfo        MaintenanceInfo `json:"maintenance_info,omitempty"`
}

type PlanMetadata struct {
	Bullets     []string `json:"bullets,omitempty"`
	Costs       Cost     `json:"costs,omitempty"`
	DisplayName string   `json:"displayName,omitempty"`
}

type Cost struct {
	Amount int    `json:"amount"`
	Unit   string `json:"unit"`
}

type MaintenanceInfo struct {
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}
type Schemas struct {
	ServiceInstance ServiceInstanceSchema `json:"service_instance,omitempty"`
	ServiceBinding  ServiceBindingSchema  `json:"service_binding,omitempty"`
}

type ServiceInstanceSchema struct {
	Create SchemaParameters `json:"create,omitempty"`
	Update SchemaParameters `json:"update,omitempty"`
}

type ServiceBindingSchema struct {
	Create SchemaParameters `json:"create,omitempty"`
}

type SchemaParameters struct {
	Parameters map[string]intstr.IntOrString `json:"parameters,omitempty"`
}
type ParamSpec struct {
	Description string             `json:"description,omitempty"`
	DisplayName string             `json:"displayName,omitempty"`
	From        string             `json:"from,omitempty"`
	Generate    string             `json:"generate,omitempty"`
	Name        string             `json:"name"`
	Required    bool               `json:"required,omitempty"`
	Value       intstr.IntOrString `json:"value,omitempty"`
	ValueType   string             `json:"valueType,omitempty"`
}

type LabelSpec struct {
	AdditionalProperties string `json:"additionalProperties,omitempty"`
}

// TemplateSpec defines the desired state of Template
// +kubebuilder:resource:shortName="tp"
type TemplateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Labels              map[string]string      `json:"labels,omitempty"`
	Message             string                 `json:"message,omitempty"`
	ShortDescription    string                 `json:"shortDescription,omitempty"`
	LongDescription     string                 `json:"longDescription,omitempty"`
	UrlDescription      string                 `json:"urlDescription"`
	MarkDownDescription string                 `json:"markdownDescription,omitempty"`
	Provider            string                 `json:"provider,omitempty"`
	ImageUrl            string                 `json:"imageUrl,omitempty"`
	Recommend           bool                   `json:"recommend,omitempty"`
	Tags                []string               `json:"tags,omitempty"`
	ObjectKinds         []string               `json:"objectKinds,omitempty"`
	Objects             []runtime.RawExtension `json:"objects,omitempty"`
	Plans               []PlanSpec             `json:"plans,omitempty"`
	Parameters          []ParamSpec            `json:"parameters,omitempty"`
}

// TemplateStatus defines the observed state of Template
type TemplateStatus struct {
	Message string             `json:"message,omitempty"`
	Reason  string             `json:"reason,omitempty"`
	Status  TemplateStatusType `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=templates,scope=Namespaced

// Template is the Schema for the templates API
type Template struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	TemplateSpec      `json:",inline,omitempty"`
	Status            TemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TemplateList contains a list of Template
type TemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Template `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Template{}, &TemplateList{})
}
