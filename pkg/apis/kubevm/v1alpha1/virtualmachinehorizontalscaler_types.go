package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type VirtualMachineHorizontalScalerTargetRef struct {

	// +kubebuilder:validation:Enum={"kubevm.io/v1alpha1"}
	ApiVersion string `json:"apiVersion"`

	// +kubebuilder:validation:Enum={"VirtualMachineDeployment"}
	Kind string `json:"kind"`

	Name string `json:"name"`
}

type VirtualMachineHorizontalScalerMetricResourceTarget struct {
	// +kubebuilder:validation:Enum={"Utilization"}
	Type string `json:"type"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	AverageUtilization int32 `json:"averageUtilization"`
}

type VirtualMachineHorizontalScalerMetricResource struct {
	// +kubebuilder:validation:Enum={"cpu"}
	Name string `json:"name"`

	Resource VirtualMachineHorizontalScalerMetricResourceTarget `json:"target"`
}

type VirtualMachineHorizontalScalerMetric struct {
	// +kubebuilder:validation:Enum={"Resource"}
	Type string `json:"type"`

	Resource VirtualMachineHorizontalScalerMetricResource `json:"resource"`
}

// VirtualMachineHorizontalScalerSpec defines the desired state of VirtualMachineHorizontalScaler
type VirtualMachineHorizontalScalerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	ScaleTargetRef VirtualMachineHorizontalScalerTargetRef `json:"scaleTargetRef"`

	// +kubebuilder:validation:Minimum=1
	MinReplicas int32 `json:"minReplicas"`

	// +kubebuilder:validation:Minimum=1
	MaxReplicas int32 `json:"maxReplicas"`

	// +kubebuilder:validation:MinItems=1
	Metrics []VirtualMachineHorizontalScalerMetric `json:"metrics"`
}

// VirtualMachineHorizontalScalerStatus defines the observed state of VirtualMachineHorizontalScaler
type VirtualMachineHorizontalScalerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualMachineHorizontalScaler is the Schema for the virtualmachinehorizontalscalers API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=virtualmachinehorizontalscalers,scope=Namespaced
type VirtualMachineHorizontalScaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineHorizontalScalerSpec   `json:"spec,omitempty"`
	Status VirtualMachineHorizontalScalerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualMachineHorizontalScalerList contains a list of VirtualMachineHorizontalScaler
type VirtualMachineHorizontalScalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineHorizontalScaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualMachineHorizontalScaler{}, &VirtualMachineHorizontalScalerList{})
}
