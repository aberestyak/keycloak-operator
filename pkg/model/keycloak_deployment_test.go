package model

import (
	"fmt"
	"testing"

	"github.com/aberestyak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	"github.com/stretchr/testify/assert"
	v13 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

type createDeploymentStatefulSet func(*v1alpha1.Keycloak, *v1.Secret) *v13.StatefulSet

func testPostgresEnvs(t *testing.T, deploymentFunction createDeploymentStatefulSet) {
	//given
	cr := &v1alpha1.Keycloak{}

	//when
	envs := deploymentFunction(cr, nil).Spec.Template.Spec.Containers[0].Env

	//then
	assert.Equal(t, getEnvValueByName(envs, "DB_VENDOR"), "POSTGRES")
	assert.Equal(t, getEnvValueByName(envs, "DB_SCHEMA"), "public")
	assert.Equal(t, getEnvValueByName(envs, "DB_ADDR"), PostgresqlServiceName+"."+cr.Namespace)
	assert.True(t, getEnvValueByName(envs, "DB_PORT") != "")
	assert.Equal(t, getEnvValueByName(envs, "DB_PORT"), fmt.Sprintf("%v", PostgresDefaultPort))
	assert.Equal(t, getEnvValueByName(envs, "DB_DATABASE"), PostgresqlDatabase)

	//given
	cr = &v1alpha1.Keycloak{
		Spec: v1alpha1.KeycloakSpec{
			ExternalDatabase: v1alpha1.KeycloakExternalDatabase{
				Enabled: true,
			},
		},
	}

	//when
	dbSecret := &v1.Secret{
		Data: map[string][]byte{
			DatabaseSecretDatabaseProperty:        []byte("test"),
			DatabaseSecretExternalAddressProperty: []byte("postgres.example.com"),
			DatabaseSecretExternalPortProperty:    []byte("12345"),
		},
	}
	envs = deploymentFunction(cr, dbSecret).Spec.Template.Spec.Containers[0].Env

	//then
	assert.Equal(t, "POSTGRES", getEnvValueByName(envs, "DB_VENDOR"))
	assert.Equal(t, "public", getEnvValueByName(envs, "DB_SCHEMA"))
	assert.Equal(t, PostgresqlServiceName+"."+cr.Namespace, getEnvValueByName(envs, "DB_ADDR"))
	assert.True(t, getEnvValueByName(envs, "DB_PORT") != "")
	assert.Equal(t, "12345", getEnvValueByName(envs, "DB_PORT"))
	assert.Equal(t, "test", getEnvValueByName(envs, "DB_DATABASE"))
}

func getEnvValueByName(envs []v1.EnvVar, name string) string {
	for _, v := range envs {
		if v.Name == name {
			return v.Value
		}
	}
	return ""
}
