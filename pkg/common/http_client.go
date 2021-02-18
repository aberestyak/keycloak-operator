package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/aberestyak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Generic create function for creating new Keycloak resources
func (c *Client) create(obj T, resourcePath, resourceName string) (string, error) {
	jsonValue, err := json.Marshal(obj)
	if err != nil {
		logrus.Errorf("error %+v marshalling object", err)
		return "", nil
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/auth/admin/%s", c.URL, resourcePath),
		bytes.NewBuffer(jsonValue),
	)
	if err != nil {
		logrus.Errorf("error creating POST %s request %+v", resourceName, err)
		return "", errors.Wrapf(err, "error creating POST %s request", resourceName)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	res, err := c.requester.Do(req)

	if err != nil {
		logrus.Errorf("error on request %+v", err)
		return "", errors.Wrapf(err, "error performing POST %s request", resourceName)
	}
	defer res.Body.Close()

	if res.StatusCode == 409 {
		log.Info("Object", resourceName, "already exists")
	}

	if res.StatusCode != 201 && res.StatusCode != 204 && res.StatusCode != 409 {
		return "", errors.Errorf("failed to create %s: (%d) %s", resourceName, res.StatusCode, res.Status)
	}

	if resourceName == "client" {
		d, _ := ioutil.ReadAll(res.Body)
		fmt.Println("user response ", string(d))
	}

	location := strings.Split(res.Header.Get("Location"), "/")
	uid := location[len(location)-1]
	return uid, nil
}

// Generic put function for updating Keycloak resources
func (c *Client) update(obj T, resourcePath, resourceName string) error {
	jsonValue, err := json.Marshal(obj)
	if err != nil {
		return nil
	}
	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("%s/auth/admin/%s", c.URL, resourcePath),
		bytes.NewBuffer(jsonValue),
	)
	if err != nil {
		logrus.Errorf("error creating UPDATE %s request %+v", resourceName, err)
		return errors.Wrapf(err, "error creating UPDATE %s request", resourceName)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+c.token)
	res, err := c.requester.Do(req)
	fmt.Println(string(jsonValue))
	if err != nil {
		logrus.Errorf("error on request %+v", err)
		return errors.Wrapf(err, "error performing UPDATE %s request", resourceName)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		logrus.Errorf("failed to UPDATE %s %v", resourceName, res.Status)
		return errors.Errorf("failed to UPDATE %s: (%d) %s", resourceName, res.StatusCode, res.Status)
	}

	return nil
}

// Generic list function for listing Keycloak resources
func (c *Client) list(resourcePath, resourceName string, unMarshalListFunc func(body []byte) (T, error)) (T, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/auth/admin/%s", c.URL, resourcePath),
		nil,
	)
	if err != nil {
		logrus.Errorf("error creating LIST %s request %+v", resourceName, err)
		return nil, errors.Wrapf(err, "error creating LIST %s request", resourceName)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	res, err := c.requester.Do(req)
	if err != nil {
		logrus.Errorf("error on request %+v", err)
		return nil, errors.Wrapf(err, "error performing LIST %s request", resourceName)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, errors.Errorf("failed to LIST %s: (%d) %s", resourceName, res.StatusCode, res.Status)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logrus.Errorf("error reading response %+v", err)
		return nil, errors.Wrapf(err, "error reading %s LIST response", resourceName)
	}

	objs, err := unMarshalListFunc(body)
	if err != nil {
		logrus.Error(err)
	}

	return objs, nil
}

// Generic get function for returning a Keycloak resource
func (c *Client) get(resourcePath, resourceName string, unMarshalFunc func(body []byte) (T, error)) (T, error) {
	u := fmt.Sprintf("%s/auth/admin/%s", c.URL, resourcePath)
	req, err := http.NewRequest(
		"GET",
		u,
		nil,
	)
	if err != nil {
		logrus.Errorf("error creating GET %s request %+v", resourceName, err)
		return nil, errors.Wrapf(err, "error creating GET %s request", resourceName)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	res, err := c.requester.Do(req)
	if err != nil {
		logrus.Errorf("error on request %+v", err)
		return nil, errors.Wrapf(err, "error performing GET %s request", resourceName)
	}

	defer res.Body.Close()
	if res.StatusCode == 404 {
		logrus.Errorf("Resource %v/%v doesn't exist", resourcePath, resourceName)
		return nil, nil
	}

	if res.StatusCode != 200 {
		return nil, errors.Errorf("failed to GET %s: (%d) %s", resourceName, res.StatusCode, res.Status)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logrus.Errorf("error reading response %+v", err)
		return nil, errors.Wrapf(err, "error reading %s GET response", resourceName)
	}

	obj, err := unMarshalFunc(body)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return obj, nil
}

// Generic delete function for deleting Keycloak resources
func (c *Client) delete(resourcePath, resourceName string, obj T) error {
	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/auth/admin/%s", c.URL, resourcePath),
		nil,
	)

	if obj != nil {
		jsonValue, err := json.Marshal(obj)
		if err != nil {
			return nil
		}
		req, err = http.NewRequest(
			"DELETE",
			fmt.Sprintf("%s/auth/admin/%s", c.URL, resourcePath),
			bytes.NewBuffer(jsonValue),
		)
		if err != nil {
			return nil
		}
		req.Header.Set("Content-Type", "application/json")
	}

	if err != nil {
		logrus.Errorf("error creating DELETE %s request %+v", resourceName, err)
		return errors.Wrapf(err, "error creating DELETE %s request", resourceName)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	res, err := c.requester.Do(req)
	if err != nil {
		logrus.Errorf("error on request %+v", err)
		return errors.Wrapf(err, "error performing DELETE %s request", resourceName)
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		logrus.Errorf("Resource %v/%v already deleted", resourcePath, resourceName)
	}
	if res.StatusCode != 204 && res.StatusCode != 404 {
		return errors.Errorf("failed to DELETE %s: (%d) %s", resourceName, res.StatusCode, res.Status)
	}

	return nil
}

// login requests a new auth token from Keycloak
func (c *Client) login(user, pass string) error {
	form := url.Values{}
	form.Add("username", user)
	form.Add("password", pass)
	form.Add("client_id", "admin-cli")
	form.Add("grant_type", "password")

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/%s", c.URL, authURL),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return errors.Wrap(err, "error creating login request")
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.requester.Do(req)
	if err != nil {
		logrus.Errorf("error on request %+v", err)
		return errors.Wrap(err, "error performing token request")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logrus.Errorf("error reading response %+v", err)
		return errors.Wrap(err, "error reading token response")
	}

	tokenRes := &v1alpha1.TokenResponse{}
	err = json.Unmarshal(body, tokenRes)
	if err != nil {
		return errors.Wrap(err, "error parsing token response")
	}

	if tokenRes.Error != "" {
		logrus.Errorf("error with request: " + tokenRes.ErrorDescription)
		return errors.Errorf(tokenRes.ErrorDescription)
	}

	c.token = tokenRes.AccessToken

	return nil
}
