package model

import (
	"fmt"
	"strings"

	"github.com/berestyak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	v13 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	LivenessProbeInitialDelay  = 30
	ReadinessProbeInitialDelay = 40
	//10s (curl) + 10s (curl) + 2s (just in case)
	ProbeTimeoutSeconds         = 22
	ProbeTimeBetweenRunsSeconds = 30
	ProbeFailureThreshold       = 10
)

func GetServiceEnvVar(suffix string) string {
	serviceName := strings.ToUpper(PostgresqlServiceName)
	serviceName = strings.ReplaceAll(serviceName, "-", "_")
	return fmt.Sprintf("%v_%v", serviceName, suffix)
}

func findInitContainerInSlice(cr *v1alpha1.Keycloak, name string) int {
	for i, container := range cr.Spec.KeycloakDeploymentSpec.InitContainers {
		if container.Name == name {
			return i
		}
	}
	log.Log.Error(nil, "Init container with name "+name+" wasn't found in spec!")
	return 0
}

func findContainerInSlice(cr *v1alpha1.Keycloak, name string) int {
	for i, container := range cr.Spec.KeycloakDeploymentSpec.Containers {
		if container.Name == name {
			return i
		}
	}
	log.Log.Error(nil, "Container with name "+name+" wasn't found in spec!")
	return 0
}

func KeycloakDeployment(cr *v1alpha1.Keycloak, dbSecret *v1.Secret) *v13.StatefulSet {
	return &v13.StatefulSet{
		ObjectMeta: v12.ObjectMeta{
			Name:      KeycloakDeploymentName,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"app":       ApplicationName,
				"component": KeycloakDeploymentComponent,
			},
		},
		Spec: v13.StatefulSetSpec{
			Replicas: SanitizeNumberOfReplicas(cr.Spec.Instances, true),
			Selector: &v12.LabelSelector{
				MatchLabels: map[string]string{
					"app":       ApplicationName,
					"component": KeycloakDeploymentComponent,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: v12.ObjectMeta{
					Name:      KeycloakDeploymentName,
					Namespace: cr.Namespace,
					Labels: map[string]string{
						"app":       ApplicationName,
						"component": KeycloakDeploymentComponent,
					},
				},
				// Append standart values to keycloak container in CR spec
				Spec: func(cr *v1alpha1.Keycloak) v1.PodSpec {
					specPod := cr.Spec.KeycloakDeploymentSpec.PodSpec
					keycloakContainerPosition := findContainerInSlice(cr, KeycloakDeploymentName)
					specPod.Containers[keycloakContainerPosition].Image = getKeycloakImageFromCR(cr, keycloakContainerPosition)
					specPod.Containers[keycloakContainerPosition].Ports = KeycloakPorts(cr, keycloakContainerPosition)
					specPod.Containers[keycloakContainerPosition].Env = getKeycloakEnv(cr, dbSecret, keycloakContainerPosition)
					specPod.Containers[keycloakContainerPosition].VolumeMounts = KeycloakVolumeMounts(cr, keycloakContainerPosition)
					specPod.Containers[keycloakContainerPosition].LivenessProbe = livenessProbe(cr, keycloakContainerPosition)
					specPod.Containers[keycloakContainerPosition].ReadinessProbe = readinessProbe(cr, keycloakContainerPosition)
					specPod.Volumes = KeycloakVolumes(cr)
					specPod.InitContainers = KeycloakExtensionsInitContainers(cr)
					return specPod
				}(cr),
			},
		},
	}
}

func KeycloakDeploymentReconciled(cr *v1alpha1.Keycloak, currentState *v13.StatefulSet, dbSecret *v1.Secret) *v13.StatefulSet {
	reconciled := currentState.DeepCopy()
	keycloakContainerPosition := findContainerInSlice(cr, KeycloakDeploymentName)
	reconciled.ResourceVersion = currentState.ResourceVersion
	reconciled.Spec.Replicas = SanitizeNumberOfReplicas(cr.Spec.Instances, false)
	reconciled.Spec.Template.Spec.Containers[keycloakContainerPosition].Image = getKeycloakImageFromCR(cr, keycloakContainerPosition)
	reconciled.Spec.Template.Spec.Containers[keycloakContainerPosition].Ports = KeycloakPorts(cr, keycloakContainerPosition)
	reconciled.Spec.Template.Spec.Containers[keycloakContainerPosition].Env = getKeycloakEnv(cr, dbSecret, keycloakContainerPosition)
	reconciled.Spec.Template.Spec.Containers[keycloakContainerPosition].VolumeMounts = KeycloakVolumeMounts(cr, keycloakContainerPosition)
	reconciled.Spec.Template.Spec.Containers[keycloakContainerPosition].LivenessProbe = livenessProbe(cr, keycloakContainerPosition)
	reconciled.Spec.Template.Spec.Containers[keycloakContainerPosition].ReadinessProbe = readinessProbe(cr, keycloakContainerPosition)
	reconciled.Spec.Template.Spec.InitContainers = KeycloakExtensionsInitContainers(cr)
	reconciled.Spec.Template.Spec.Volumes = KeycloakVolumes(cr)
	return reconciled
}

