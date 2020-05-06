package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OvsDpdkConfigSpec defines the desired state of OvsDpdkConfig
type OvsDpdkConfigSpec struct {
	// Nodes on which OvS-DPDK should run
	NodeSelectorLabels map[string]string `json:"nodeSelectorLabels"`
	// Node specific configuration
	// +optional
	NodeConfig NodeConfig `json:"nodeConfig,omitempty"`
	// Interfaces to be used for OvS-DPDK configuration
	InterfaceConfig []InterfaceConfig `json:"interfaceConfig"`
}

type NodeConfig struct {
	HugePage1G    string `json:"hugepage1g,omitempty"`
	PMDCount      uint32 `json:"pmdCount,omitempty"`
	MemoryChannel uint32 `json:"memoryChannel,omitempty"`
}

type InterfaceConfig struct {
	Bridge      string      `json:"bridge"`
	Bond        bool        `json:"bond,omitempty"`
	BondMode    string      `json:"bondMonde,omitempty"`
	MTU         uint32      `json:"mtu,omitempty"`
	Queues      uint32      `json:"queues,omitempty"`
	Driver      string      `json:"driver,omitempty"`
	NicSelector NicSelector `json:"nicSelector"`
}

type NicSelector struct {
	Devices []string `json:"devices,omitempty"`
	IfNames []string `json:"ifNames,omitempty"`
}

// OvsDpdkConfigStatus defines the observed state of OvsDpdkConfig
type OvsDpdkConfigStatus struct {
	// List of nodes on which OvS-DPDK is enabled (is it useful?)
	Nodes []string `json:"nodes"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OvsDpdkConfig is the Schema for the ovsdpdkconfigs API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=ovsdpdkconfigs,scope=Namespaced
type OvsDpdkConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OvsDpdkConfigSpec   `json:"spec,omitempty"`
	Status OvsDpdkConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OvsDpdkConfigList contains a list of OvsDpdkConfig
type OvsDpdkConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OvsDpdkConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OvsDpdkConfig{}, &OvsDpdkConfigList{})
}
