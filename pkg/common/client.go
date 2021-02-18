package common

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aberestyak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	"github.com/aberestyak/keycloak-operator/pkg/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	config2 "sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	authURL = "auth/realms/master/protocol/openid-connect/token"
)

type Requester interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	requester Requester
	URL       string
	token     string
}

// T is a generic type for keycloak spec resources
type T interface{}

//go:generate moq -out keycloakClient_moq.go . KeycloakInterface
type KeycloakInterface interface {
	Ping() error

	Endpoint() string

	CreateRealm(realm *v1alpha1.KeycloakRealm) (string, error)
	GetRealm(realmName string) (*v1alpha1.KeycloakRealm, error)
	UpdateRealmGroups(specRealm *v1alpha1.KeycloakRealm) error
	UpdateRealm(specRealm *v1alpha1.KeycloakRealm) error
	DeleteRealm(realmName string) error
	ListRealms() ([]*v1alpha1.KeycloakRealm, error)

	CreateClient(client *v1alpha1.KeycloakAPIClient, realmName string) (string, error)
	GetClient(clientID, realmName string) (*v1alpha1.KeycloakAPIClient, error)
	GetClientSecret(clientID, realmName string) (string, error)
	GetClientInstall(clientID, realmName string) ([]byte, error)
	UpdateClient(specClient *v1alpha1.KeycloakAPIClient, realmName string) error
	DeleteClient(clientID, realmName string) error
	ListClients(realmName string) ([]*v1alpha1.KeycloakAPIClient, error)

	CreateUser(user *v1alpha1.KeycloakAPIUser, realmName string) (string, error)
	CreateFederatedIdentity(fid v1alpha1.FederatedIdentity, userID string, realmName string) (string, error)
	RemoveFederatedIdentity(fid v1alpha1.FederatedIdentity, userID string, realmName string) error
	GetUserFederatedIdentities(userName string, realmName string) ([]v1alpha1.FederatedIdentity, error)
	UpdatePassword(user *v1alpha1.KeycloakAPIUser, realmName, newPass string) error
	FindUserByEmail(email, realm string) (*v1alpha1.KeycloakAPIUser, error)
	FindUserByUsername(name, realm string) (*v1alpha1.KeycloakAPIUser, error)
	GetUser(userID, realmName string) (*v1alpha1.KeycloakAPIUser, error)
	UpdateUser(specUser *v1alpha1.KeycloakAPIUser, realmName string) error
	DeleteUser(userID, realmName string) error
	ListUsers(realmName string) ([]*v1alpha1.KeycloakAPIUser, error)

	CreateIdentityProvider(identityProvider *v1alpha1.KeycloakIdentityProvider, realmName string) (string, error)
	GetIdentityProvider(alias, realmName string) (*v1alpha1.KeycloakIdentityProvider, error)
	UpdateIdentityProvider(specIdentityProvider *v1alpha1.KeycloakIdentityProvider, realmName string) error
	DeleteIdentityProvider(alias, realmName string) error
	ListIdentityProviders(realmName string) ([]*v1alpha1.KeycloakIdentityProvider, error)

	CreateUserClientRole(role *v1alpha1.KeycloakUserRole, realmName, clientID, userID string) (string, error)
	ListUserClientRoles(realmName, clientID, userID string) ([]*v1alpha1.KeycloakUserRole, error)
	ListAvailableUserClientRoles(realmName, clientID, userID string) ([]*v1alpha1.KeycloakUserRole, error)
	DeleteUserClientRole(role *v1alpha1.KeycloakUserRole, realmName, clientID, userID string) error

	CreateUserRealmRole(role *v1alpha1.KeycloakUserRole, realmName, userID string) (string, error)
	ListUserRealmRoles(realmName, userID string) ([]*v1alpha1.KeycloakUserRole, error)
	ListAvailableUserRealmRoles(realmName, userID string) ([]*v1alpha1.KeycloakUserRole, error)
	DeleteUserRealmRole(role *v1alpha1.KeycloakUserRole, realmName, userID string) error

	ListAuthenticationExecutionsForFlow(flowAlias, realmName string) ([]*v1alpha1.AuthenticationExecutionInfo, error)

	CreateAuthenticatorConfig(authenticatorConfig *v1alpha1.AuthenticatorConfig, realmName, executionID string) (string, error)
	GetAuthenticatorConfig(configID, realmName string) (*v1alpha1.AuthenticatorConfig, error)
	UpdateAuthenticatorConfig(authenticatorConfig *v1alpha1.AuthenticatorConfig, realmName string) error
	DeleteAuthenticatorConfig(configID, realmName string) error
}

