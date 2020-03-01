package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VirtualMachineDeploymentSpec defines the desired state of VirtualMachineDeployment
type VirtualMachineDeploymentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// +kubebuilder:validation:Minimum=1
	Replicas int32 `json:"replicas"`

	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:UniqueItems=false
	Template VirtualMachineInstanceSpec `json:"template"`
}

// VirtualMachineDeploymentStatus defines the observed state of VirtualMachineDeployment
type VirtualMachineDeploymentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	Replicas int32 `json:"replicas"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualMachineDeployment is the Schema for the virtualmachinedeployments API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=virtualmachinedeployments,scope=Namespaced
type VirtualMachineDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineDeploymentSpec   `json:"spec,omitempty"`
	Status VirtualMachineDeploymentStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualMachineDeploymentList contains a list of VirtualMachineDeployment
type VirtualMachineDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualMachineDeployment{}, &VirtualMachineDeploymentList{})
}
