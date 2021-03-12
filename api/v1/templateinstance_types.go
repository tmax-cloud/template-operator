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
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type MetadataSpec struct {
	Name string `json:"name,omitempty"`
}

// +kubebuilder:validation:XPreserveUnknownFields
type ExtraSpec struct {
}

type RequesterSpec struct {
	Extra    ExtraSpec `json:"extra,omitempty"`
	Groups   []string  `json:"groups,omitempty"`
	Uid      string    `json:"uid,omitempty"`
	Username string    `json:"username,omitempty"`
}

type SecretSpec struct {
	Name string `json:"name,omitempty"`
}

type ObjectInfo struct {
	Metadata   MetadataSpec           `json:"metadata,omitempty"`
	Objects    []runtime.RawExtension `json:"objects,omitempty"`
	Parameters []ParamSpec            `json:"parameters,omitempty"`
}

// +kubebuilder:resource:shortName="ti"
// TemplateInstanceSpec defines the desired state of TemplateInstance
type TemplateInstanceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Requester RequesterSpec `json:"requester,omitempty"`
	Secret    SecretSpec    `json:"secret,omitempty"`
	// Important: Use only one of the fields Template and ClusterTemplate. Fill in only metadata.name and parameters inside this field.
	// +kubebuilder:validation:OneOf [TODO: 나중에 추가할것]
	Template *ObjectInfo `json:"template,omitempty"`
	// Important: Use only one of the fields Template and ClusterTemplate. Fill in only metadata.name and parameters inside this field.
	// +kubebuilder:validation:OneOf [TODO: 나중에 추가할것]
	ClusterTemplate *ObjectInfo `json:"clustertemplate,omitempty"`
}

type RefSpec struct {
	ApiVersion      string `json:"apiVersion,omitempty"`
	FieldPath       string `json:"fieldPath,omitempty"`
	Kind            string `json:"kind,omitempty"`
	Name            string `json:"name,omitempty"`
	Namespace       string `json:"namespace,omitempty"`
	ResourceVersion string `json:"resourceVersion,omitempty"`
	Uid             string `json:"uid,omitempty"`
}

type StatusObjectSpec struct {
	Ref RefSpec `json:"ref"`
}

type ConditionSpec struct {
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`
	Message            string       `json:"message,omitempty"`
	Reason             string       `json:"reason,omitempty"`
	Status             string       `json:"status,omitempty"`
	Type               string       `json:"type"`
}

// TemplateInstanceStatus defines the observed state of TemplateInstance
type TemplateInstanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions []ConditionSpec    `json:"conditions,omitempty"`
	Objects    []StatusObjectSpec `json:"objects,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=templateinstances,scope=Namespaced
// TemplateInstance is the Schema for the templateinstances API
type TemplateInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TemplateInstanceSpec   `json:"spec,omitempty"`
	Status TemplateInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TemplateInstanceList contains a list of TemplateInstance
type TemplateInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TemplateInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TemplateInstance{}, &TemplateInstanceList{})
}
