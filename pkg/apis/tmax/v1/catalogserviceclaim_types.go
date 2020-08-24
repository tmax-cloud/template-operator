package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type CatalogServiceClaimStatus struct {
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Message            string      `json:"message,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	// +kubebuilder:validation:Enum:=Awaiting;Success;Reject;Error
	Status string `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CatalogServiceClaim is the Schema for the catalogserviceclaims API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=catalogserviceclaims,scope=Cluster,shortName=csc
type CatalogServiceClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   ClusterTemplate           `json:"spec"`
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
