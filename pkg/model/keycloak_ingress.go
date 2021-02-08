package model

import (
	kc "github.com/berestyak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	"k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func KeycloakIngress(cr *kc.Keycloak) *v1beta1.Ingress {
	ingressHost := cr.Spec.ExternalAccess.Host
	if ingressHost == "" {
		ingressHost = IngressDefaultHost
	}

	return &v1beta1.Ingress{
		ObjectMeta: v1.ObjectMeta{
			Name:      ApplicationName,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"app": ApplicationName,
			},
			Annotations: cr.Spec.ExternalAccess.Annotations,
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: ingressHost,
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/",
									Backend: v1beta1.IngressBackend{
										ServiceName: ApplicationName,
										ServicePort: intstr.FromInt(KeycloakHTTPServicePort),
									},
								},
							},
						},
					},
				},
			},
			TLS: []v1beta1.IngressTLS{
				{
					Hosts:      []string{ingressHost},
					SecretName: ingressHost + "-tls",
				},
			},
		},
	}
}

func KeycloakIngressReconciled(cr *kc.Keycloak, currentState *v1beta1.Ingress) *v1beta1.Ingress {
	reconciled := currentState.DeepCopy()
	reconciledHost := currentState.Spec.Rules[0].Host
	reconciledSpecTLS := currentState.Spec.TLS
	reconciled.ObjectMeta.Annotations = cr.Spec.ExternalAccess.Annotations
	reconciled.Spec = v1beta1.IngressSpec{
		TLS: reconciledSpecTLS,
		Rules: []v1beta1.IngressRule{
			{
				Host: reconciledHost,
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{
							{
								Path: "/",
								Backend: v1beta1.IngressBackend{
									ServiceName: ApplicationName,
									ServicePort: intstr.FromInt(KeycloakHTTPServicePort),
								},
							},
						},
					},
				},
			},
		},
	}

	return reconciled
}

func KeycloakIngressSelector(cr *kc.Keycloak) client.ObjectKey {
	return client.ObjectKey{
		Name:      ApplicationName,
		Namespace: cr.Namespace,
	}
}
