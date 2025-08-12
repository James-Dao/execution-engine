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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DataPolicy defines how data sources should be selected
type DataPolicy struct {
	// Selector for data sources by name
	SourceNameSelector []string `json:"sourceNameSelector,omitempty"`

	// Selector for data sources by classification
	// MatchClassifications []ClassificationMatch `json:"matchClassifications,omitempty"`
}

// ClassificationMatch defines how to match data sources by their classification
type ClassificationMatch struct {
	// Domain of the data
	Domain string `json:"domain,omitempty"`

	// Category of the data (can be string or array of strings)
	Category []string `json:"category,omitempty"`

	// Subcategory of the data
	Subcategory string `json:"subcategory,omitempty"`

	// Matching policy (exact/prefix/regex)
	MatchPolicy string `json:"matchPolicy,omitempty"`
}

// AgentCard defines the agent's metadata and capabilities
type AgentCard struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	URL         string       `json:"url"`
	Skills      []AgentSkill `json:"skills"`
}

// AgentSkill defines a specific skill the agent provides
type AgentSkill struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags,omitempty"`
	Examples    []string  `json:"examples,omitempty"`
	InputModes  *[]string `json:"inputModes,omitempty"`
	OutputModes *[]string `json:"outputModes,omitempty"`
}

// ModelSpec defines the LLM and embedding models to use
type ModelSpec struct {
	Provider  string `json:"provider"`
	LLM       string `json:"llm"`
	Embedding string `json:"embedding"`
}

// DataAgentContainerSpec defines the desired state of DataAgentContainer
type DataAgentContainerSpec struct {
	DataPolicy DataPolicy `json:"dataPolicy"`
	AgentCard  AgentCard  `json:"agentCard"`
	Model      ModelSpec  `json:"model"`
}

// ActiveDataDescriptor tracks which data descriptors are being used
type ActiveDataDescriptor struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	LastSynced string `json:"lastSynced"`
}

// Endpoint defines how to connect to the agent
type Endpoint struct {
	Address  string `json:"address"`
	Port     int32  `json:"port"`
	Protocol string `json:"protocol"`
}

// DataAgentContainerStatus defines the observed state of DataAgentContainer
type DataAgentContainerStatus struct {
	ActiveDataDescriptors []ActiveDataDescriptor `json:"activeDataDescriptors,omitempty"`
	Endpoint              Endpoint               `json:"endpoint,omitempty"`
	Conditions            []Condition            `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DataAgentContainer is the Schema for the dataagentcontainers API.
type DataAgentContainer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataAgentContainerSpec   `json:"spec,omitempty"`
	Status DataAgentContainerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DataAgentContainerList contains a list of DataAgentContainer.
type DataAgentContainerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataAgentContainer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DataAgentContainer{}, &DataAgentContainerList{})
}
