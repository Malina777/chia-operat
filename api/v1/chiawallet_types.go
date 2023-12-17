/*
Copyright 2023 Chia Network Inc.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ChiaWalletSpec defines the desired state of ChiaWallet
type ChiaWalletSpec struct {
	CommonSpec `json:",inline"`

	// ChiaConfig defines the configuration options available to Chia component containers
	ChiaConfig ChiaWalletConfigSpec `json:"chia"`
}

// ChiaWalletConfigSpec defines the desired state of Chia component configuration
type ChiaWalletConfigSpec struct {
	CommonChiaConfigSpec `json:",inline"`

	// SecretKeySpec defines the k8s Secret name and key for a Chia mnemonic
	SecretKeySpec ChiaKeysSpec `json:"secretKey"`

	// FullNodePeer defines the farmer's full_node peer in host:port format.
	// In Kubernetes this is likely to be <node service name>.<namespace>.svc.cluster.local:8555
	FullNodePeer string `json:"fullNodePeer"`
}

// ChiaWalletStatus defines the observed state of ChiaWallet
type ChiaWalletStatus struct {
	// Ready says whether the node is ready, this should be true when the node statefulset is in the target namespace
	// +kubebuilder:default=false
	Ready bool `json:"ready,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ChiaWallet is the Schema for the chiawallets API
type ChiaWallet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChiaWalletSpec   `json:"spec,omitempty"`
	Status ChiaWalletStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ChiaWalletList contains a list of ChiaWallet
type ChiaWalletList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ChiaWallet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ChiaWallet{}, &ChiaWalletList{})
}
