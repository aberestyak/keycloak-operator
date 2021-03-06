package model

import (
	"strings"

	"github.com/aberestyak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

func KeycloakExtensionsInitContainers(cr *v1alpha1.Keycloak) []v1.Container {
	initContainers := []v1.Container{}
	for _, initContainer := range cr.Spec.KeycloakDeploymentSpec.InitContainers {
		if initContainer.Name == "extensions-init" {
			initContainer.Image = getInitContainerImageFromCR(cr)
			initContainer.Env = getInitContainerEnv(cr)
			initContainer.VolumeMounts = []v1.VolumeMount{
				{
					Name:      "keycloak-extensions",
					ReadOnly:  false,
					MountPath: KeycloakExtensionsInitContainerPath,
				},
			}
		}
		initContainers = append(initContainers, initContainer)
	}
	return initContainers
}

func getInitContainerImageFromCR(cr *v1alpha1.Keycloak) string {
	if cr.Spec.KeycloakDeploymentSpec.InitContainers[findInitContainerInSlice(cr, "extensions-init")].Image != "" {
		return cr.Spec.KeycloakDeploymentSpec.InitContainers[findInitContainerInSlice(cr, "extensions-init")].Image
	} else {
		return DefaultKeycloakInitContainer
	}
}

func getInitContainerEnv(cr *v1alpha1.Keycloak) []v1.EnvVar {
	env := []v1.EnvVar{
		{
			Name:  KeycloakExtensionEnvVar,
			Value: strings.Join(cr.Spec.Extensions, ","),
		},
	}
	if len(cr.Spec.KeycloakDeploymentSpec.InitContainers[findInitContainerInSlice(cr, "extensions-init")].Env) > 0 {
		// We override Keycloak pre-defined envs with what user specified. Not the other way around.
		env = MergeEnvs(cr.Spec.KeycloakDeploymentSpec.InitContainers[findInitContainerInSlice(cr, "extensions-init")].Env, env)
	}
	return env
}
