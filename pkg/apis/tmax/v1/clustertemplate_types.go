package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterTemplate is the Schema for the clustertemplates API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=clustertemplates,scope=Cluster
type ClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	TemplateSpec      `json:",inline,omitempty"`
	Status            TemplateStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterTemplateList contains a list of ClusterTemplate
type ClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterTemplate{}, &ClusterTemplateList{})
}
