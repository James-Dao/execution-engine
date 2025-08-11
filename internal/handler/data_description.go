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

// SourceStatusResult 包含检查数据源状态的结果
type SourceStatusResult struct {
	Phase        string
	LastSyncTime metav1.Time
	Records      int64
	Error        error
}

// Do 确保DataDescriptor处于预期状态并更新状态
func (h *DataDescriptorHandler) Do(ctx context.Context, dd *dacv1alpha1.DataDescriptor) error {
	logger := h.Logger.WithValues("namespace", dd.Namespace, "name", dd.Name)
	logger.Info("开始处理DataDescriptor")

	// 检查所有数据源状态
	sourceStatuses := make([]dacv1alpha1.SourceStatus, len(dd.Spec.Sources))
	allHealthy := true

	for i, source := range dd.Spec.Sources {
		status := h.checkSourceStatus(ctx, source)
		sourceStatuses[i] = dacv1alpha1.SourceStatus{
			Name:         source.Name,
			Phase:        status.Phase,
			LastSyncTime: status.LastSyncTime,
			Records:      status.Records,
		}

		if status.Error != nil {
			logger.Error(status.Error, "数据源状态检查失败", "source", source.Name)
			allHealthy = false
			h.EventsCli.Warning(dd, "SourceCheckFailed",
				fmt.Sprintf("数据源 %s 检查失败: %v", source.Name, status.Error))
		} else if status.Phase != "Ready" {
			allHealthy = false
		}
	}

	// 更新状态
	if allHealthy {
		h.EventsCli.Normal(dd, "AllSourcesHealthy", "所有数据源状态正常")
	} else {
		h.EventsCli.Warning(dd, "SomeSourcesUnhealthy", "部分数据源状态异常")
	}

	return nil
}

// checkSourceStatus 检查单个数据源的状态
func (h *DataDescriptorHandler) checkSourceStatus(ctx context.Context, source dacv1alpha1.DataSource) SourceStatusResult {
	switch source.Type {
	case dacv1alpha1.DataSourceRedis:
		return h.checkRedisStatus(ctx, source)
	case dacv1alpha1.DataSourceMySQL:
		return h.checkMySQLStatus(ctx, source)
	case dacv1alpha1.DataSourceMinIO:
		return h.checkMinIOStatus(ctx, source)
	default:
		return SourceStatusResult{
			Phase: "Unknown",
			Error: fmt.Errorf("未知的数据源类型: %s", source.Type),
		}
	}
}

// checkRedisStatus 检查Redis数据源状态
func (h *DataDescriptorHandler) checkRedisStatus(ctx context.Context, source dacv1alpha1.DataSource) SourceStatusResult {

	// 模拟返回
	return SourceStatusResult{
		Phase:        "Ready",
		LastSyncTime: metav1.NewTime(time.Now()),
		Records:      1000, // 模拟值
	}
}

// checkMySQLStatus 检查MySQL数据源状态
func (h *DataDescriptorHandler) checkMySQLStatus(ctx context.Context, source dacv1alpha1.DataSource) SourceStatusResult {

	// 模拟返回
	return SourceStatusResult{
		Phase:        "Ready",
		LastSyncTime: metav1.NewTime(time.Now()),
		Records:      5000, // 模拟值
	}
}

// checkMinIOStatus 检查MinIO数据源状态
func (h *DataDescriptorHandler) checkMinIOStatus(ctx context.Context, source dacv1alpha1.DataSource) SourceStatusResult {

	// 模拟返回
	return SourceStatusResult{
		Phase:        "Ready",
		LastSyncTime: metav1.NewTime(time.Now()),
		Records:      200, // 模拟值
	}
}

// GetSourceStatus 获取指定数据源状态(供控制器调用)
func (h *DataDescriptorHandler) GetSourceStatus(ctx context.Context, source dacv1alpha1.DataSource) dacv1alpha1.SourceStatus {
	result := h.checkSourceStatus(ctx, source)
	return dacv1alpha1.SourceStatus{
		Name:         source.Name,
		Phase:        result.Phase,
		LastSyncTime: result.LastSyncTime,
		Records:      result.Records,
	}
}
