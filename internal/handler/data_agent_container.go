package handler

import (
	"context"

	"fmt"
	dacv1alpha1 "github.com/James-Dao/execution-engine/api/v1alpha1"
	"github.com/James-Dao/execution-engine/client/k8s"
	"github.com/James-Dao/execution-engine/internal/generator"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// DataAgentContainerHandler handles the reconciliation logic for DataAgentContainer resources.
type DataAgentContainerHandler struct {
	K8sServices  k8s.Services
	EventsCli    k8s.Event
	Kubeclient   client.Client
	Logger       logr.Logger
	DACGenerator *generator.DataAgentContainerGenerator
}

// AgentStatusResult contains the result of checking agent status
type AgentStatusResult struct {
	Endpoint              dacv1alpha1.Endpoint
	ActiveDataDescriptors []dacv1alpha1.ActiveDataDescriptor
	Phase                 string
	Error                 error
}

func (h *DataAgentContainerHandler) Do(ctx context.Context, dac *dacv1alpha1.DataAgentContainer) error {
	logger := h.Logger.WithValues("namespace", dac.Namespace, "name", dac.Name)
	logger.Info("Processing DataAgentContainer")

	// handle dac logic
	err := h.handleDAC(ctx, dac)
	if err != nil {
		return err
	}

	// handle dac status
	err = h.handleDACStatus(ctx, dac)
	if err != nil {
		return err
	}

	return nil
}

func (h *DataAgentContainerHandler) handleDAC(ctx context.Context, dac *dacv1alpha1.DataAgentContainer) error {
	logger := h.Logger.WithValues("namespace", dac.Namespace, "name", dac.Name)
	logger.Info("Processing DataAgentContainer Logic")

	// todo check service and deployment is exist

	serviceExist := h.checkK8SServiceExist(dac)

	deploymentExist := h.checkK8SDeploymentExist(dac)

	if serviceExist && deploymentExist {
		return nil
	}

	err := h.DACGenerator.Do(ctx, dac)
	if err != nil {
		return err
	}

	return nil
}

// Do processes the DataAgentContainer resource and updates its status.
func (h *DataAgentContainerHandler) handleDACStatus(ctx context.Context, dac *dacv1alpha1.DataAgentContainer) error {
	logger := h.Logger.WithValues("namespace", dac.Namespace, "name", dac.Name)
	logger.Info("Processing DataAgentContainer Status")

	// Initialize status fields if needed
	if dac.Status.Conditions == nil {
		dac.Status.Conditions = make([]dacv1alpha1.Condition, 0)
	}
	if dac.Status.ActiveDataDescriptors == nil {
		dac.Status.ActiveDataDescriptors = make([]dacv1alpha1.ActiveDataDescriptor, 0)
	}

	// Set creating condition if this is a new resource
	if dac.Status.Endpoint.Address == "" {
		dac.Status.SetCreateCondition("Initializing DataAgentContainer")
	}

	// Check agent status
	status := h.checkAgentStatus(ctx, dac)

	// Update status based on check results
	if status.Error != nil {
		// Handle error state
		logger.Error(status.Error, "Agent status check failed")
		h.EventsCli.Warning(dac, "AgentCheckFailed", fmt.Sprintf("Agent check failed: %v", status.Error))

		c := dacv1alpha1.NewCondition(
			dacv1alpha1.ConditionFailed,
			corev1.ConditionTrue,
			"CheckFailed",
			fmt.Sprintf("Agent status check failed: %v", status.Error),
		)
		dac.Status.SetDataAgentContainerCondition(*c)
	} else {
		// Update successful status
		dac.Status.Endpoint = status.Endpoint
		dac.Status.ActiveDataDescriptors = status.ActiveDataDescriptors

		c := dacv1alpha1.NewCondition(
			dacv1alpha1.ConditionAvailable,
			corev1.ConditionTrue,
			"Healthy",
			"Agent is healthy and ready",
		)
		dac.Status.SetDataAgentContainerCondition(*c)
		h.EventsCli.Normal(dac, "AgentHealthy", "Agent is healthy")
	}

	// Sort conditions by time
	dac.Status.DescConditionsByTime()

	// Update the status in Kubernetes
	if err := h.Kubeclient.Status().Update(ctx, dac); err != nil {
		logger.Error(err, "Failed to update DataAgentContainer status")
		return fmt.Errorf("failed to update status: %w", err)
	}

	if status.Error != nil {
		return status.Error
	}

	return nil
}

func (h *DataAgentContainerHandler) checkK8SServiceExist(dac *dacv1alpha1.DataAgentContainer) bool {
	serviceName := h.DACGenerator.GenerateDataAgentContainerServiceName(dac)
	if _, err := h.K8sServices.GetService(dac.Namespace, serviceName); err != nil {
		if errors.IsNotFound(err) {
			return false
		}
		return false
	}
	return true
}

func (h *DataAgentContainerHandler) checkK8SDeploymentExist(dac *dacv1alpha1.DataAgentContainer) bool {
	deploymentName := h.DACGenerator.GenerateDataAgentContainerDeploymentName(dac)
	_, err := h.K8sServices.GetDeployment(dac.Namespace, deploymentName)
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			return false
		}
		return false
	}

	return true
}

// checkAgentStatus checks the current status of the agent.
func (h *DataAgentContainerHandler) checkAgentStatus(ctx context.Context, dac *dacv1alpha1.DataAgentContainer) AgentStatusResult {
	// In a real implementation, this would:
	// 1. Verify agent endpoint connectivity
	// 2. Check active data descriptors
	// 3. Validate agent capabilities

	// Mock implementation - assume agent is ready
	return AgentStatusResult{
		Endpoint: dacv1alpha1.Endpoint{
			Address:  "agent-service.default.svc.cluster.local",
			Port:     8080,
			Protocol: "http",
		},
		ActiveDataDescriptors: []dacv1alpha1.ActiveDataDescriptor{
			{
				Name:       "example-descriptor",
				Namespace:  "default",
				LastSynced: time.Now().Format(time.RFC3339),
			},
		},
		Phase: "Ready",
	}
}