//go:generate moq -out keycloakClientFactory_moq.go . KeycloakClientFactory

//KeycloakClientFactory interface
type KeycloakClientFactory interface {
	AuthenticatedClient(kc v1alpha1.Keycloak) (KeycloakInterface, error)
}

type LocalConfigKeycloakFactory struct {
}

func (c *Client) Endpoint() string {
	return c.URL
}

func (c *Client) CreateClient(client *v1alpha1.KeycloakAPIClient, realmName string) (string, error) {
	return c.create(client, fmt.Sprintf("realms/%s/clients", realmName), "client")
}

func (c *Client) CreateUser(user *v1alpha1.KeycloakAPIUser, realmName string) (string, error) {
	return c.create(user, fmt.Sprintf("realms/%s/users", realmName), "user")
}

func (c *Client) CreateFederatedIdentity(fid v1alpha1.FederatedIdentity, userID string, realmName string) (string, error) {
	return c.create(fid, fmt.Sprintf("realms/%s/users/%s/federated-identity/%s", realmName, userID, fid.IdentityProvider), "federated-identity")
}

func (c *Client) RemoveFederatedIdentity(fid v1alpha1.FederatedIdentity, userID string, realmName string) error {
	return c.delete(fmt.Sprintf("realms/%s/users/%s/federated-identity/%s", realmName, userID, fid.IdentityProvider), "federated-identity", fid)
}

func (c *Client) GetUserFederatedIdentities(userID string, realmName string) ([]v1alpha1.FederatedIdentity, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/users/%s/federated-identity", realmName, userID), "federated-identity", func(body []byte) (T, error) {
		var fids []v1alpha1.FederatedIdentity
		err := json.Unmarshal(body, &fids)
		return fids, err
	})
	if err != nil {
		return nil, err
	}
	return result.([]v1alpha1.FederatedIdentity), err
}

func (c *Client) CreateUserClientRole(role *v1alpha1.KeycloakUserRole, realmName, clientID, userID string) (string, error) {
	return c.create(
		[]*v1alpha1.KeycloakUserRole{role},
		fmt.Sprintf("realms/%s/users/%s/role-mappings/clients/%s", realmName, userID, clientID),
		"user-client-role",
	)
}
func (c *Client) CreateUserRealmRole(role *v1alpha1.KeycloakUserRole, realmName, userID string) (string, error) {
	return c.create(
		[]*v1alpha1.KeycloakUserRole{role},
		fmt.Sprintf("realms/%s/users/%s/role-mappings/realm", realmName, userID),
		"user-realm-role",
	)
}

func (c *Client) CreateAuthenticatorConfig(authenticatorConfig *v1alpha1.AuthenticatorConfig, realmName, executionID string) (string, error) {
	return c.create(authenticatorConfig, fmt.Sprintf("realms/%s/authentication/executions/%s/config", realmName, executionID), "AuthenticatorConfig")
}

func (c *Client) DeleteUserClientRole(role *v1alpha1.KeycloakUserRole, realmName, clientID, userID string) error {
	err := c.delete(
		fmt.Sprintf("realms/%s/users/%s/role-mappings/clients/%s", realmName, userID, clientID),
		"user-client-role",
		[]*v1alpha1.KeycloakUserRole{role},
	)
	return err
}

func (c *Client) DeleteUserRealmRole(role *v1alpha1.KeycloakUserRole, realmName, userID string) error {
	err := c.delete(
		fmt.Sprintf("realms/%s/users/%s/role-mappings/realm", realmName, userID),
		"user-realm-role",
		[]*v1alpha1.KeycloakUserRole{role},
	)
	return err
}

func (c *Client) UpdatePassword(user *v1alpha1.KeycloakAPIUser, realmName, newPass string) error {
	passReset := &v1alpha1.KeycloakAPIPasswordReset{}
	passReset.Type = "password"
	passReset.Temporary = false
	passReset.Value = newPass
	u := fmt.Sprintf("realms/%s/users/%s/reset-password", realmName, user.ID)
	if err := c.update(passReset, u, "paswordreset"); err != nil {
		return errors.Wrap(err, "error calling keycloak api ")
	}
	return nil
}

