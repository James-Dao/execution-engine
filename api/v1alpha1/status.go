package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

// ConditionType defines the condition that the RF can have
type ConditionType string

const (
	DataDescriptionConditionAdd ConditionType = "Added"
	ConditionAvailable          ConditionType = "Available"
	ConditionHealthy            ConditionType = "Healthy"
	ConditionRunning            ConditionType = "Running"
	ConditionCreating           ConditionType = "Creating"
	ConditionRecovering         ConditionType = "Recovering"
	ConditionUpdating           ConditionType = "Updating"
	ConditionFailed             ConditionType = "Failed"
)

// Condition saves the state information of the cr
type Condition struct {
	// Status of cr condition.
	Type ConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime string `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}
