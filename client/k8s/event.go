package k8s

import (
	"fmt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	dacv1alpha1 "github.com/James-Dao/execution-engine/api/v1alpha1"
)

// Event 定义向 Kubernetes 发送事件的客户端接口
type Event interface {
	// 数据描述符相关事件
	Normal(object runtime.Object, reason, message string)
	Warning(object runtime.Object, reason, message string)

	// 数据源相关事件
	SourceHealthy(object runtime.Object, sourceName, message string)
	SourceUnhealthy(object runtime.Object, sourceName, message string)
	SourceSyncSuccess(object runtime.Object, sourceName string, records int64)
	SourceSyncFailed(object runtime.Object, sourceName, message string)

	// 生命周期事件
	Created(object runtime.Object, message string)
	Updated(object runtime.Object, message string)
	Deleted(object runtime.Object, message string)

	// 特定条件事件
	ConditionChanged(object runtime.Object, conditionType dacv1alpha1.ConditionType, message string)
}

// EventRecorder 是 Event 接口的实现，使用 Kubernetes API 调用记录事件
type EventRecorder struct {
	recorder record.EventRecorder
	logger   logr.Logger
}

// NewEvent 返回一个新的 Event 客户端
func NewEvent(recorder record.EventRecorder, logger logr.Logger) Event {
	return &EventRecorder{
		recorder: recorder,
		logger:   logger.WithName("event-recorder"),
	}
}

// Normal 记录普通类型事件
func (e *EventRecorder) Normal(object runtime.Object, reason, message string) {
	e.recorder.Event(object, corev1.EventTypeNormal, reason, message)
	e.logger.Info(message, "reason", reason, "object", object)
}

// Warning 记录警告类型事件
func (e *EventRecorder) Warning(object runtime.Object, reason, message string) {
	e.recorder.Event(object, corev1.EventTypeWarning, reason, message)
	e.logger.Info(message, "reason", reason, "object", object)
}

// SourceHealthy 记录数据源健康事件
func (e *EventRecorder) SourceHealthy(object runtime.Object, sourceName, message string) {
	reason := "SourceHealthy"
	e.recorder.Eventf(object, corev1.EventTypeNormal, reason, "数据源 %s: %s", sourceName, message)
	e.logger.Info(message, "reason", reason, "source", sourceName, "object", object)
}

// SourceUnhealthy 记录数据源不健康事件
func (e *EventRecorder) SourceUnhealthy(object runtime.Object, sourceName, message string) {
	reason := "SourceUnhealthy"
	e.recorder.Eventf(object, corev1.EventTypeWarning, reason, "数据源 %s: %s", sourceName, message)
	e.logger.Info(message, "reason", reason, "source", sourceName, "object", object)
}

// SourceSyncSuccess 记录数据源同步成功事件
func (e *EventRecorder) SourceSyncSuccess(object runtime.Object, sourceName string, records int64) {
	reason := "SourceSyncSuccess"
	message := fmt.Sprintf("数据源 %s 同步成功，记录数: %d", sourceName, records)
	e.recorder.Eventf(object, corev1.EventTypeNormal, reason, message)
	e.logger.Info(message, "reason", reason, "source", sourceName, "records", records)
}

// SourceSyncFailed 记录数据源同步失败事件
func (e *EventRecorder) SourceSyncFailed(object runtime.Object, sourceName, message string) {
	reason := "SourceSyncFailed"
	e.recorder.Eventf(object, corev1.EventTypeWarning, reason, "数据源 %s 同步失败: %s", sourceName, message)
	e.logger.Info(message, "reason", reason, "source", sourceName)
}

// Created 记录创建事件
func (e *EventRecorder) Created(object runtime.Object, message string) {
	reason := "Created"
	e.recorder.Event(object, corev1.EventTypeNormal, reason, message)
	e.logger.Info(message, "reason", reason)
}

// Updated 记录更新事件
func (e *EventRecorder) Updated(object runtime.Object, message string) {
	reason := "Updated"
	e.recorder.Event(object, corev1.EventTypeNormal, reason, message)
	e.logger.Info(message, "reason", reason)
}

// Deleted 记录删除事件
func (e *EventRecorder) Deleted(object runtime.Object, message string) {
	reason := "Deleted"
	e.recorder.Event(object, corev1.EventTypeNormal, reason, message)
	e.logger.Info(message, "reason", reason)
}

// ConditionChanged 记录条件变更事件
func (e *EventRecorder) ConditionChanged(object runtime.Object, conditionType dacv1alpha1.ConditionType, message string) {
	reason := string(conditionType) + "ConditionChanged"
	e.recorder.Eventf(object, corev1.EventTypeNormal, reason, "条件 %s 变更: %s", conditionType, message)
	e.logger.Info(message, "reason", reason, "condition", conditionType)
}
