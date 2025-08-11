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

package controller

import (
	"context"
	"github.com/James-Dao/execution-engine/internal/handler"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	dacv1alpha1 "github.com/James-Dao/execution-engine/api/v1alpha1"
)

const (
	requeueAfter = 20 * time.Second
)

// DataDescriptorReconciler reconciles a DataDescriptor object
type DataDescriptorReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Handler *handler.DataDescriptorHandler
}

// +kubebuilder:rbac:groups=dac.dac.io,resources=datadescriptors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dac.dac.io,resources=datadescriptors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dac.dac.io,resources=datadescriptors/finalizers,verbs=update
// +kubebuilder:rbac:groups=dac.dac.io,resources=dataagentcontainers,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DataDescriptor object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *DataDescriptorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	// TODO(user): your logic here

	logger.Info("开始协调 DataDescriptor", "namespace", req.Namespace, "name", req.Name, "type", "DataDescriptor")
	logger.Info("Reconciling DataDescriptor")

	// Fetch the DataDescriptor instance
	instance := &dacv1alpha1.DataDescriptor{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("DataDescriptor deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	err = r.Handler.Do(ctx, instance)

	if err != nil {
		logger.Error(err, "DataDescriptor Handler err")
		return ctrl.Result{RequeueAfter: requeueAfter}, err
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// Example controller function that updates status
func (r *DataDescriptorReconciler) updateStatus(ctx context.Context, dd *dacv1alpha1.DataDescriptor, sources []dacv1alpha1.SourceStatus) error {
	// Set creating condition if this is a new resource
	if dd.Status.OverallPhase == "" {
		dd.Status.SetCreateCondition("Initializing data descriptor")
	}

	// Update the source statuses
	sourceStatuses := make([]dacv1alpha1.SourceStatus, len(sources))
	copy(sourceStatuses, sources)

	// Update consumedBy if needed (example)
	consumedBy := []dacv1alpha1.LocalObjectReference{
		{Name: "example-consumer"},
	}

	// Calculate overall phase based on source statuses
	overallPhase := "Ready"
	for _, status := range sources {
		if status.Phase != "Ready" {
			overallPhase = "Degraded"
			break
		}
	}

	// Update the status
	r.updateDataDescriptorStatus(dd, overallPhase, sourceStatuses, consumedBy)

	// Add additional conditions as needed
	if overallPhase == "Ready" {
		c := dacv1alpha1.NewCondition(dacv1alpha1.ConditionAvailable, corev1.ConditionTrue, "Available", "All data sources are available")
		dd.Status.SetDataDescriptorCondition(*c)
	} else {
		c := dacv1alpha1.NewCondition(dacv1alpha1.ConditionFailed, corev1.ConditionTrue, "Degraded", "Some data sources are not ready")
		dd.Status.SetDataDescriptorCondition(*c)
	}

	// Update the status in Kubernetes
	return r.Status().Update(ctx, dd)
}

// Example controller function that updates status
func (r *DataDescriptorReconciler) updateDataDescriptorStatus(dd *dacv1alpha1.DataDescriptor, phase string, sourceStatuses []dacv1alpha1.SourceStatus, consumedBy []dacv1alpha1.LocalObjectReference) {
	// Update overall phase
	dd.Status.OverallPhase = phase

	// Update source statuses
	dd.Status.SourceStatuses = sourceStatuses

	// Update consumedBy references
	dd.Status.ConsumedBy = consumedBy

	// Sort conditions by time
	dd.Status.DescConditionsByTime()
}

// SetupWithManager sets up the controller with the Manager.
func (r *DataDescriptorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dacv1alpha1.DataDescriptor{}).
		Named("datadescriptor").
		Complete(r)
}
