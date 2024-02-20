# Spin k8s plugin

A [Spin plugin](https://github.com/fermyon/spin-plugins) for interacting with Kubernetes.

## Install

The latest stable release of the plugin can be installed like so:

```sh
spin plugins update
spin plugins install k8s
```

The canary release of the plugin represents the most recent commits on `main` and may not be stable, with some features
still in progress.

To install the canary release, use the `--url` parameter.

```sh
spin plugins install --url https://github.com/spinkube/spin-plugin-k8s/releases/download/canary/k8s.json
```

## Prerequisites

Make sure you have spin-operator installed in your Kubernetes cluster. Follow the [spin-operator Quickstart
Guide](https://github.com/spinkube/spin-operator/blob/main/documentation/content/quickstart.md) for a step-by-step
tutorial to set up a development environment.

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
spin registry push bacongobbler/hello-rust:latest
```

Deploy it to your Kubernetes cluster with `spin k8s scaffold` and `kubectl apply`:

```sh
spin k8s scaffold --from bacongobbler/hello-rust:latest | kubectl apply -f -
```

View your application with `kubectl get spinapps`.

```sh
$ kubectl get spinapps
NAME         READY REPLICAS   EXECUTOR
hello-rust   2     2          containerd-shim-spin
```

You'll notice two replicas are running. `spin k8s scaffold` deploys two replicas by default. You can change this with
the `--replicas` flag:

```sh
spin k8s scaffold --from bacongobbler/hello-rust:latest --replicas 3 | kubectl apply -f -
```

Delete an app from the cluster with `kubectl delete`.

```sh
kubectl delete spinapp hello-rust
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
make
```

Install the plugin.

```sh
make install
```
