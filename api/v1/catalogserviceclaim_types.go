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

type ClaimStatusType string

const (
	ClaimAwating ClaimStatusType = "Awaiting"
	ClaimSuccess ClaimStatusType = "Success"
	ClaimApprove ClaimStatusType = "Approve"
	ClaimReject  ClaimStatusType = "Reject"
	ClaimError   ClaimStatusType = "Error"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CatalogServiceClaimStatus defines the observed state of CatalogServiceClaim
type CatalogServiceClaimStatus struct {
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Message            string      `json:"message,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	// +kubebuilder:validation:Enum:=Awaiting;Success;Approve;Reject;Error
	Status ClaimStatusType `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=catalogserviceclaims,scope=Cluster,shortName=csc
// CatalogServiceClaim is the Schema for the catalogserviceclaims API
type CatalogServiceClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   ClusterTemplate           `json:"spec"`
	Status CatalogServiceClaimStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CatalogServiceClaimList contains a list of CatalogServiceClaim
type CatalogServiceClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CatalogServiceClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CatalogServiceClaim{}, &CatalogServiceClaimList{})
}