func getResources(cr *v1alpha1.Keycloak, containerNum int) v1.ResourceRequirements {
	requirements := v1.ResourceRequirements{}
	requirements.Limits = v1.ResourceList{}
	requirements.Requests = v1.ResourceList{}

	cpu, err := resource.ParseQuantity(cr.Spec.KeycloakDeploymentSpec.Containers[containerNum].Resources.Requests.Cpu().String())
	if err == nil && cpu.String() != "0" {
		requirements.Requests[v1.ResourceCPU] = cpu
	}

	memory, err := resource.ParseQuantity(cr.Spec.KeycloakDeploymentSpec.Containers[containerNum].Resources.Requests.Memory().String())
	if err == nil && memory.String() != "0" {
		requirements.Requests[v1.ResourceMemory] = memory
	}

	cpu, err = resource.ParseQuantity(cr.Spec.KeycloakDeploymentSpec.Containers[containerNum].Resources.Limits.Cpu().String())
	if err == nil && cpu.String() != "0" {
		requirements.Limits[v1.ResourceCPU] = cpu
	}
	memory, err = resource.ParseQuantity(cr.Spec.KeycloakDeploymentSpec.Containers[containerNum].Resources.Limits.Memory().String())
	if err == nil && memory.String() != "0" {
		requirements.Limits[v1.ResourceMemory] = memory
	}

	return requirements
}

func getKeycloakEnv(cr *v1alpha1.Keycloak, dbSecret *v1.Secret, containerNum int) []v1.EnvVar {
	env := []v1.EnvVar{
		// Database settings
		{
			Name:  "DB_VENDOR",
			Value: "POSTGRES",
		},
		{
			Name:  "DB_SCHEMA",
			Value: "public",
		},
		{
			Name:  "DB_ADDR",
			Value: PostgresqlServiceName + "." + cr.Namespace,
		},
		{
			Name:  "DB_PORT",
			Value: fmt.Sprintf("%v", GetExternalDatabasePort(dbSecret)),
		},
		{
			Name:  "DB_DATABASE",
			Value: GetExternalDatabaseName(dbSecret),
		},
		{
			Name: "DB_USER",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: DatabaseSecretName,
					},
					Key: DatabaseSecretUsernameProperty,
				},
			},
		},
		{
			Name: "DB_PASSWORD",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: DatabaseSecretName,
					},
					Key: DatabaseSecretPasswordProperty,
				},
			},
		},
		// Discovery settings
		{
			Name:  "NAMESPACE",
			Value: cr.Namespace,
		},
		{
			Name:  "JGROUPS_DISCOVERY_PROTOCOL",
			Value: "dns.DNS_PING",
		},
		{
			Name:  "JGROUPS_DISCOVERY_PROPERTIES",
			Value: "dns_query=" + KeycloakDiscoveryServiceName + "." + cr.Namespace,
		},
		// Cache settings
		{
			Name:  "CACHE_OWNERS_COUNT",
			Value: "2",
		},
		{
			Name:  "CACHE_OWNERS_AUTH_SESSIONS_COUNT",
			Value: "2",
		},
		{
			Name: "KEYCLOAK_USER",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: "credential-" + cr.Name,
					},
					Key: AdminUsernameProperty,
				},
			},
		},
		{
			Name: "KEYCLOAK_PASSWORD",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: "credential-" + cr.Name,
					},
					Key: AdminPasswordProperty,
				},
			},
		},
		{
			Name:  "X509_CA_BUNDLE",
			Value: "/var/run/secrets/kubernetes.io/serviceaccount/*.crt",
		},
		{
			Name:  "PROXY_ADDRESS_FORWARDING",
			Value: "true",
		},
	}

	if len(cr.Spec.KeycloakDeploymentSpec.Containers[containerNum].Env) > 0 {
		// We override Keycloak pre-defined envs with what user specified. Not the other way around.
		env = MergeEnvs(cr.Spec.KeycloakDeploymentSpec.Containers[containerNum].Env, env)
	}

	return env
}

