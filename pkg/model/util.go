package model

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"strings"
	"unicode"

	v1 "k8s.io/api/core/v1"
)

// Copy pasted from https://blog.questionable.services/article/generating-secure-random-numbers-crypto-rand/

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomString(s int) string {
	b := GenerateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b)
}

func GetRealmUserSecretName(keycloakNamespace, realmName, userName string) string {
	return SanitizeResourceName(fmt.Sprintf("credential-%v-%v-%v",
		realmName,
		userName,
		keycloakNamespace))
}

func SanitizeNumberOfReplicas(numberOfReplicas int, isCreate bool) *int32 {
	numberOfReplicasCasted := int32(numberOfReplicas)
	if isCreate && numberOfReplicasCasted < 1 {
		numberOfReplicasCasted = 1
	}
	return &[]int32{numberOfReplicasCasted}[0]
}

func SanitizeResourceName(name string) string {
	sb := strings.Builder{}
	for _, char := range name {
		ascii := int(char)
		// number
		if ascii >= 48 && ascii <= 57 {
			sb.WriteRune(char)
			continue
		}

		// Uppercase letters are transformed to lowercase
		if ascii >= 65 && ascii <= 90 {
			sb.WriteRune(unicode.ToLower(char))
			continue
		}

		// Lowercase letters
		if ascii >= 97 && ascii <= 122 {
			sb.WriteRune(char)
			continue
		}

		// dash
		if ascii == 45 {
			sb.WriteRune(char)
			continue
		}

		if char == '.' {
			sb.WriteRune(char)
			continue
		}

		if ascii == '_' {
			sb.WriteRune('-')
			continue
		}

		// Ignore all invalid chars
		continue
	}

	return sb.String()
}

func IsIP(host []byte) bool {
	return net.ParseIP(string(host)) != nil
}

func GetExternalDatabaseHost(secret *v1.Secret) string {
	host := secret.Data[DatabaseSecretExternalAddressProperty]
	return string(host)
}

func GetExternalDatabaseName(secret *v1.Secret) string {
	if secret == nil {
		return PostgresqlDatabase
	}

	name := secret.Data[DatabaseSecretDatabaseProperty]
	return string(name)
}

func GetExternalDatabasePort(secret *v1.Secret) int32 {
	if secret == nil {
		return PostgresDefaultPort
	}

	port := secret.Data[DatabaseSecretExternalPortProperty]
	parsed, err := strconv.ParseInt(string(port), 10, 32)
	if err != nil {
		return PostgresDefaultPort
	}
	return int32(parsed)
}

// This function favors values in "a".
func MergeEnvs(a []v1.EnvVar, b []v1.EnvVar) []v1.EnvVar {
	for _, bb := range b {
		found := false
		for _, aa := range a {
			if aa.Name == bb.Name {
				aa.Value = bb.Value
				found = true
				break
			}
		}
		if !found {
			a = append(a, bb)
		}
	}
	return a
}

func MergePorts(a []v1.ContainerPort, b []v1.ContainerPort) []v1.ContainerPort {
	for _, bb := range b {
		found := false
		for _, aa := range a {
			if aa.ContainerPort == bb.ContainerPort {
				aa.Name = bb.Name
				found = true
				break
			}
		}
		if !found {
			a = append(a, bb)
		}
	}
	return a
}

func MergeVolumeMounts(a []v1.VolumeMount, b []v1.VolumeMount) []v1.VolumeMount {
	for _, bb := range b {
		found := false
		for _, aa := range a {
			if aa.Name == bb.Name {
				aa.Name = bb.Name
				found = true
				break
			}
		}
		if !found {
			a = append(a, bb)
		}
	}
	return a
}

func MergeVolumes(a []v1.Volume, b []v1.Volume) []v1.Volume {
	for _, bb := range b {
		found := false
		for _, aa := range a {
			if aa.Name == bb.Name {
				aa.Name = bb.Name
				found = true
				break
			}
		}
		if !found {
			a = append(a, bb)
		}
	}
	return a
}
