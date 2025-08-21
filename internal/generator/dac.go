package generator

import (
	"context"
	"fmt"
	dacv1alpha1 "github.com/James-Dao/execution-engine/api/v1alpha1"
	"github.com/James-Dao/execution-engine/client/k8s"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	// "os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DataAgentContainerHandler handles the reconciliation logic for DataAgentContainer resources.
type DataAgentContainerGenerator struct {
	K8sServices k8s.Services
	Kubeclient  client.Client
	Logger      logr.Logger
}

// LLMConfig
type LLMConfig struct {
	Provider string
	APIKey   string
	BaseURL  string
	Model    string
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
	serviceName := h.GenerateDataAgentContainerServiceName(dac)
	if _, err := h.K8sServices.GetService(dac.Namespace, serviceName); err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			err := h.K8sServices.CreateService(dac.Namespace, service)
			if err != nil {
				return err
			}
		}
	}

	deployment, err := h.GenerateDataAgentContainerDeployment(ctx, dac, labels, ownerRefs)
	if err != nil {
		return err
	}

	deploymentName := h.GenerateDataAgentContainerDeploymentName(dac)
	if _, err := h.K8sServices.GetDeployment(dac.Namespace, deploymentName); err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			err = h.K8sServices.CreateDeployment(dac.Namespace, deployment)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *DataAgentContainerGenerator) GenerateDataAgentContainerService(dac *dacv1alpha1.DataAgentContainer, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Service {
	serviceName := h.GenerateDataAgentContainerServiceName(dac)
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
			// Type:      corev1.ServiceTypeClusterIP,
			// ClusterIP: corev1.ClusterIPNone,
			Type: corev1.ServiceTypeNodePort,
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

func (h *DataAgentContainerGenerator) GenerateDataAgentContainerServiceName(dac *dacv1alpha1.DataAgentContainer) string {
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

// getLLMConfig get data from configmap
func (h *DataAgentContainerGenerator) getLLMConfig(ctx context.Context, dac *dacv1alpha1.DataAgentContainer) (*LLMConfig, error) {
	configMap := &corev1.ConfigMap{}

	err := h.Kubeclient.Get(ctx, client.ObjectKey{Name: dac.Spec.Model.LLM, Namespace: dac.Namespace}, configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to get ConfigMap: %v", err)
	}

	return &LLMConfig{
		Provider: configMap.Data["provider"],
		APIKey:   configMap.Data["api-key"],
		BaseURL:  configMap.Data["base-url"],
		Model:    configMap.Data["model"],
	}, nil
}

func (h *DataAgentContainerGenerator) generateOrchestratorAgentArgs(dac *dacv1alpha1.DataAgentContainer, llmConfig *LLMConfig) []string {
	port := "10100"
	redisHost := "redis-server.dac.svc.cluster.local"
	redisPort := "6379"
	redisDB := "0"
	password := "123"

	cmds := []string{
		"--port",
		port,
		"--redis-host",
		redisHost,
		"--redis-port",
		redisPort,
		"--redis-db",
		redisDB,
		"--password",
		password,
		"--provider",
		llmConfig.Provider,
		"--api-key",
		llmConfig.APIKey,
		"--base-url",
		llmConfig.BaseURL,
		"--model",
		llmConfig.Model,
	}
	return cmds
}

func (h *DataAgentContainerGenerator) generateExpertAgentArgs(dac *dacv1alpha1.DataAgentContainer, llmConfig *LLMConfig) []string {
	port := "10101"
	redisHost := "redis-server.dac.svc.cluster.local"
	redisPort := "6379"
	redisDB := "1"
	password := "123"

	cmds := []string{
		"--port",
		port,
		"--redis-host",
		redisHost,
		"--redis-port",
		redisPort,
		"--redis-db",
		redisDB,
		"--password",
		password,
		"--provider",
		llmConfig.Provider,
		"--api-key",
		llmConfig.APIKey,
		"--base-url",
		llmConfig.BaseURL,
		"--model",
		llmConfig.Model,
	}
	return cmds
}

func (h *DataAgentContainerGenerator) GenerateDataAgentContainerDeploymentName(dac *dacv1alpha1.DataAgentContainer) string {
	deploymentName := fmt.Sprintf("%s-%s", dac.Name, "deployment")
	return deploymentName
}

func (h *DataAgentContainerGenerator) GenerateDataAgentContainerDeployment(ctx context.Context, dac *dacv1alpha1.DataAgentContainer, labels map[string]string, ownerRefs []metav1.OwnerReference) (*appsv1.Deployment, error) {

	orchestratorAgentImage := "registry.cn-shanghai.aliyuncs.com/jamesxiong/orchestrator-agent:v0.0.1-amd64"
	expertAgentImage := "registry.cn-shanghai.aliyuncs.com/jamesxiong/expert-agent:v0.0.1-amd64"

	name := h.GenerateDataAgentContainerDeploymentName(dac)

	serviceName := h.GenerateDataAgentContainerServiceName(dac)

	replicas := int32(1)

	llmConfig, err := h.getLLMConfig(ctx, dac)

	if err != nil {
		return nil, err
	}

	orchestratorAgentArgs := h.generateOrchestratorAgentArgs(dac, llmConfig)

	expertAgentArgs := h.generateExpertAgentArgs(dac, llmConfig)

	// todo handle private image PullSecrets
	// var imagePullSecrets []corev1.LocalObjectReference

	// secretName := os.Getenv("IMAGE_PULL_SECRET")
	// if secretName != "" {
	// 	imagePullSecrets = []corev1.LocalObjectReference{
	// 		{Name: secretName},
	// 	}
	// }

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
					// ImagePullSecrets: imagePullSecrets,
					Containers: []corev1.Container{
						{
							Name:            "orchestrator",
							Image:           orchestratorAgentImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							// ImagePullPolicy: corev1.PullAlways,
							Args: orchestratorAgentArgs,
							Ports: []corev1.ContainerPort{
								{
									Name:          "orchestrator",
									ContainerPort: 10100,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: h.generateOrchestratorAgentEnvs(dac, serviceName),
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("2000Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("1000Mi"),
								},
							},
						},
						{
							Name:            "expert",
							Image:           expertAgentImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							// ImagePullPolicy: corev1.PullAlways,
							Args: expertAgentArgs,
							Ports: []corev1.ContainerPort{
								{
									Name:          "expert",
									ContainerPort: 10101,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: h.generateExpertAgentEnvs(dac, serviceName),
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("2000Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("1000Mi"),
								},
							},
						},
					},
				},
			},
		},
	}
	return deployment, nil
}
