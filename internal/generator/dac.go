package generator

import (
	"context"
	"fmt"
	dacv1alpha1 "github.com/James-Dao/execution-engine/api/v1alpha1"
	"github.com/James-Dao/execution-engine/client/k8s"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DataAgentContainerHandler handles the reconciliation logic for DataAgentContainer resources.
type DataAgentContainerGenerator struct {
	K8sServices k8s.Services
	Kubeclient  client.Client
	Logger      logr.Logger
}

func (h *DataAgentContainerGenerator) Do(ctx context.Context, dac *dacv1alpha1.DataAgentContainer) error {
	logger := h.Logger.WithValues("namespace", dac.Namespace, "name", dac.Name)
	logger.Info("Generate DataAgentContainer K8S resources")

	labels := map[string]string{
		"app": dac.Name,
	}

	isController := true
	ownerRefs := []metav1.OwnerReference{
		{
			APIVersion: dac.APIVersion,
			Kind:       dac.Kind,
			Name:       dac.Name,
			UID:        dac.UID,
			Controller: &isController,
		},
	}

	service := h.GenerateDataAgentContainerService(dac, labels, ownerRefs)
	err := h.K8sServices.CreateService(dac.Namespace, service)
	if err != nil {
		return err
	}

	deployment := h.GenerateDataAgentContainerDeployment(dac, labels, ownerRefs)
	err = h.K8sServices.CreateDeployment(dac.Namespace, deployment)
	if err != nil {
		return err
	}

	return nil
}

func (h *DataAgentContainerGenerator) GenerateDataAgentContainerService(dac *dacv1alpha1.DataAgentContainer, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Service {
	serviceName := h.generateDataAgentContainerServiceName(dac)
	orchestratorTargetPort := intstr.FromInt(10100)
	dacTargetPort := intstr.FromInt(10101)
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
					Port:       10100,
					Protocol:   corev1.ProtocolTCP,
					Name:       "orchestrator",
					TargetPort: orchestratorTargetPort,
				},
				{
					Port:       10101,
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
	serviceName := fmt.Sprintf("%s-%s", dac.Name, "service")
	return serviceName
}

func (h *DataAgentContainerGenerator) generateExpertAgentEnvs(dac *dacv1alpha1.DataAgentContainer, serviceName string) []corev1.EnvVar {
	envs := []corev1.EnvVar{}

	envs = append(envs, corev1.EnvVar{
		Name:  "Agent_Host",
		Value: serviceName,
	})

	envs = append(envs, corev1.EnvVar{
		Name:  "Agent_Port",
		Value: "10101",
	})

	return envs
}

func (h *DataAgentContainerGenerator) generateOrchestratorAgentEnvs(dac *dacv1alpha1.DataAgentContainer, serviceName string) []corev1.EnvVar {
	envs := []corev1.EnvVar{}

	envs = append(envs, corev1.EnvVar{
		Name:  "Agent_Host",
		Value: serviceName,
	})

	envs = append(envs, corev1.EnvVar{
		Name:  "Agent_Port",
		Value: "10100",
	})

	envs = append(envs, corev1.EnvVar{
		Name:  "AgentRegistry",
		Value: "expert-registry:10100",
	})

	return envs
}

func (h *DataAgentContainerGenerator) generateOrchestratorAgentArgs(dac *dacv1alpha1.DataAgentContainer) []string {
	redisHost := "redis-server"
	redisPort := "6379"
	redisDB := "0"
	password := "123"

	cmds := []string{
		"--redis-host",
		redisHost,
		"--redis-port",
		redisPort,
		"--redis-db",
		redisDB,
		"--password",
		password,
	}
	return cmds
}

func (h *DataAgentContainerGenerator) generateExpertAgentArgs(dac *dacv1alpha1.DataAgentContainer) []string {
	redisHost := "redis-server"
	redisPort := "6379"
	redisDB := "1"
	password := "123"

	cmds := []string{
		"--redis-host",
		redisHost,
		"--redis-port",
		redisPort,
		"--redis-db",
		redisDB,
		"--password",
		password,
	}
	return cmds
}

func (h *DataAgentContainerGenerator) generateDataAgentContainerDeploymentName(dac *dacv1alpha1.DataAgentContainer) string {
	deploymentName := fmt.Sprintf("%s-%s", dac.Name, "deployment")
	return deploymentName
}

func (h *DataAgentContainerGenerator) GenerateDataAgentContainerDeployment(dac *dacv1alpha1.DataAgentContainer, labels map[string]string, ownerRefs []metav1.OwnerReference) *appsv1.Deployment {

	orchestratorAgentImage := "registry.cn-shanghai.aliyuncs.com/jamesxiong/orchestrator-agent:v0.0.1-amd64"
	expertAgentImage := "registry.cn-shanghai.aliyuncs.com/jamesxiong/expert-agent:v0.0.1-amd64"

	name := h.generateDataAgentContainerDeploymentName(dac)

	serviceName := h.generateDataAgentContainerServiceName(dac)

	replicas := int32(1)

	orchestratorAgentArgs := h.generateOrchestratorAgentArgs(dac)

	expertAgentArgs := h.generateExpertAgentArgs(dac)

	var imagePullSecrets []corev1.LocalObjectReference

	secretName := os.Getenv("IMAGE_PULL_SECRET")
	if secretName != "" {
		imagePullSecrets = []corev1.LocalObjectReference{
			{Name: secretName},
		}
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       dac.Namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Strategy: appsv1.DeploymentStrategy{
				Type: "RollingUpdate",
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: imagePullSecrets,
					Containers: []corev1.Container{
						{
							Name:            "orchestrator-agent",
							Image:           orchestratorAgentImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args:            orchestratorAgentArgs,
							Ports: []corev1.ContainerPort{
								{
									Name:          "orchestrator-agent",
									ContainerPort: 10100,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: h.generateOrchestratorAgentEnvs(dac, serviceName),
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("100Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("100Mi"),
								},
							},
						},
						{
							Name:            "expert-agent",
							Image:           expertAgentImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args:            expertAgentArgs,
							Ports: []corev1.ContainerPort{
								{
									Name:          "expert-agent",
									ContainerPort: 10101,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: h.generateExpertAgentEnvs(dac, serviceName),
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("100Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("100Mi"),
								},
							},
						},
					},
				},
			},
		},
	}
	return deployment
}
