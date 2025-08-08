package k8s

import (
	"github.com/go-logr/logr"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	dacv1alpha1 "github.com/James-Dao/execution-engine/api/v1alpha1"
)

// Event the client that push event to kubernetes
type Event interface {
	// NewSlaveAdd event ClusterScaling
	NewDataDescriptionAdd(object runtime.Object, message string)
}

// EventOption is the Event client interface implementation that using API calls to kubernetes.
type EventOption struct {
	eventsCli record.EventRecorder
	logger    logr.Logger
}

// NewEvent returns a new Event client
func NewEvent(eventCli record.EventRecorder, logger logr.Logger) Event {
	return &EventOption{
		eventsCli: eventCli,
		logger:    logger,
	}
}

// NewDataDescriptionAdd implement the Event.Interface
func (e *EventOption) NewDataDescriptionAdd(object runtime.Object, message string) {
	e.eventsCli.Event(object, v1.EventTypeNormal, string(dacv1alpha1.DataDescriptionConditionAdd), message)
}
