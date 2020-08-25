package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

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

// +kubebuilder:resource:shortName="ti"
// TemplateInstanceSpec defines the desired state of TemplateInstance
type TemplateInstanceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Requester       RequesterSpec   `json:"requester,omitempty"`
	Secret          SecretSpec      `json:"secret,omitempty"`
	Template        Template        `json:"template,omitempty"`
	ClusterTemplate ClusterTemplate `json:"clustertemplate,omitempty"`
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
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	Conditions []ConditionSpec    `json:"conditions,omitempty"`
	Objects    []StatusObjectSpec `json:"objects,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TemplateInstance is the Schema for the Templateinstances API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=templateinstances,scope=Namespaced
type TemplateInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TemplateInstanceSpec   `json:"spec,omitempty"`
	Status TemplateInstanceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TemplateInstanceList contains a list of TemplateInstance
type TemplateInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TemplateInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TemplateInstance{}, &TemplateInstanceList{})
}