// KeycloakPorts - Append ports from CR to default ports
func KeycloakPorts(cr *v1alpha1.Keycloak, containerNum int) []v1.ContainerPort {
	defaultPorts := []v1.ContainerPort{
		{
			ContainerPort: KeycloakHTTPSServicePort,
			Protocol:      "TCP",
		},
		{
			ContainerPort: 9990,
			Protocol:      "TCP",
		},
		{
			ContainerPort: KeycloakHTTPServicePort,
			Protocol:      "TCP",
		},
	}
	return MergePorts(cr.Spec.KeycloakDeploymentSpec.Containers[containerNum].Ports, defaultPorts)
}

func KeycloakDeploymentSelector(cr *v1alpha1.Keycloak) client.ObjectKey {
	return client.ObjectKey{
		Name:      KeycloakDeploymentName,
		Namespace: cr.Namespace,
	}
}

func KeycloakVolumeMounts(cr *v1alpha1.Keycloak, containerNum int) []v1.VolumeMount {
	defaultVolumeMounts := []v1.VolumeMount{
		{
			Name:      ServingCertSecretName,
			MountPath: "/etc/x509/https",
		},
		{
			Name:      "keycloak-extensions",
			ReadOnly:  false,
			MountPath: KeycloakExtensionPath,
		},
		{
			Name:      KeycloakProbesName,
			MountPath: "/probes",
		},
	}
	return MergeVolumeMounts(cr.Spec.KeycloakDeploymentSpec.Containers[containerNum].VolumeMounts, defaultVolumeMounts)
}

func getKeycloakImageFromCR(cr *v1alpha1.Keycloak, containerNum int) string {
	if cr.Spec.KeycloakDeploymentSpec.Containers[containerNum].Image != "" {
		return cr.Spec.KeycloakDeploymentSpec.Containers[containerNum].Image
	} else {
		return DefaultKeycloakImage
	}
}

func KeycloakVolumes(cr *v1alpha1.Keycloak) []v1.Volume {
	defaultVolumes := []v1.Volume{
		{
			Name: ServingCertSecretName,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: ServingCertSecretName,
					Optional:   &[]bool{true}[0],
				},
			},
		},
		{
			Name: "keycloak-extensions",
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: KeycloakProbesName,
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: KeycloakProbesName,
					},
					DefaultMode: &[]int32{0555}[0],
				},
			},
		},
	}

	return MergeVolumes(cr.Spec.KeycloakDeploymentSpec.Volumes, defaultVolumes)
}

func livenessProbe(cr *v1alpha1.Keycloak, containerNum int) *v1.Probe {
	if cr.Spec.KeycloakDeploymentSpec.Containers[containerNum].LivenessProbe != nil {
		return cr.Spec.KeycloakDeploymentSpec.Containers[containerNum].LivenessProbe
	}
	return &v1.Probe{
		Handler: v1.Handler{
			Exec: &v1.ExecAction{
				Command: []string{
					"/bin/sh",
					"-c",
					"/probes/" + LivenessProbeProperty,
				},
			},
		},
		InitialDelaySeconds: LivenessProbeInitialDelay,
		TimeoutSeconds:      ProbeTimeoutSeconds,
		PeriodSeconds:       ProbeTimeBetweenRunsSeconds,
		FailureThreshold:    ProbeFailureThreshold,
	}
}

func readinessProbe(cr *v1alpha1.Keycloak, containerNum int) *v1.Probe {
	if cr.Spec.KeycloakDeploymentSpec.Containers[containerNum].ReadinessProbe != nil {
		return cr.Spec.KeycloakDeploymentSpec.Containers[containerNum].ReadinessProbe
	}
	return &v1.Probe{
		Handler: v1.Handler{
			Exec: &v1.ExecAction{
				Command: []string{
					"/bin/sh",
					"-c",
					"/probes/" + ReadinessProbeProperty,
				},
			},
		},
		InitialDelaySeconds: ReadinessProbeInitialDelay,
		TimeoutSeconds:      ProbeTimeoutSeconds,
		PeriodSeconds:       ProbeTimeBetweenRunsSeconds,
		FailureThreshold:    ProbeFailureThreshold,
	}
}