func (c *Client) FindUserByEmail(email, realm string) (*v1alpha1.KeycloakAPIUser, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/users?first=0&max=1&search=%s", realm, email), "user", func(body []byte) (T, error) {
		var users []*v1alpha1.KeycloakAPIUser
		if err := json.Unmarshal(body, &users); err != nil {
			return nil, err
		}
		if len(users) == 0 {
			return nil, nil
		}
		return users[0], nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, err
	}
	return result.(*v1alpha1.KeycloakAPIUser), nil
}

func (c *Client) FindUserByUsername(name, realm string) (*v1alpha1.KeycloakAPIUser, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/users?username=%s&max=-1", realm, name), "user", func(body []byte) (T, error) {
		var users []*v1alpha1.KeycloakAPIUser
		if err := json.Unmarshal(body, &users); err != nil {
			return nil, err
		}

		for _, user := range users {
			if user.UserName == name {
				return user, nil
			}
		}
		return nil, errors.Errorf("not found")
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(*v1alpha1.KeycloakAPIUser), nil
}

func (c *Client) CreateIdentityProvider(identityProvider *v1alpha1.KeycloakIdentityProvider, realmName string) (string, error) {
	return c.create(identityProvider, fmt.Sprintf("realms/%s/identity-provider/instances", realmName), "identity provider")
}

func (c *Client) GetClient(clientID, realmName string) (*v1alpha1.KeycloakAPIClient, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/clients/%s", realmName, clientID), "client", func(body []byte) (T, error) {
		client := &v1alpha1.KeycloakAPIClient{}
		err := json.Unmarshal(body, client)
		return client, err
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	ret := result.(*v1alpha1.KeycloakAPIClient)
	return ret, err
}

