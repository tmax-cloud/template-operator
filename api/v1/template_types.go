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
	// An identifier used to correlate this Service Plan in future requests to the Service Broker.
	// Populated by the system.
	Id string `json:"id,omitempty"`
	// The name of the Service Plan. MUST be unique within the Service Class.
	// MUST be a non-empty string. Using a CLI-friendly name is RECOMMENDED.
	Name string `json:"name"`
	// A short description of the Service Plan. MUST be a non-empty string.
	Description string `json:"description,omitempty"`
	// An opaque object of metadata for a Service Plan.
	// It is expected that Platforms will treat this as a blob.
	// Note that there are conventions in existing Service Brokers and Platforms for fields that aid in the display of catalog data.
	Metadata PlanMetadata `json:"metadata,omitempty"`
	// When false, Service Instances of this Service Plan have a cost. The default is true.
	Free bool `json:"free,omitempty"`
	// Specifies whether Service Instances of the Service Plan can be bound to applications.
	Bindable bool `json:"bindable,omitempty"`
	// Whether the Plan supports upgrade/downgrade/sidegrade to another version.
	PlanUpdateable bool `json:"plan_updateable,omitempty"`
	// Schema definitions for Service Instances and Service Bindings for the Service Plan.
	Schemas Schemas `json:"schemas,omitempty"`
	// A duration, in seconds, that the Platform SHOULD use as the Service's maximum polling duration.
	MaximumPollingDuration int `json:"maximum_polling_duration,omitempty"`
	// Maintenance information for a Service Instance which is provisioned using the Service Plan.
	MaintenanceInfo MaintenanceInfo `json:"maintenance_info,omitempty"`
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
	// A description of the parameter.
	// Provide more detailed information for the purpose of the parameter, including any constraints on the expected value.
	// Descriptions should use complete sentences to follow the console’s text standards.
	// Don’t make this a duplicate of the display name.
	Description string `json:"description,omitempty"`
	// The user-friendly name for the parameter. This will be displayed to users.
	DisplayName string `json:"displayName,omitempty"`
	// The name of the parameter. This value is used to reference the parameter within the template.
	Name string `json:"name"`
	// Indicates this parameter is required, meaning the user cannot override it with an empty value.
	// If the parameter does not provide a default or generated value, the user must supply a value.
	Required bool `json:"required,omitempty"`
	// A default value for the parameter which will be used if the user does not override the value when instantiating the template.
	// Avoid using default values for things like passwords, instead use generated parameters in combination with Secrets.
	Value intstr.IntOrString `json:"value,omitempty"`
	// Set the data type of the parameter.
	// You can specify string and number for a string or integer type.
	// If not specified, it defaults to string.
	// +kubebuilder:validation:Enum:=string;number
	ValueType string `json:"valueType,omitempty"`
	// Set the "regex" value for the parameter value.
	// Given "regex" is used to validate parameter value from template instance.
	Regex string `json:"regex,omitempty"`
}

type LabelSpec struct {
	AdditionalProperties string `json:"additionalProperties,omitempty"`
}

// TemplateSpec defines the desired state of Template
// +kubebuilder:resource:shortName="tp"
type TemplateSpec struct {
	// Templates can include a set of labels.
	// These labels will be added to each object created when the template is instantiated.
	// Defining a label in this way makes it easy for users to find and manage all the objects created from a particular template.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// An instructional message that is displayed when this template is instantiated.
	// This field should inform the user how to use the newly created resources.
	// Parameter substitution is performed on the message before being displayed so that generated credentials and other parameters can be included in the output.
	// Include links to any next-steps documentation that users should follow.
	// +optional
	Message string `json:"message,omitempty"`
	// A description of the template.
	// Include enough detail that the user will understand what is being deployed and any caveats they need to know before deploying.
	// This will be displayed by the service catalog.
	// +optional
	ShortDescription string `json:"shortDescription,omitempty"`
	// Additional template description.
	// +optional
	LongDescription string `json:"longDescription,omitempty"`
	// A URL referencing further documentation for the template.
	// +optional
	UrlDescription string `json:"urlDescription,omitempty"`
	// Markdown format template description.
	// +optional
	MarkDownDescription string `json:"markdownDescription,omitempty"`
	// The name of the person or organization providing the template.
	// +optional
	Provider string `json:"provider,omitempty"`
	// An image url to be displayed with your template in the web console.
	// +optional
	ImageUrl string `json:"imageUrl,omitempty"`
	// Recommend specifies whether the template is recommended or not.
	Recommend bool `json:"recommend,omitempty"`
	// Tags to be associated with the template for searching and grouping.
	// Add tags that will include it into one of the provided catalog categories.
	// +optional
	Tags []string `json:"tags,omitempty"`
	// Categories for arranging templates by similarity
	// +optional
	Categories []string `json:"categories,omitempty"`
	// The kind list of objects that will be created by the template.
	// Populated by the system.
	// Read-only.
	ObjectKinds []string `json:"objectKinds,omitempty"`
	// Objects can be any valid API object, such as a IntegrationConfig, Deployment, Service, etc.
	// The object will be created exactly as defined here, with any parameter values substituted in prior to creation.
	// The definition of these objects can reference parameters defined earlier.
	// +kubebuilder:validation:XPreserveUnknownFields
	Objects []runtime.RawExtension `json:"objects"`
	// Service plan information to be used in the service catalog.
	Plans []PlanSpec `json:"plans,omitempty"`
	// Parameters allow a value to be supplied by the user or generated when the template is instantiated.
	// Then, that value is substituted wherever the parameter is referenced.
	// References can be defined in any field in the objects list field.
	// +optional
	Parameters []ParamSpec `json:"parameters,omitempty"`
}

// TemplateStatus defines the observed state of Template
type TemplateStatus struct {
	// Message indicates the message for the state of the template
	Message string `json:"message,omitempty"`
	// Reason indicates the reason for the state of the template
	Reason string `json:"reason,omitempty"`
	// Status indicates the status of the template.
	Status TemplateStatusType `json:"status,omitempty"`
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
