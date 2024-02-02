# Spin k8s plugin

A [Spin plugin](https://github.com/fermyon/spin-plugins) for interacting with Kubernetes.

## Install

The latest stable release of the plugin can be installed like so:

```sh
spin plugins update
spin plugins install k8s
```

The canary release of the plugin represents the most recent commits on `main` and may not be stable, with some
features still in progress.

To install the canary release, use the `--url` parameter.

```sh
spin plugins install --url https://github.com/spinkube/spin-plugin-k8s/releases/download/canary/k8s.json
```

## Usage

Below is an example of using the plugin to deploy a Spin application to Kubernetes.

To run the example, you will need to install the wasm32-wasi target for Rust.

```sh
rustup target add wasm32-wasi
```

Run the spin new command to create a Spin application from a template.

```sh
spin new --accept-defaults -t http-rust hello-rust
```

Running `spin new` created a `hello-rust` directory with all the necessary files for your application. Change to the
`hello-rust` directory and build the application with `spin build`.

```sh
cd hello-rust
spin build
```

Publish your application to a container registry:

```sh
docker login
spin registry push bacongobbler/hello-rust
```

Deploy it to your Kubernetes cluster with `spin k8s deploy`.

```sh
spin k8s deploy --from bacongobbler/hello-rust
```

Connect to your app with `spin k8s connect`.

```sh
spin k8s connect hello-rust
```

List apps currently running in your cluster with `spin k8s list`.

```sh
spin k8s list
```

Delete an app from the cluster with `spin k8s delete`.

```sh
spin k8s delete hello-rust
```

## Compiling from source

As an alternative to the plugin manager, you can download and manually install the plugin. Manual installation is
commonly used to test in-flight changes. For a user, it's better to install the plugin using Spin's plugin manager.

Install the `pluginify` Spin plugin.

```sh
spin plugins update
spin plugins install pluginify --yes
```

Compile the plugin from source.

```sh
mkdir bin
go build -ldflags "-X github.com/spinkube/spin-plugin-k8s/pkg/cmd.Version=git-$(git rev-parse --short HEAD)" -o bin ./...
```

Install the plugin.

```sh
spin pluginify --install
```

## Scaffold the SpinApp

The `spin k8s scaffold` command produces the Kubernetes manifest for a SpinApp Kubernetes custom resource. This can be used in case you want to generate and inspect the SpinApp before deploying to a cluster. The `-f` flag is used to pass in a reference to the Spin application in a registry and is a required flag.

Example usage:

```console
$ spin k8s scaffold -f ghcr.io/deislabs/containerd-wasm-shims/examples/spin-rust-hello:v0.10.0

apiVersion: core.spinoperator.dev/v1
kind: SpinApp
metadata:
  name: spin-rust-hello
spec:
  image: "ghcr.io/deislabs/containerd-wasm-shims/examples/spin-rust-hello:v0.10.0"
  replicas: 2
```

Use `-o` to save the manifest to a file.

```console
$ spin k8s scaffold -f ghcr.io/deislabs/containerd-wasm-shims/examples/spin-rust-hello:v0.10.0 -o spinapp.yaml

SpinApp manifest saved to spinapp.yaml
```
