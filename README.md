[![Go Report Card](https://goreportcard.com/badge/github.com/aberestyak/keycloak-operator)](https://goreportcard.com/report/github.com/keycloak/keycloak-operator)
[![Coverage Status](https://coveralls.io/repos/github/aberestyak/keycloak-operator/badge.svg?branch=master)](https://coveralls.io/github/keycloak/keycloak-operator?branch=master)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

# Keycloak Operator
A Kubernetes Operator based on the Operator SDK for creating and syncing resources in Keycloak.

## Help and Documentation

The official documentation might be found in the [here](https://www.keycloak.org/docs/latest/server_installation/index.html#_operator).

* [Keycloak documentation](https://www.keycloak.org/documentation.html)
* [User Mailing List](https://lists.jboss.org/mailman/listinfo/keycloak-user) - Mailing list for help and general questions about Keycloak
* [JIRA](https://issues.redhat.com/browse/KEYCLOAK-16220?jql=project%20%3D%20KEYCLOAK%20AND%20component%20%3D%20%22Container%20-%20Operator%22%20ORDER%20BY%20updated%20DESC) - Issue tracker for bugs and feature requests

## Supported Custom Resources
| *CustomResourceDefinition*                                            | *Description*                                            |
| --------------------------------------------------------------------- | -------------------------------------------------------- |
| [Keycloak](./deploy/crds/keycloak.org_keycloaks_crd.yaml)             | Manages, installs and configures Keycloak on the cluster |
| [KeycloakRealm](./deploy/crds/keycloak.org_keycloakrealms_crd.yaml)   | Represents a realm in a keycloak server                  |
| [KeycloakClient](./deploy/crds/keycloak.org_keycloakclients_crd.yaml) | Represents a client in a keycloak server                 |

## Deploying to a Cluster
*Note*: You will need a running Kubernetes with Postgres DB cluster to use the Operator

#### Alternative Step 2: Debug in Goland
Debug the operator in [Goland](https://www.jetbrains.com/go/)
1. go get -u github.com/go-delve/delve/cmd/dlv
2. Create new `Go Build` debug configuration
3. Change the properties to the following
```
* Name = Keycloak Operator
* Run Kind = File
* Files = <project full path>/cmd/manager/main.go
* Working Directory = <project full path>
* Environment = KUBERNETES_CONFIG=<kube config path>;WATCH_NAMESPACE=keycloak
```
3. Apply and click Debug Keycloak operator

#### Alternative Step 3: Debug in VSCode
Debug the operator in [VS Code](https://code.visualstudio.com/docs/languages/go)
1. go get -u github.com/go-delve/delve/cmd/dlv
2. Create new launch configuration, changing your kube config location
```json
{
  "name": "Keycloak Operator",
  "type": "go",
  "request": "launch",
  "mode": "auto",
  "program": "${workspaceFolder}/cmd/manager/main.go",
  "env": {
    "WATCH_NAMESPACE": "keycloak",
    "KUBERNETES_CONFIG": "<kube config path>"
  },
  "cwd": "${workspaceFolder}",
  "args": []
}
```
3. Debug Keycloak Operator

### Makefile command reference
#### Operator Setup Management
| *Command*                      | *Description*                                                                                          |
| ------------------------------ | ------------------------------------------------------------------------------------------------------ |
| `make cluster/prepare`         | Creates the `keycloak` namespace, applies all CRDs to the cluster and sets up the RBAC files           |
| `make cluster/clean`           | Deletes the `keycloak` namespace, all `keycloak.org` CRDs and all RBAC files named `keycloak-operator` |
| `make cluster/create/examples` | Applies the example Keycloak and KeycloakRealm CRs                                                     |

#### Tests
| *Command*                    | *Description*                                               |
| ---------------------------- | ----------------------------------------------------------- |
| `make test/unit`             | Runs unit tests                                             |
| `make test/e2e`              | Runs e2e tests with operator ran locally                    |
| `make test/e2e-latest-image` | Runs e2e tests with latest available operator image running in the cluster |
| `make test/e2e-local-image`  | Runs e2e tests with local operator image running in the cluster |
| `make test/coverage/prepare` | Prepares coverage report from unit and e2e test results     |
| `make test/coverage`         | Generates coverage report                                   |

##### Running tests without cluster admin permissions
It's possible to deploy CRDs, roles, role bindings, etc. separately from running the tests:
1. Run `make cluster/prepare` as a cluster admin.
2. Run `make test/ibm-validation` as a user. The user needs the following permissions to run te tests:
```
apiGroups: ["", "apps", "keycloak.org"]
resources: ["persistentvolumeclaims", "deployments", "statefulsets", "keycloaks", "keycloakrealms", "keycloakusers", "keycloakclients", "keycloakbackups"]
verbs: ["*"]
```
Please bear in mind this is intended to be used for internal purposes as there's no guarantee it'll work without any issues.

#### Local Development
| *Command*                 | *Description*                                                                    |
| ------------------------- | -------------------------------------------------------------------------------- |
| `make setup`              | Runs `setup/mod` `setup/githooks` `code/gen`                                     |
| `make setup/githooks`     | Copys githooks from `./githooks` to `.git/hooks`                                 |
| `make setup/mod`          | Resets the main module's vendor directory to include all packages                |
| `make setup/operator-sdk` | Installs the operator-sdk                                                        |
| `make code/run`           | Runs the operator locally for development purposes                               |
| `make code/compile`       | Builds the operator                                                              |
| `make code/gen`           | Generates/Updates the operator files based on the CR status and spec definitions |
| `make code/check`         | Checks for linting errors in the code                                            |
| `make code/fix`           | Formats code using [gofmt](https://golang.org/cmd/gofmt/)                        |
| `make code/lint`          | Checks for linting errors in the code                                            |

#### Application Monitoring

NOTE: This functionality works only in OpenShift environment.

| *Command*                         | *Description*                                           |
| --------------------------------- | ------------------------------------------------------- |
| `make cluster/prepare/monitoring` | Installs and configures Application Monitoring Operator |

* [Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0)
