package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/James-Dao/execution-engine/client/k8s"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dacv1alpha1 "github.com/James-Dao/execution-engine/api/v1alpha1"
)

type DataDescriptorHandler struct {
	K8sServices k8s.Services
	EventsCli   k8s.Event
	Kubeclient  client.Client
	Logger      logr.Logger
}

// SourceStatusResult contains the result of checking a data source status
type SourceStatusResult struct {
	Name         string
	Phase        string
	LastSyncTime metav1.Time
	Records      int64
	Error        error
}

// Do ensures the DataDescriptor is in the desired state and updates its status
func (h *DataDescriptorHandler) Do(ctx context.Context, dd *dacv1alpha1.DataDescriptor) error {
	logger := h.Logger.WithValues("namespace", dd.Namespace, "name", dd.Name)
	logger.Info("Start processing DataDescriptor")

	// Initialize status fields if needed
	if dd.Status.SourceStatuses == nil {
		dd.Status.SourceStatuses = make([]dacv1alpha1.SourceStatus, 0)
	}

	// Check all data source statuses
	sourceStatuses := make([]dacv1alpha1.SourceStatus, len(dd.Spec.Sources))
	allHealthy := true
	var aggregatedErrors []error

	for i, source := range dd.Spec.Sources {
		status := h.checkSourceStatus(ctx, source)
		sourceStatus := dacv1alpha1.SourceStatus{
			Name:         source.Name,
			Phase:        status.Phase,
			LastSyncTime: status.LastSyncTime,
			Records:      status.Records,
		}
		sourceStatuses[i] = sourceStatus

		if status.Error != nil {
			logger.Error(status.Error, "Data source status check failed", "source", source.Name)
			allHealthy = false
			aggregatedErrors = append(aggregatedErrors, fmt.Errorf("data source %s error: %w", source.Name, status.Error))

			// Update status to Error
			sourceStatuses[i].Phase = "Error"

			h.EventsCli.Warning(dd, "SourceCheckFailed",
				fmt.Sprintf("Data source %s check failed: %v", source.Name, status.Error))
		} else if status.Phase != "Ready" {
			allHealthy = false
			aggregatedErrors = append(aggregatedErrors, fmt.Errorf("data source %s unhealthy: %s", source.Name, status.Phase))

			h.EventsCli.Warning(dd, "SourceUnhealthy",
				fmt.Sprintf("Data source %s unhealthy: %s", source.Name, status.Phase))
		}
	}

	// Update DataDescriptor status
	dd.Status.SourceStatuses = sourceStatuses
	if allHealthy {
		dd.Status.OverallPhase = "Ready"
		h.EventsCli.Normal(dd, "AllSourcesHealthy", "All data sources healthy")
	} else {
		dd.Status.OverallPhase = "Error"
		errorMsg := fmt.Sprintf("%d data sources have issues", len(aggregatedErrors))
		h.EventsCli.Warning(dd, "SomeSourcesUnhealthy", errorMsg)
	}

	// Attempt to update status
	if err := h.Kubeclient.Status().Update(ctx, dd); err != nil {
		logger.Error(err, "Failed to update DataDescriptor status")
		aggregatedErrors = append(aggregatedErrors, fmt.Errorf("status update failed: %w", err))
	}

	// Return aggregated errors if any
	if len(aggregatedErrors) > 0 {
		return fmt.Errorf("encountered %d errors: %v", len(aggregatedErrors), aggregatedErrors)
	}

	// add code to call celery http server to send task for dd.
	// todo

	return nil
}

// checkSourceStatus checks the status of a single data source
func (h *DataDescriptorHandler) checkSourceStatus(ctx context.Context, source dacv1alpha1.DataSource) SourceStatusResult {
	// Validate data source configuration
	if source.Name == "" {
		return SourceStatusResult{
			Name:  source.Name,
			Phase: "Invalid",
			Error: fmt.Errorf("data source name cannot be empty"),
		}
	}

	switch source.Type {
	case dacv1alpha1.DataSourceRedis:
		return h.checkRedisStatus(ctx, source)
	case dacv1alpha1.DataSourceMySQL:
		return h.checkMySQLStatus(ctx, source)
	case dacv1alpha1.DataSourceMinIO:
		return h.checkMinIOStatus(ctx, source)
	default:
		return SourceStatusResult{
			Name:  source.Name,
			Phase: "Unknown",
			Error: fmt.Errorf("unknown data source type: %s", source.Type),
		}
	}
}

// checkRedisStatus checks Redis data source status
func (h *DataDescriptorHandler) checkRedisStatus(ctx context.Context, source dacv1alpha1.DataSource) SourceStatusResult {
	// In a real implementation, we would:
	// 1. Get connection details from source.Metadata
	// 2. Connect to Redis
	// 3. Check health and get stats

	// Mock implementation
	endpoint, ok := source.Metadata["endpoint"]
	if !ok || endpoint == "" {
		return SourceStatusResult{
			Name:  source.Name,
			Phase: "Invalid",
			Error: fmt.Errorf("Redis endpoint not configured in metadata"),
		}
	}

	return SourceStatusResult{
		Name:         source.Name,
		Phase:        "Ready",
		LastSyncTime: metav1.NewTime(time.Now()),
		Records:      1000,
	}
}

// checkMySQLStatus checks MySQL data source status
func (h *DataDescriptorHandler) checkMySQLStatus(ctx context.Context, source dacv1alpha1.DataSource) SourceStatusResult {
	// Validate configuration
	endpoint, ok := source.Metadata["endpoint"]
	if !ok || endpoint == "" {
		return SourceStatusResult{
			Name:  source.Name,
			Phase: "Invalid",
			Error: fmt.Errorf("MySQL endpoint not configured in metadata"),
		}
	}

	return SourceStatusResult{
		Name:         source.Name,
		Phase:        "Ready",
		LastSyncTime: metav1.NewTime(time.Now()),
		Records:      5000,
	}
}

// checkMinIOStatus checks MinIO data source status
func (h *DataDescriptorHandler) checkMinIOStatus(ctx context.Context, source dacv1alpha1.DataSource) SourceStatusResult {
	// Validate configuration
	endpoint, ok := source.Metadata["endpoint"]
	if !ok || endpoint == "" {
		return SourceStatusResult{
			Name:  source.Name,
			Phase: "Invalid",
			Error: fmt.Errorf("MinIO endpoint not configured in metadata"),
		}
	}

	return SourceStatusResult{
		Name:         source.Name,
		Phase:        "Ready",
		LastSyncTime: metav1.NewTime(time.Now()),
		Records:      200,
	}
}

// GetSourceStatus gets the status of a specific data source (for controller use)
func (h *DataDescriptorHandler) GetSourceStatus(ctx context.Context, source dacv1alpha1.DataSource) dacv1alpha1.SourceStatus {
	result := h.checkSourceStatus(ctx, source)
	return dacv1alpha1.SourceStatus{
		Name:         source.Name,
		Phase:        result.Phase,
		LastSyncTime: result.LastSyncTime,
		Records:      result.Records,
	}
}
