package handler

import (
	"github.com/James-Dao/execution-engine/client/k8s"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dacv1alpha1 "github.com/James-Dao/execution-engine/api/v1alpha1"
)

type DataAgentContainerHandler struct {
	K8sServices k8s.Services
	EventsCli   k8s.Event
	Kubeclient  client.Client
	Logger      logr.Logger
}

// Do will ensure the RedisCluster is in the expected state and update the RedisCluster status.
func (r *DataAgentContainerHandler) Do(dac *dacv1alpha1.DataAgentContainer) error {
	r.Logger.WithValues("namespace", dac.Namespace, "name", dac.Name, "type", "DataAgentContainer").Info("handler doing")

	clusterHealthy := true

	if clusterHealthy {
		r.Logger.WithValues("namespace", dac.Namespace, "name", dac.Name, "type", "DataAgentContainer").Info("SetReadyCondition...")
		// r.EventsCli.HealthCluster(dac)
		// dac.Status.SetReadyCondition("Cluster ok")
	}

	return nil
}
