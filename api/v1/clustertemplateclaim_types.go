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
)

const (
	Awating                string = "Awaiting"
	Approved               string = "Approved"
	ClusterTemplateDeleted string = "Cluster Template Deleted"
	Rejected               string = "Rejected"
	Error                  string = "Error"
)

// ClusterTemplateClaimSpec defines the desired state of ClusterTemplateClaim
type ClusterTemplateClaimSpec struct {
	ResourceName string `json:"resourceName"`
	TemplateName string `json:"template"`
}

// ClusterTemplateClaimStatus defines the observed state of ClusterTemplateClaim
type ClusterTemplateClaimStatus struct {
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Handled            bool        `json:"handled,omitempty"`
	// +kubebuilder:validation:Enum:=Awaiting;Approved;Cluster Template Deleted;Error;Rejected
	Status string `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=clustertemplateclaims,scope=Namespaced
// ClusterTemplateClaim is the Schema for the clustertemplateclaims API
type ClusterTemplateClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterTemplateClaimSpec   `json:"spec,omitempty"`
	Status ClusterTemplateClaimStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterTemplateClaimList contains a list of ClusterTemplateClaim
type ClusterTemplateClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterTemplateClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterTemplateClaim{}, &ClusterTemplateClaimList{})
}
