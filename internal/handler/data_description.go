package handler

import (
	"github.com/James-Dao/execution-engine/client/k8s"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dacv1alpha1 "github.com/James-Dao/execution-engine/api/v1alpha1"
)

type DataDescriptorHandler struct {
	K8sServices k8s.Services
	EventsCli   k8s.Event
	Kubeclient  client.Client
	Logger      logr.Logger
}

// Do will ensure the RedisCluster is in the expected state and update the RedisCluster status.
func (r *DataDescriptorHandler) Do(dd *dacv1alpha1.DataDescriptor) error {
	r.Logger.WithValues("namespace", dd.Namespace, "name", dd.Name, "type", "DataDescriptor").Info("handler doing")

	clusterHealthy := true

	if clusterHealthy {
		r.Logger.WithValues("namespace", dd.Namespace, "name", dd.Name, "type", "DataDescriptor").Info("SetReadyCondition...")
		// r.EventsCli.HealthCluster(dd)
		// dd.Status.SetReadyCondition("Cluster ok")
	}

	return nil
}
