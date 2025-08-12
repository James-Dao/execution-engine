package handler

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	dacv1alpha1 "github.com/James-Dao/execution-engine/api/v1alpha1"
	"github.com/James-Dao/execution-engine/client/http"
	"github.com/James-Dao/execution-engine/client/k8s"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DataDescriptorHandler struct {
	K8sServices k8s.Services
	EventsCli   k8s.Event
	Kubeclient  client.Client
	Logger      logr.Logger
	HTTPClient  *http.APIClient
}

// SourceStatusResult contains the result of checking a data source status
type SourceStatusResult struct {
	Name         string
	Phase        string
	LastSyncTime metav1.Time
	Records      int64
	TaskID       string
	Error        error
}

func (h *DataDescriptorHandler) Do(ctx context.Context, dd *dacv1alpha1.DataDescriptor) error {
	logger := h.Logger.WithValues("namespace", dd.Namespace, "name", dd.Name)
	logger.Info("Processing DataDescriptor")

	// handle dd logic
	taskIDs, err := h.handleDD(ctx, dd)
	if err != nil {
		return fmt.Errorf("failed to handle data descriptor: %w", err)
	}

	logger.Info("task IDs", "taskIDs", taskIDs)

	// handle dd status
	err = h.handleDDStatus(ctx, dd, taskIDs)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

func (h *DataDescriptorHandler) handleDD(ctx context.Context, dd *dacv1alpha1.DataDescriptor) (map[string]string, error) {
	logger := h.Logger.WithValues("namespace", dd.Namespace, "name", dd.Name)
	logger.Info("Processing DataDescriptor sources")

	taskIDs := make(map[string]string)

	for _, source := range dd.Spec.Sources {

		// 检查是否已有有效任务
		if existingStatus := h.getExistingSourceStatus(dd, source.Name); existingStatus != nil {
			if existingStatus.TaskID != "" {
				logger.Info("Skipping source with existing task",
					"source", source.Name,
					"taskID", existingStatus.TaskID)
				taskIDs[source.Name] = existingStatus.TaskID
				continue
			}
		}

		// 构建符合API要求的请求数据结构
		requestData := map[string]interface{}{
			"data": map[string]interface{}{
				"source": map[string]interface{}{
					"type":     source.Type,
					"name":     source.Name,
					"metadata": source.Metadata,
				},
				"descriptor": map[string]interface{}{
					"name":      dd.Name,
					"namespace": dd.Namespace,
				},
				"extract":        source.Extract,
				"processing":     source.Processing,
				"classification": source.Classification,
			},
		}

		taskID, err := h.HTTPClient.TriggerTask(ctx, requestData)
		if err != nil {
			logger.Error(err, "Failed to trigger task for source",
				"source", source.Name,
				"requestData", requestData)
			return nil, fmt.Errorf("failed to trigger task for source %s: %w", source.Name, err)
		}

		logger.Info("Successfully triggered task",
			"source", source.Name,
			"taskID", taskID,
			"requestData", requestData)
		taskIDs[source.Name] = taskID
	}

	return taskIDs, nil
}

// 获取已有的 SourceStatus
func (h *DataDescriptorHandler) getExistingSourceStatus(dd *dacv1alpha1.DataDescriptor, name string) *dacv1alpha1.SourceStatus {
	for _, status := range dd.Status.SourceStatuses {
		if status.Name == name {
			return &status
		}
	}
	return nil
}

func (h *DataDescriptorHandler) handleDDStatus(ctx context.Context, dd *dacv1alpha1.DataDescriptor, taskIDs map[string]string) error {
	logger := h.Logger.WithValues("namespace", dd.Namespace, "name", dd.Name)
	logger.Info("Processing DataDescriptor Status")

	// Save the original status for comparison later
	originalStatus := dd.Status.DeepCopy()

	// Initialize Status if needed
	newStatus := dacv1alpha1.DataDescriptorStatus{
		SourceStatuses: make([]dacv1alpha1.SourceStatus, 0),
		Conditions:     make([]dacv1alpha1.Condition, 0),
	}

	// Copy existing conditions if they exist
	if dd.Status.Conditions != nil {
		newStatus.Conditions = append(newStatus.Conditions, dd.Status.Conditions...)
	}

	// If this is a new resource, set initial condition
	if dd.Status.OverallPhase == "" {
		newStatus.SetCreateCondition("Initializing data descriptor")
	}

	// Check data source statuses
	sourceStatuses := make([]dacv1alpha1.SourceStatus, len(dd.Spec.Sources))
	allHealthy := true
	var aggregatedErrors []error

	var aggregatedNotReady []string

	for i, source := range dd.Spec.Sources {
		task := ""
		if taskID, exists := taskIDs[source.Name]; exists {
			task = taskID
		}

		status := h.checkSourceStatus(ctx, source, task)

		sourceStatuses[i] = dacv1alpha1.SourceStatus{
			Name:         source.Name,
			Phase:        status.Phase,
			LastSyncTime: status.LastSyncTime,
			Records:      status.Records,
			TaskID:       status.TaskID,
		}

		if status.Error != nil || status.Phase != "Ready" {
			allHealthy = false
			if status.Error != nil {
				logger.Error(
					status.Error,
					"Data source status check failed",
					"source", source.Name,
					"phase", status.Phase,
				)
				aggregatedErrors = append(aggregatedErrors, fmt.Errorf("data source %s error: %w", source.Name, status.Error))
			} else {
				logger.Info(
					"Data source or data source task is not ready",
					"source", source.Name,
					"phase", status.Phase,
				)
				aggregatedNotReady = append(aggregatedNotReady, source.Name)
			}
		} else {
			h.EventsCli.Normal(dd, "TaskTriggered", fmt.Sprintf("Task %s Completed for data source %s", status.TaskID, source.Name))
		}
	}

	// Update the new status
	newStatus.SourceStatuses = sourceStatuses
	if allHealthy {
		newStatus.OverallPhase = "Ready"
		c := dacv1alpha1.NewCondition(dacv1alpha1.ConditionAvailable, corev1.ConditionTrue, "Available", "All data sources healthy.")
		newStatus.SetDataDescriptorCondition(*c)
		h.EventsCli.Normal(dd, "AllSourcesHealthy", "All data sources healthy and tasks triggered and Completed.")
	} else {
		newStatus.OverallPhase = "NotReady"
		errorMsg := fmt.Sprintf("%d data sources task not completed, %d data sources have issues ", len(aggregatedNotReady), len(aggregatedErrors))
		c := dacv1alpha1.NewCondition(dacv1alpha1.ConditionNotReady, corev1.ConditionTrue, "NotReady", errorMsg)
		newStatus.SetDataDescriptorCondition(*c)
		h.EventsCli.Warning(dd, "SomeSourcesTaskErrorOrNotReady", errorMsg)
	}

	// Compare the new status with original, ignoring time fields
	if !h.isStatusEqualIgnoringTime(*originalStatus, newStatus) {
		// Update the status in the original object
		dd.Status = newStatus

		// Submit status update
		if err := h.Kubeclient.Status().Update(ctx, dd); err != nil {
			logger.Error(err, "Failed to update status")
			return fmt.Errorf("status update failed: %w", err)
		}
		logger.Info("Status updated successfully")
	} else {
		logger.Info("Status unchanged, skipping update")
	}

	// Return aggregated errors (if any)
	if len(aggregatedErrors) > 0 {
		return fmt.Errorf("%d errors: %v", len(aggregatedErrors), aggregatedErrors)
	}
	return nil
}

// isStatusEqualIgnoringTime compares two DataDescriptorStatus objects while ignoring time fields
func (h *DataDescriptorHandler) isStatusEqualIgnoringTime(oldStatus, newStatus dacv1alpha1.DataDescriptorStatus) bool {
	// Compare OverallPhase
	if oldStatus.OverallPhase != newStatus.OverallPhase {
		return false
	}

	// Compare Conditions (ignoring LastTransitionTime)
	if len(oldStatus.Conditions) != len(newStatus.Conditions) {
		return false
	}
	for i := range oldStatus.Conditions {
		oldCond := oldStatus.Conditions[i]
		newCond := newStatus.Conditions[i]
		if oldCond.Type != newCond.Type ||
			oldCond.Status != newCond.Status ||
			oldCond.Reason != newCond.Reason ||
			oldCond.Message != newCond.Message {
			return false
		}
	}

	// Compare SourceStatuses (ignoring LastSyncTime)
	if len(oldStatus.SourceStatuses) != len(newStatus.SourceStatuses) {
		return false
	}
	for i := range oldStatus.SourceStatuses {
		oldSource := oldStatus.SourceStatuses[i]
		newSource := newStatus.SourceStatuses[i]
		if oldSource.Name != newSource.Name ||
			oldSource.Phase != newSource.Phase ||
			oldSource.Records != newSource.Records ||
			oldSource.TaskID != newSource.TaskID {
			return false
		}
	}

	// Compare ConsumedBy
	if !reflect.DeepEqual(oldStatus.ConsumedBy, newStatus.ConsumedBy) {
		return false
	}

	return true
}

func (h *DataDescriptorHandler) checkSourceStatus(ctx context.Context, source dacv1alpha1.DataSource, taskID string) SourceStatusResult {
	// 如果有 TaskID，检查任务状态
	if taskID != "" {
		statusResp, err := h.HTTPClient.GetTaskStatus(ctx, taskID)
		if err != nil {
			return SourceStatusResult{
				Name:   source.Name,
				Phase:  "Error",
				TaskID: taskID,
				Error:  fmt.Errorf("failed to check task status: %w", err),
			}
		}

		// 根据任务状态返回结果
		switch strings.ToUpper(statusResp.Status) { // Handle case insensitivity
		case "SUCCESS":
			return SourceStatusResult{
				Name:   source.Name,
				Phase:  "Ready",
				TaskID: taskID,
			}
		case "FAILED":
			return SourceStatusResult{
				Name:  source.Name,
				Phase: "FAILED",
				Error: fmt.Errorf("task %s failed: %v", taskID, statusResp.Result),
			}
		case "PENDING":
			return SourceStatusResult{
				Name:   source.Name,
				Phase:  "PENDING",
				TaskID: taskID,
			}
		default: // running/queued or other statuses
			return SourceStatusResult{
				Name:   source.Name,
				Phase:  "OTHERS",
				TaskID: taskID,
			}
		}
	}

	// 无 TaskID 时的默认检查数据源的连通性
	return h.checkDataSourceConnectivity(ctx, source)
}

// checkDataSourceConnectivity checks the Connectivy of a single data source
func (h *DataDescriptorHandler) checkDataSourceConnectivity(ctx context.Context, source dacv1alpha1.DataSource) SourceStatusResult {
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
