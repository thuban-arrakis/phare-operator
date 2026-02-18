# phare-controller
<img alt="logo" src="./assets/logo.png" width="420" height="420" />

Kubernetes operator for reconciling `Phare` custom resources into workloads and supporting infrastructure.

## Description
`phare-controller` watches `Phare` resources and creates/updates the runtime objects needed to run an app.
It manages:
- `Deployment` or `StatefulSet` (based on `spec.microservice.kind`)
- optional `Service`
- optional generated `ConfigMap` from `spec.toolchain.config`
- optional `HTTPRoute`
- optional GKE policy resources (`GCPBackendPolicy`, `HealthCheckPolicy`)

The reconcile loop is idempotent and updates `status.phase`/`status.message` when reconciliation succeeds.

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Local development
1. Run fast checks (format, vet, tests):

```sh
make check
```

2. Run tests directly:

```sh
make unit-test
```

3. Run the controller locally:

```sh
make run
```

### Running on a cluster
1. Install CustomResourceDefinitions and sample resources:

```sh
make install
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<some-registry>/operator:tag
```

3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/operator:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
Undeploy the controller from the cluster:

```sh
make undeploy
```

## Contributing
Keep reconciliation behavior deterministic and idempotent.

Before opening a PR, run:

```sh
make check
```

If you change API types or markers, also run:

```sh
make manifests generate
```

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Modifying the API definitions
If you are editing API definitions or kubebuilder markers, regenerate manifests and deep-copy code:

```sh
make manifests generate
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
