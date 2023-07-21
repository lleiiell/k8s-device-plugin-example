# k8s-device-plugin-example

This repo is a minimal example for Kubernetes device plugin.

## Deploy

```bash
kubectl apply -f deploy/device-plugin-example.yaml
```

Then describe worker nodes.

```bash
Capacity:
  example.com/device:        2
Allocatable:
  example.com/device:        2
```

## Test

```bash
kubectl apply -f deploy/test-pod.yaml
kubectl apply -f deploy/test-pod2.yaml
kubectl apply -f deploy/test-pod3.yaml
```

Then watch pods.

## issues

### unknown revision v0.0.0

Add the `replace` for the unknown modules to `go.mod` as follows.

```bash
replace k8s.io/component-helpers => k8s.io/component-helpers latest
replace k8s.io/api => k8s.io/api latest
replace k8s.io/apimachinery => k8s.io/apimachinery latest
```

Then run `go mod tidy`

## refs

- https://github.com/NVIDIA/k8s-device-plugin
- https://github.com/AliyunContainerService/gpushare-device-plugin
- https://github.com/kubernetes/kubernetes/tree/v1.27.4/test/images/sample-device-plugin