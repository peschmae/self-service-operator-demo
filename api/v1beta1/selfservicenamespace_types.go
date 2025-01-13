/*
Copyright 2025.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SelfServiceNamespaceSpec defines the desired state of SelfServiceNamespace.
type SelfServiceNamespaceSpec struct {
	// Additional labels to be added to the Namespace
	// +optional
	AdditionalLabels map[string]string `json:"additionalLabels" protobuf:"bytes,11,rep,name=labels"`

	// Additional annotations to be added to the Namespace
	// +optional
	AdditionalAnnotations map[string]string `json:"additionalAnnotations" protobuf:"bytes,12,rep,name=annotations"`

	// EgressEndpoints to be configured as NetworkPolicies in the Namespace
	// +optional
	EgressConfigurations []EgressConfigurationSpec `json:"egressConfiguration"`

	// Enable network checks for the Namespace
	// +kubebuilder:default=false
	// +optional
	NetworkChecksEnabled bool `json:"networkChecksEnabled"`

	// Networks checks to be configured for the check script
	// +optional
	NetworkChecks []NetworkCheckConfigurationSpec `json:"networkChecks"`
}

// EgressConfigurationSpec reflects a simplified version of the NetworkPolicy EgressRule
type EgressConfigurationSpec struct {
	// cidr is a string representing the IPBlock Valid examples are "192.168.1.0/24" or "2001:db8::/64"
	// +required
	// +kubebuilder:validation:MinLength=1
	Cidr string `json:"cidr"`
	// port represents the port on the given protocol
	// +required
	// +kubebuilder:validation:Minimum=1
	Port int32 `json:"port"`
	// protocol represents the protocol (TCP, UDP, or SCTP) which traffic must match. If not specified, this field defaults to TCP.
	// +kubebuilder:default=TCP
	// +kubebuilder:validation:Enum=TCP;UDP;SCTP
	Protocol string `json:"protocol"`
}

// NetworkCheckConfigurationSpec defines a URL to be checked from within that namespace
type NetworkCheckConfigurationSpec struct {
	// +required
	// +kubebuilder:validation:MinLength=1
	Url string `json:"url"`
}

// SelfServiceNamespaceStatus defines the observed state of SelfServiceNamespace.
type SelfServiceNamespaceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// SelfServiceNamespace is the Schema for the selfservicenamespaces API.
// +kubebuilder:printcolumn:name="ChecksEnabled",type="boolean",JSONPath=".spec.networkChecksEnabled"
// +kubebuilder:printcolumn:name="Status",type="integer",JSONPath=".status"
type SelfServiceNamespace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SelfServiceNamespaceSpec   `json:"spec,omitempty"`
	Status SelfServiceNamespaceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SelfServiceNamespaceList contains a list of SelfServiceNamespace.
type SelfServiceNamespaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SelfServiceNamespace `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SelfServiceNamespace{}, &SelfServiceNamespaceList{})
}
