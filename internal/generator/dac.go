package generator

import (
	"context"
	dacv1alpha1 "github.com/James-Dao/execution-engine/api/v1alpha1"
	"github.com/James-Dao/execution-engine/client/k8s"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DataAgentContainerHandler handles the reconciliation logic for DataAgentContainer resources.
type DataAgentContainerGenerator struct {
	K8sServices k8s.Services
	EventsCli   k8s.Event
	Kubeclient  client.Client
	Logger      logr.Logger
}

func (h *DataAgentContainerGenerator) Do(ctx context.Context, dac *dacv1alpha1.DataAgentContainer) error {
	logger := h.Logger.WithValues("namespace", dac.Namespace, "name", dac.Name)
	logger.Info("Generate DataAgentContainer K8S resources")

	return nil
}

func (h *DataAgentContainerGenerator) GenerateDataAgentContainerService(dac *dacv1alpha1.DataAgentContainer, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Service {
	serviceName := h.generateDataAgentContainerServiceName(dac)
	dacTargetPort := intstr.FromInt(6379)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            serviceName,
			Namespace:       dac.Namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: corev1.ClusterIPNone,
			Ports: []corev1.ServicePort{
				{
					Port:       6379,
					Protocol:   corev1.ProtocolTCP,
					Name:       "dac",
					TargetPort: dacTargetPort,
				},
			},
			Selector: labels,
		},
	}
}

func (h *DataAgentContainerGenerator) generateDataAgentContainerServiceName(dac *dacv1alpha1.DataAgentContainer) string {
	return ""
}
