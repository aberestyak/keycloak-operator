package keycloakrealm

import (
	"fmt"

	kc "github.com/aberestyak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	"github.com/aberestyak/keycloak-operator/pkg/common"
	"github.com/aberestyak/keycloak-operator/pkg/model"
)

type Reconciler interface {
	Reconcile(cr *kc.KeycloakRealm) error
}

type KeycloakRealmReconciler struct { // nolint
	Keycloak kc.Keycloak
}

func NewKeycloakRealmReconciler(keycloak kc.Keycloak) *KeycloakRealmReconciler {
	return &KeycloakRealmReconciler{
		Keycloak: keycloak,
	}
}

func (i *KeycloakRealmReconciler) Reconcile(state *common.RealmState, cr *kc.KeycloakRealm) common.DesiredClusterState {
	desired := common.DesiredClusterState{}
	// Create realm
	if cr.DeletionTimestamp == nil {
		desired.AddAction(i.getKeycloakDesiredState())
		desired.AddAction(i.getNewRealmState(state, cr))
		for _, user := range cr.Spec.Realm.Users {
			desired.AddAction(i.getDesiredUserSate(state, cr, user))
		}

		desired.AddAction(i.getBrowserRedirectorDesiredState(state, cr))
	}
	// Delete realm
	if cr.DeletionTimestamp != nil {
		desired.AddAction(i.getDeletedRealmState(state, cr))
		return desired
	}
	// Manage realm groups
	if cr.Spec.Realm.Groups != nil {
		desired.AddAction(i.getDesiredRealmGroups(state, cr))
	}
	// Update realm config if Federation provider
	if cr.Spec.Realm.UserFederationProviders != nil {
		desired.AddAction(i.getDesiredRealmState(state, cr))
	}
	return desired
}

func (i *KeycloakRealmReconciler) ReconcileRealmDelete(state *common.RealmState, cr *kc.KeycloakRealm) common.DesiredClusterState {
	desired := common.DesiredClusterState{}
	desired.AddAction(i.getKeycloakDesiredState())
	desired.AddAction(i.getDesiredRealmState(state, cr))
	return desired
}

// Always make sure keycloak is able to respond
func (i *KeycloakRealmReconciler) getKeycloakDesiredState() common.ClusterAction {
	return &common.PingAction{
		Msg: "check if keycloak is available",
	}
}

// Configure the browser redirector if provided by the user
func (i *KeycloakRealmReconciler) getBrowserRedirectorDesiredState(state *common.RealmState, cr *kc.KeycloakRealm) common.ClusterAction {
	if len(cr.Spec.RealmOverrides) == 0 {
		return nil
	}

	if state.Realm != nil {
		return nil
	}

	return &common.ConfigureRealmAction{
		Ref: cr,
		Msg: "configure browser redirector",
	}
}

func (i *KeycloakRealmReconciler) getNewRealmState(state *common.RealmState, cr *kc.KeycloakRealm) common.ClusterAction {

	if state.Realm == nil {
		return &common.CreateRealmAction{
			Ref: cr,
			Msg: fmt.Sprintf("create realm %v/%v", cr.Namespace, cr.Spec.Realm.Realm),
		}
	}
	return nil
}

func (i *KeycloakRealmReconciler) getDesiredUserSate(state *common.RealmState, cr *kc.KeycloakRealm, user *kc.KeycloakAPIUser) common.ClusterAction {
	val, ok := state.RealmUserSecrets[user.UserName]
	if !ok || val == nil {
		return &common.GenericCreateAction{
			Ref: model.RealmCredentialSecret(cr, user, &i.Keycloak),
			Msg: fmt.Sprintf("create credential secret for user %v in realm %v/%v", user.UserName, cr.Namespace, cr.Spec.Realm.Realm),
		}
	}

	return nil
}

func (i *KeycloakRealmReconciler) getDeletedRealmState(state *common.RealmState, cr *kc.KeycloakRealm) common.ClusterAction {
	return common.DeleteRealmAction{
		Ref: cr,
		Msg: fmt.Sprintf("removing realm %v/%v", cr.Namespace, cr.Spec.Realm.Realm),
	}
}

func (i *KeycloakRealmReconciler) getDesiredRealmGroups(state *common.RealmState, cr *kc.KeycloakRealm) common.ClusterAction {
	return &common.UpdateRealmGroupsAction{
		Ref:   cr,
		Realm: cr.Spec.Realm.Realm,
		Msg:   fmt.Sprintf("update realm groups %v/%v", cr.Namespace, cr.Spec.Realm.Realm),
	}
}

func (i *KeycloakRealmReconciler) getDesiredRealmState(state *common.RealmState, cr *kc.KeycloakRealm) common.ClusterAction {
	return &common.UpdateRealmAction{
		Ref:   cr,
		Realm: cr.Spec.Realm.Realm,
		Msg:   fmt.Sprintf("update realm %v/%v", cr.Namespace, cr.Spec.Realm.Realm),
	}
}
