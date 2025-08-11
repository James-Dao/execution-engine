package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/James-Dao/execution-engine/client/k8s"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
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

func (h *DataDescriptorHandler) Do(ctx context.Context, dd *dacv1alpha1.DataDescriptor) error {
	logger := h.Logger.WithValues("namespace", dd.Namespace, "name", dd.Name)
	logger.Info("Processing DataDescriptor")

	// handle dd logic
	err := h.handleDD(ctx, dd)
	if err != nil {
		return err
	}

	// handle dd status
	err = h.handleDDStatus(ctx, dd)
	if err != nil {
		return err
	}

	return nil
}

func (h *DataDescriptorHandler) handleDD(ctx context.Context, dd *dacv1alpha1.DataDescriptor) error {
	logger := h.Logger.WithValues("namespace", dd.Namespace, "name", dd.Name)
	logger.Info("Processing DataDescriptor Logic")

	return nil
}

func (h *DataDescriptorHandler) handleDDStatus(ctx context.Context, dd *dacv1alpha1.DataDescriptor) error {
	logger := h.Logger.WithValues("namespace", dd.Namespace, "name", dd.Name)
	logger.Info("Processing DataDescriptor Status")

	// 初始化 Status 字段
	if dd.Status.SourceStatuses == nil {
		dd.Status.SourceStatuses = make([]dacv1alpha1.SourceStatus, 0)
	}
	if dd.Status.Conditions == nil {
		dd.Status.Conditions = make([]dacv1alpha1.Condition, 0)
	}

	// 如果是新资源，设置初始化 Condition
	if dd.Status.OverallPhase == "" {
		dd.Status.SetCreateCondition("Initializing data descriptor")
	}

	// 检查数据源状态
	sourceStatuses := make([]dacv1alpha1.SourceStatus, len(dd.Spec.Sources))
	allHealthy := true
	var aggregatedErrors []error

	for i, source := range dd.Spec.Sources {
		status := h.checkSourceStatus(ctx, source)
		sourceStatuses[i] = dacv1alpha1.SourceStatus{
			Name:         source.Name,
			Phase:        status.Phase,
			LastSyncTime: status.LastSyncTime,
			Records:      status.Records,
		}

		if status.Error != nil || status.Phase != "Ready" {
			allHealthy = false
			// 1. 记录错误日志
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
					"Data source is not ready",
					"source", source.Name,
					"phase", status.Phase,
				)
				aggregatedErrors = append(aggregatedErrors, fmt.Errorf("data source %s is unhealthy (phase: %s)", source.Name, status.Phase))
			}

			// 2. 标记数据源状态为 Error（如果未设置）
			if sourceStatuses[i].Phase != "Error" {
				sourceStatuses[i].Phase = "Error"
			}

			// 3. 触发 Kubernetes 事件（Warning 级别）
			eventMsg := fmt.Sprintf("Data source %s check failed", source.Name)
			if status.Error != nil {
				eventMsg = fmt.Sprintf("%s: %v", eventMsg, status.Error)
			} else {
				eventMsg = fmt.Sprintf("%s: phase=%s", eventMsg, status.Phase)
			}
			h.EventsCli.Warning(dd, "SourceUnhealthy", eventMsg)
		}
	}

	// 更新 OverallPhase 和 Conditions
	dd.Status.SourceStatuses = sourceStatuses
	if allHealthy {
		dd.Status.OverallPhase = "Ready"
		c := dacv1alpha1.NewCondition(dacv1alpha1.ConditionAvailable, corev1.ConditionTrue, "Available", "All data sources healthy")
		dd.Status.SetDataDescriptorCondition(*c)
		h.EventsCli.Normal(dd, "AllSourcesHealthy", "All data sources healthy")
	} else {
		dd.Status.OverallPhase = "Error"
		errorMsg := fmt.Sprintf("%d data sources have issues", len(aggregatedErrors))
		c := dacv1alpha1.NewCondition(dacv1alpha1.ConditionFailed, corev1.ConditionTrue, "Degraded", errorMsg)
		dd.Status.SetDataDescriptorCondition(*c)
		h.EventsCli.Warning(dd, "SomeSourcesUnhealthy", errorMsg)
	}

	// 提交状态更新
	if err := h.Kubeclient.Status().Update(ctx, dd); err != nil {
		logger.Error(err, "Failed to update status")
		return fmt.Errorf("status update failed: %w", err)
	}

	// 返回聚合错误（如果有）
	if len(aggregatedErrors) > 0 {
		return fmt.Errorf("%d errors: %v", len(aggregatedErrors), aggregatedErrors)
	}
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
