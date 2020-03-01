package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VirtualMachineInstanceSpec defines the desired state of VirtualMachineInstance
type VirtualMachineInstanceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// +kubebuilder:validation:MinLength=1
	ImageName string `json:"image"`

	// +kubebuilder:validation:MinLength=1
	NetworkName string `json:"network"`

	// +kubebuilder:validation:Enum={"Started","Stopped","Paused"}
	Status string `json:"status"`
}

// VirtualMachineInstanceStatus defines the observed state of VirtualMachineInstance
type VirtualMachineInstanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	ID     string `json:"id,omitempty"`
	Status string `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualMachineInstance is the Schema for the virtualmachineinstances API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=virtualmachineinstances,scope=Namespaced
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.id",description="VM identifier"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".spec.Status",description="Desired VM Running Status"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type VirtualMachineInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineInstanceSpec   `json:"spec,omitempty"`
	Status VirtualMachineInstanceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualMachineInstanceList contains a list of VirtualMachineInstance
type VirtualMachineInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualMachineInstance{}, &VirtualMachineInstanceList{})
}