func (c *Client) GetClientSecret(clientID, realmName string) (string, error) {
	//"https://{{ rhsso_route }}/auth/admin/realms/{{ rhsso_realm }}/clients/{{ rhsso_client_id }}/client-secret"
	result, err := c.get(fmt.Sprintf("realms/%s/clients/%s/client-secret", realmName, clientID), "client-secret", func(body []byte) (T, error) {
		res := map[string]string{}
		if err := json.Unmarshal(body, &res); err != nil {
			return nil, err
		}
		return res["value"], nil
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to get: "+fmt.Sprintf("realms/%s/clients/%s/client-secret", realmName, clientID))
	}
	if result == nil {
		return "", nil
	}
	return result.(string), nil
}

func (c *Client) GetClientInstall(clientID, realmName string) ([]byte, error) {
	var response []byte
	if _, err := c.get(fmt.Sprintf("realms/%s/clients/%s/installation/providers/keycloak-oidc-keycloak-json", realmName, clientID), "client-installation", func(body []byte) (T, error) {
		response = body
		return body, nil
	}); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) GetUser(userID, realmName string) (*v1alpha1.KeycloakAPIUser, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/users/%s", realmName, userID), "user", func(body []byte) (T, error) {
		user := &v1alpha1.KeycloakAPIUser{}
		err := json.Unmarshal(body, user)
		return user, err
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	ret := result.(*v1alpha1.KeycloakAPIUser)
	return ret, err
}

func (c *Client) GetIdentityProvider(alias string, realmName string) (*v1alpha1.KeycloakIdentityProvider, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/identity-provider/instances/%s", realmName, alias), "identity provider", func(body []byte) (T, error) {
		provider := &v1alpha1.KeycloakIdentityProvider{}
		err := json.Unmarshal(body, provider)
		return provider, err
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(*v1alpha1.KeycloakIdentityProvider), err
}

func (c *Client) GetAuthenticatorConfig(configID, realmName string) (*v1alpha1.AuthenticatorConfig, error) {
	result, err := c.get(fmt.Sprintf("realms/%s/authentication/config/%s", realmName, configID), "AuthenticatorConfig", func(body []byte) (T, error) {
		authenticatorConfig := &v1alpha1.AuthenticatorConfig{}
		err := json.Unmarshal(body, authenticatorConfig)
		return authenticatorConfig, err
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(*v1alpha1.AuthenticatorConfig), err
}

func (c *Client) UpdateClient(specClient *v1alpha1.KeycloakAPIClient, realmName string) error {
	return c.update(specClient, fmt.Sprintf("realms/%s/clients/%s", realmName, specClient.ID), "client")
}

func (c *Client) UpdateUser(specUser *v1alpha1.KeycloakAPIUser, realmName string) error {
	return c.update(specUser, fmt.Sprintf("realms/%s/users/%s", realmName, specUser.ID), "user")
}

func (c *Client) UpdateIdentityProvider(specIdentityProvider *v1alpha1.KeycloakIdentityProvider, realmName string) error {
	return c.update(specIdentityProvider, fmt.Sprintf("realms/%s/identity-provider/instances/%s", realmName, specIdentityProvider.Alias), "identity provider")
}

func (c *Client) UpdateAuthenticatorConfig(authenticatorConfig *v1alpha1.AuthenticatorConfig, realmName string) error {
	return c.update(authenticatorConfig, fmt.Sprintf("realms/%s/authentication/config/%s", realmName, authenticatorConfig.ID), "AuthenticatorConfig")
}

func (c *Client) DeleteClient(clientID, realmName string) error {
	err := c.delete(fmt.Sprintf("realms/%s/clients/%s", realmName, clientID), "client", nil)
	return err
}

func (c *Client) DeleteUser(userID, realmName string) error {
	err := c.delete(fmt.Sprintf("realms/%s/users/%s", realmName, userID), "user", nil)
	return err
}

func (c *Client) DeleteIdentityProvider(alias string, realmName string) error {
	err := c.delete(fmt.Sprintf("realms/%s/identity-provider/instances/%s", realmName, alias), "identity provider", nil)
	return err
}

func (c *Client) DeleteAuthenticatorConfig(configID, realmName string) error {
	err := c.delete(fmt.Sprintf("realms/%s/authentication/config/%s", realmName, configID), "AuthenticatorConfig", nil)
	return err
}

func (c *Client) ListClients(realmName string) ([]*v1alpha1.KeycloakAPIClient, error) {
	result, err := c.list(fmt.Sprintf("realms/%s/clients", realmName), "clients", func(body []byte) (T, error) {
		var clients []*v1alpha1.KeycloakAPIClient
		err := json.Unmarshal(body, &clients)
		return clients, err
	})

	if err != nil {
		return nil, err
	}

	res, ok := result.([]*v1alpha1.KeycloakAPIClient)

	if !ok {
		return nil, errors.Errorf("error decoding list clients response")
	}

	return res, nil
}

func (c *Client) ListUsers(realmName string) ([]*v1alpha1.KeycloakAPIUser, error) {
	result, err := c.list(fmt.Sprintf("realms/%s/users", realmName), "users", func(body []byte) (T, error) {
		var users []*v1alpha1.KeycloakAPIUser
		err := json.Unmarshal(body, &users)
		return users, err
	})
	if err != nil {
		return nil, err
	}
	return result.([]*v1alpha1.KeycloakAPIUser), err
}

func (c *Client) ListIdentityProviders(realmName string) ([]*v1alpha1.KeycloakIdentityProvider, error) {
	result, err := c.list(fmt.Sprintf("realms/%s/identity-provider/instances", realmName), "identity providers", func(body []byte) (T, error) {
		var providers []*v1alpha1.KeycloakIdentityProvider
		err := json.Unmarshal(body, &providers)
		return providers, err
	})
	if err != nil {
		return nil, err
	}
	return result.([]*v1alpha1.KeycloakIdentityProvider), err
}

func (c *Client) ListUserClientRoles(realmName, clientID, userID string) ([]*v1alpha1.KeycloakUserRole, error) {
	objects, err := c.list("realms/"+realmName+"/users/"+userID+"/role-mappings/clients/"+clientID, "userClientRoles", func(body []byte) (t T, e error) {
		var userClientRoles []*v1alpha1.KeycloakUserRole
		err := json.Unmarshal(body, &userClientRoles)
		return userClientRoles, err
	})
	if err != nil {
		return nil, err
	}
	if objects == nil {
		return nil, nil
	}
	return objects.([]*v1alpha1.KeycloakUserRole), err
}

func (c *Client) ListAvailableUserClientRoles(realmName, clientID, userID string) ([]*v1alpha1.KeycloakUserRole, error) {
	objects, err := c.list("realms/"+realmName+"/users/"+userID+"/role-mappings/clients/"+clientID+"/available", "userClientRoles", func(body []byte) (t T, e error) {
		var userClientRoles []*v1alpha1.KeycloakUserRole
		err := json.Unmarshal(body, &userClientRoles)
		return userClientRoles, err
	})
	if err != nil {
		return nil, err
	}
	if objects == nil {
		return nil, nil
	}
	return objects.([]*v1alpha1.KeycloakUserRole), err
}

func (c *Client) ListUserRealmRoles(realmName, userID string) ([]*v1alpha1.KeycloakUserRole, error) {
	objects, err := c.list("realms/"+realmName+"/users/"+userID+"/role-mappings/realm", "userRealmRoles", func(body []byte) (t T, e error) {
		var userRealmRoles []*v1alpha1.KeycloakUserRole
		err := json.Unmarshal(body, &userRealmRoles)
		return userRealmRoles, err
	})
	if err != nil {
		return nil, err
	}
	if objects == nil {
		return nil, nil
	}
	return objects.([]*v1alpha1.KeycloakUserRole), err
}

func (c *Client) ListAvailableUserRealmRoles(realmName, userID string) ([]*v1alpha1.KeycloakUserRole, error) {
	objects, err := c.list("realms/"+realmName+"/users/"+userID+"/role-mappings/realm/available", "userClientRoles", func(body []byte) (t T, e error) {
		var userRealmRoles []*v1alpha1.KeycloakUserRole
		err := json.Unmarshal(body, &userRealmRoles)
		return userRealmRoles, err
	})
	if err != nil {
		return nil, err
	}
	if objects == nil {
		return nil, nil
	}
	return objects.([]*v1alpha1.KeycloakUserRole), err
}

func (c *Client) ListAuthenticationExecutionsForFlow(flowAlias, realmName string) ([]*v1alpha1.AuthenticationExecutionInfo, error) {
	result, err := c.list(fmt.Sprintf("realms/%s/authentication/flows/%s/executions", realmName, flowAlias), "AuthenticationExecution", func(body []byte) (T, error) {
		var authenticationExecutions []*v1alpha1.AuthenticationExecutionInfo
		err := json.Unmarshal(body, &authenticationExecutions)
		return authenticationExecutions, err
	})
	if err != nil {
		return nil, err
	}
	return result.([]*v1alpha1.AuthenticationExecutionInfo), err
}

func (c *Client) Ping() error {
	u := c.URL + "/auth/"
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		logrus.Errorf("error creating ping request %+v", err)
		return errors.Wrap(err, "error creating ping request")
	}

	res, err := c.requester.Do(req)
	if err != nil {
		logrus.Errorf("error on request %+v", err)
		return errors.Wrapf(err, "error performing ping request")
	}

	logrus.Debugf("response status: %v, %v", res.StatusCode, res.Status)
	if res.StatusCode != 200 {
		return errors.Errorf("failed to ping, response status code: %v", res.StatusCode)
	}
	defer res.Body.Close()

	return nil
}

// defaultRequester returns a default client for requesting http endpoints
func defaultRequester() Requester {
	transport := http.DefaultTransport.(*http.Transport).Clone()      // nolint
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // nolint

	c := &http.Client{Transport: transport, Timeout: time.Second * 10}
	return c
}

// AuthenticatedClient returns an authenticated client for requesting endpoints from the Keycloak api
func (i *LocalConfigKeycloakFactory) AuthenticatedClient(kc v1alpha1.Keycloak) (KeycloakInterface, error) {
	config, err := config2.GetConfig()
	if err != nil {
		return nil, err
	}

	secretClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	var credentialSecret, endpoint string
	if kc.Spec.External.Enabled {
		credentialSecret = "credential-" + kc.Name
		endpoint = kc.Spec.External.URL
	} else {
		credentialSecret = kc.Status.CredentialSecret
		endpoint = kc.Status.InternalURL
	}

	adminCreds, err := secretClient.CoreV1().Secrets(kc.Namespace).Get(context.TODO(), credentialSecret, v12.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the admin credentials")
	}
	user := string(adminCreds.Data[model.AdminUsernameProperty])
	pass := string(adminCreds.Data[model.AdminPasswordProperty])
	client := &Client{
		URL:       endpoint,
		requester: defaultRequester(),
	}
	if err := client.login(user, pass); err != nil {
		return nil, err
	}
	return client, nil
}
