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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	dacv1alpha1 "github.com/James-Dao/execution-engine/api/v1alpha1"
)

var (
	dacLog = logf.Log.WithName("controller_DataDescriptor")
)

// DataAgentContainerReconciler reconciles a DataAgentContainer object
type DataAgentContainerReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Handler *handler.DataAgentContainerHandler
}

// +kubebuilder:rbac:groups=dac.dac.io,resources=dataagentcontainers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dac.dac.io,resources=dataagentcontainers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dac.dac.io,resources=dataagentcontainers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DataAgentContainer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *DataAgentContainerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	// TODO(user): your logic here

	reqLogger := ddLog.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name, "type", "DataAgentContainer")
	reqLogger.Info("Reconciling DataAgentContainer")

	// Fetch the DataAgentContainer instance
	instance := &dacv1alpha1.DataAgentContainer{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("DataAgentContainer delete")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DataAgentContainerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dacv1alpha1.DataAgentContainer{}).
		Named("dataagentcontainer").
		Complete(r)
}
