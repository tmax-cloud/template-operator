package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type CatalogServiceClaimSpec struct {
	ApiVersion          string        `json:"apiVersion"`
	Kind                string        `json:"kind"`
	Labels              LabelsSpec    `json:"labels,omitempty"`
	ShortDescription    string        `json:"shortDescription,omitempty"`
	LongDescription     string        `json:"longDescription,omitempty"`
	UrlDescription      string        `json:"urlDescription,omitempty"`
	MarkdownDescription string        `json:"markdownDescription,omitempty"`
	Provider            string        `json:"provider,omitempty"`
	ImageUrl            string        `json:"imageUrl,omitempty"`
	Recommend           bool          `json:"recommend,omitempty"`
	Tags                []string      `json:"tags,omitempty"`
	Metadata            MetadataSpec  `json:"metadata"`
	Objects             []ObjectsSpec `json:"objects"`
	Plans               []ObjectsSpec `json:"plans,omitempty"`
	Params              []ParamsSpec  `json:"parameters"`
}

type CatalogServiceClaimStatus struct {
	// +kubebuilder:validation:Format:=date-time
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	Message            string `json:"message,omitempty"`
	Reason             string `json:"reason,omitempty"`
	// +kubebuilder:validation:Enum:=Awaiting;Success;Reject;Error
	Status string `json:"status,omitempty"`
}

// +kubebuilder:validation:XPreserveUnknownFields
type ObjectsSpec struct {
	Fields metav1.FieldsV1 `json:"fields,omitempty"`
}

type ParamsSpec struct {
	Description string `json:"description,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	From        string `json:"from,omitempty"`
	Generate    string `json:"generate,omitempty"`
	Name        string `json:"name"`
	Required    bool   `json:"required,omitempty"`
	Value       string `json:"value,omitempty"`
	ValueType   string `json:"valueType,omitempty"`
}

type LabelsSpec struct {
	AdditionalProperties string `json:"additionalProperties,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CatalogServiceClaim is the Schema for the catalogserviceclaims API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=catalogserviceclaims,scope=Namespaced
type CatalogServiceClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	OperatorStartTime string     `json:"operatorStartTime,omitempty"`
	ApiVersion        string     `json:"apiVersion"`
	Kind              string     `json:"kind"`
	Labels            LabelsSpec `json:"labels,omitempty"`

	Spec   CatalogServiceClaimSpec   `json:"spec"`
	Status CatalogServiceClaimStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CatalogServiceClaimList contains a list of CatalogServiceClaim
type CatalogServiceClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CatalogServiceClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CatalogServiceClaim{}, &CatalogServiceClaimList{})
}
