# Spin k8s plugin

A [Spin plugin](https://github.com/fermyon/spin-plugins) for interacting with Kubernetes.

## Install

Install the stable release:

```sh
spin plugins update
spin plugins install k8s
```

### Canary release

For the canary release:

```sh
spin plugins install --url https://github.com/spinkube/spin-plugin-k8s/releases/download/canary/k8s.json
```

The canary release may not be stable, with some features still in progress.

### Compiling from source

As an alternative to the plugin manager, you can download and manually install the plugin. Manual installation is
commonly used to test in-flight changes. For a user, it's better to install the plugin using Spin's plugin manager.

Ensure the `pluginify` plugin is installed:

```sh
spin plugins update
spin plugins install pluginify --yes
```

Compile and install the plugin:

```sh
make
make install
```

## Prerequisites

Ensure spin-operator is installed in your Kubernetes cluster. See the [spin-operator Quickstart
Guide](https://github.com/spinkube/spin-operator/blob/main/documentation/content/quickstart.md).

## Usage

Install the wasm32-wasi target for Rust:

```sh
rustup target add wasm32-wasi
```

Create a Spin application:

```sh
spin new --accept-defaults -t http-rust hello-rust
cd hello-rust
```

Compile the application:

```sh
spin build
```

Publish your application:

```sh
docker login
spin registry push bacongobbler/hello-rust:latest
```

Deploy to Kubernetes:

```sh
spin k8s scaffold --from bacongobbler/hello-rust:latest | kubectl create -f -
```

View your application:

```sh
kubectl get spinapps
```

`spin k8s scaffold` deploys two replicas by default. You can change this with the `--replicas` flag:

```sh
spin k8s scaffold --from bacongobbler/hello-rust:latest --replicas 3 | kubectl apply -f -
```

Delete the app:

```sh
kubectl delete spinapp hello-rust
```

### Working with images from private registries

Support for pulling images from private registries can be enabled by using `--image-pull-secret <secret-name>` flag, where `<secret-name>` is a secret of type [`docker-registry`](https://kubernetes.io/docs/concepts/configuration/secret/#docker-config-secrets) in same namespace as your SpinApp.

To enable multiple private registries, you can provide the flag `--image-pull-secret` multiple times with secret for each registry that you wish to use. 

Create a secret with credentials for private registry

```sh
$) kubectl create secret docker-registry registry-credentials \
  --docker-server=ghcr.io \
  --docker-username=bacongobbler \
  --docker-password=github-token

secret/registry-credentials created
```

Verify that the secret is created

```sh
$) kubectl get secret registry-credentials -o yaml

apiVersion: v1
data:
  .dockerconfigjson: eyJhdXRocyI6eyJnaGNyLmlvIjp7InVzZXJuYW1lIjoiYmFjb25nb2JibGVyIiwicGFzc3dvcmQiOiJnaXRodWItdG9rZW4iLCJhdXRoIjoiWW1GamIyNW5iMkppYkdWeU9tZHBkR2gxWWkxMGIydGxiZz09In19fQ==
kind: Secret
metadata:
  creationTimestamp: "2024-02-27T02:18:53Z"
  name: registry-credentials
  namespace: default
  resourceVersion: "162287"
  uid: 2e12ddd1-919d-44b5-b6cc-c3cd5c09fcec
type: kubernetes.io/dockerconfigjson
```

Use the secret when scaffolding the SpinApp

```sh
$) spin k8s scaffold --from bacongobbler/hello-rust:latest --image-pull-secret registry-credentials

apiVersion: core.spinoperator.dev/v1
kind: SpinApp
metadata:
  name: hello-rust
spec:
  image: "bacongobbler/hello-rust:latest"
  replicas: 2
  executor: containerd-shim-spin
  imagePullSecrets:
    - name: registry-credentials
```
