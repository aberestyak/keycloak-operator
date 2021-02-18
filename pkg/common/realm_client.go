package common

import (
	"encoding/json"
	"fmt"

	"github.com/aberestyak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	"github.com/pkg/errors"
)

func (c *Client) CreateRealm(realm *v1alpha1.KeycloakRealm) (string, error) {
	return c.create(realm.Spec.Realm, "realms", "realm")
}

func (c *Client) DeleteRealm(realmName string) error {
	err := c.delete(fmt.Sprintf("realms/%s", realmName), "realm", nil)
	return err
}

// UpdateRealmGroups - create or update realm groups and their's childrens
func (c *Client) UpdateRealmGroups(realm *v1alpha1.KeycloakRealm) error {
	existingGroups, err := c.ListRealmGroups(realm.Spec.Realm.Realm)
	if err != nil {
		return err
	}
	for _, newGroup := range realm.Spec.Realm.Groups {
		found := false
		for _, existingGroup := range existingGroups {
			if newGroup.Name == existingGroup.Name {
				found = true
				c.update(newGroup, fmt.Sprintf("realms/%s/groups/%s", realm.Spec.Realm.ID, existingGroup.ID), "group")
				// create child groups, if they not exist
				for _, newChildGroup := range newGroup.SubGroup {
					childFound := false
					for _, existingChildGroup := range existingGroup.SubGroup {
						if newChildGroup.Name == existingChildGroup.Name {
							childFound = true
							break
						}
					}
					// Create new child group
					if !childFound {
						c.create(newChildGroup, fmt.Sprintf("realms/%s/groups/%s/children", realm.Spec.Realm.ID, existingGroup.ID), "group")
					}
				}
				break
			}
		}
		// Create new group
		if !found {
			id, _ := c.create(newGroup, fmt.Sprintf("realms/%s/groups", realm.Spec.Realm.ID), "group")
			for _, newChildGroup := range newGroup.SubGroup {
				c.create(newChildGroup, fmt.Sprintf("realms/%s/groups/%s/children", realm.Spec.Realm.ID, id), "group")
			}
		}
	}
	return nil
}

// UpdateRealm- update realm
func (c *Client) UpdateRealm(realm *v1alpha1.KeycloakRealm) error {
	return c.update(realm.Spec.Realm, fmt.Sprintf("realms/%s", realm.Spec.Realm.ID), "realm")
}

func (c *Client) GetRealm(realmName string) (*v1alpha1.KeycloakRealm, error) {
	result, err := c.get(fmt.Sprintf("realms/%s", realmName), "realm", func(body []byte) (T, error) {
		realm := &v1alpha1.KeycloakAPIRealm{}
		err := json.Unmarshal(body, realm)
		return realm, err
	})
	if result == nil {
		return nil, nil
	}
	ret := &v1alpha1.KeycloakRealm{
		Spec: v1alpha1.KeycloakRealmSpec{
			Realm: result.(*v1alpha1.KeycloakAPIRealm),
		},
	}
	return ret, err
}

// func (c *Client) GetUserFederation(realmName string) (*v1alpha1.KeycloakAPIUserFederationProvider, error) {
// 	result, err := c.get(fmt.Sprintf("realms/%s", realmName), "realm", func(body []byte) (T, error) {
// 		realm := &v1alpha1.KeycloakAPIRealm{}
// 		err := json.Unmarshal(body, realm)
// 		return realm, err
// 	})
// 	if result == nil {
// 		return nil, nil
// 	}
// 	ret := &v1alpha1.KeycloakRealm{
// 		Spec: v1alpha1.KeycloakRealmSpec{
// 			Realm: result.(*v1alpha1.KeycloakAPIRealm),
// 		},
// 	}
// 	return ret, err
// }

// func (c *Client) GetUserFederatedIdentities(userID string, realmName string) ([]v1alpha1.FederatedIdentity, error) {
// 	result, err := c.get(fmt.Sprintf("realms/%s/users/%s/federated-identity", realmName, userID), "federated-identity", func(body []byte) (T, error) {
// 		var fids []v1alpha1.FederatedIdentity
// 		err := json.Unmarshal(body, &fids)
// 		return fids, err
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
// 	return result.([]v1alpha1.FederatedIdentity), err
// }

// ListGroups - get slice of realm groups
func (c *Client) ListRealmGroups(realmName string) ([]*v1alpha1.KeycloakRealmGroup, error) {
	result, err := c.list(fmt.Sprintf("realms/%s/groups", realmName), "groups", func(body []byte) (T, error) {
		var groups []*v1alpha1.KeycloakRealmGroup
		err := json.Unmarshal(body, &groups)
		return groups, err
	})

	if err != nil {
		return nil, err
	}

	res, ok := result.([]*v1alpha1.KeycloakRealmGroup)

	if !ok {
		return nil, errors.Errorf("error decoding list groups response")
	}

	return res, nil
}

func (c *Client) ListRealms() ([]*v1alpha1.KeycloakRealm, error) {
	result, err := c.list("realms", "realm", func(body []byte) (T, error) {
		var realms []*v1alpha1.KeycloakRealm
		err := json.Unmarshal(body, &realms)
		return realms, err
	})
	resultAsRealm, ok := result.([]*v1alpha1.KeycloakRealm)
	if !ok {
		return nil, err
	}
	return resultAsRealm, err
}
