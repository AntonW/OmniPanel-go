# Chapter 22: Kubernetes Deployment with kustomize

## What This Chapter Covers

OmniPanel-go's `serve` mode can be deployed to a Kubernetes cluster using the provided kustomize manifests in `k8s/`. This enables running the relay server as a managed Kubernetes workload with persistent storage, health probes, and ingress routing.

The kustomize setup:
- Deploys the container with `serve` command
- Mounts a PersistentVolumeClaim for user data (panels, blocks, themes)
- Includes liveness and readiness probes
- Supports both Gateway API (HTTPRoute) and traditional Ingress
- Provides a production overlay for custom hostnames and image tags

> **Note:** The Kubernetes deployment uses the same container image built with ko. See [Chapter 21](21-container-build.md) for building the container image.

## Directory Structure

```
k8s/
├── base/
│   ├── deployment.yaml    # Deployment with serve command, PVC, probes
│   ├── pvc.yaml           # PersistentVolumeClaim for user data
│   ├── service.yaml       # ClusterIP service
│   ├── httproute.yaml     # Gateway API HTTPRoute (default)
│   ├── ingress.yaml       # Nginx Ingress (alternative)
│   └── kustomization.yaml # Base kustomization
└── overlays/
    └── production/
        └── kustomization.yaml # Production overlay
```

## Base Manifests

### Deployment

```yaml
# k8s/base/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: omnipanel-go
spec:
  replicas: 1
  strategy:
    type: Recreate
  template:
    spec:
      containers:
        - name: omnipanel-go
          image: omnipanel-go:latest
          args:
            - serve
          ports:
            - name: http
              containerPort: 3000
          workingDir: /var/run/ko
          volumeMounts:
            - name: user-data
              mountPath: /var/run/ko/user
```

> **Concept: Recreate strategy**
> `strategy.type: Recreate` terminates the old pod before creating a new one. This is necessary because the PVC is `ReadWriteOnce` — only one pod can mount it at a time. The default `RollingUpdate` strategy would try to start a new pod while the old one is still running, causing a volume mount conflict.

Key deployment settings:

| Field | Value | Purpose |
|-------|-------|---------|
| `args: [serve]` | `serve` | Runs the relay server mode |
| `workingDir` | `/var/run/ko` | Required for ko container to resolve `static/` and `config.json` |
| `volumeMounts` | `/var/run/ko/user` | Persistent user data directory |
| `livenessProbe` | HTTP GET `/` | Restarts container if server becomes unresponsive |
| `readinessProbe` | HTTP GET `/` | Removes pod from service until server is ready |

### PersistentVolumeClaim

```yaml
# k8s/base/pvc.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: omnipanel-go-user-data
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
```

The PVC stores all user-created content: panels, blocks, themes, assets, and speech commands. On first run with an empty volume, the `starter` package copies default files into it (see [Chapter 21](21-container-build.md)).

> **Key Pattern: Persistent user data**
> The PVC ensures user data survives pod restarts, image upgrades, and node failures. The `ReadWriteOnce` access mode means only one pod can write to the volume — this matches the 1:1 relay server architecture where only one instance should run at a time.

### Service

```yaml
# k8s/base/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: omnipanel-go
spec:
  type: ClusterIP
  ports:
    - name: http
      port: 80
      targetPort: http
  selector:
    app: omnipanel-go
```

The ClusterIP service exposes the relay server internally on port 80, forwarding to the container's port 3000. External access is handled by either HTTPRoute or Ingress.

### HTTPRoute (Gateway API)

```yaml
# k8s/base/httproute.yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: omnipanel-go
spec:
  parentRefs:
    - name: default
      namespace: gateway-system
  hostnames:
    - omnipanel.example.com
  rules:
    - matches:
        - path:
            type: PathPrefix
            value: /
      backendRefs:
        - name: omnipanel-go
          port: 80
```

The HTTPRoute uses the Kubernetes Gateway API (`gateway.networking.k8s.io/v1`), the modern replacement for Ingress. It requires a Gateway controller (e.g., Envoy Gateway, Cilium, or cloud provider gateways) to be installed in your cluster.

> **Concept: Gateway API vs Ingress**
> The Gateway API is the successor to Ingress in Kubernetes. It provides better support for multi-tenant routing, role-based access, and protocol-specific features. HTTPRoute is the most common resource type. If your cluster doesn't have a Gateway controller, use the Ingress manifest instead.

### Ingress (Alternative)

```yaml
# k8s/base/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: omnipanel-go
  annotations:
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
spec:
  ingressClassName: nginx
  rules:
    - host: omnipanel.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: omnipanel-go
                port:
                  name: http
```

The Ingress uses the nginx ingress class with extended WebSocket timeouts (3600 seconds). This is important because the relay server maintains long-lived WebSocket connections for browser clients and the host agent. Without extended timeouts, the ingress controller would terminate idle connections prematurely.

> **Key Pattern: WebSocket timeout configuration**
> WebSocket connections are long-lived and may appear idle during periods of no activity. The default nginx timeout (60 seconds) would close these connections. Setting `proxy-read-timeout` and `proxy-send-timeout` to 3600 (1 hour) keeps WebSocket connections alive. This applies to both the browser WebSocket at `/ws` and the host agent WebSocket at `/ws?type=host`.

## Kustomization

### Base

```yaml
# k8s/base/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - deployment.yaml
  - pvc.yaml
  - service.yaml

  # Choose ONE of the following ingress methods:
  - httproute.yaml
  # - ingress.yaml

images:
  - name: omnipanel-go
    newTag: latest
```

To switch from HTTPRoute to Ingress, comment out `httproute.yaml` and uncomment `ingress.yaml`.

### Production Overlay

```yaml
# k8s/overlays/production/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base

images:
  - name: omnipanel-go
    newTag: v1.0.0

patches:
  - target:
      kind: HTTPRoute
      name: omnipanel-go
    patch: |
      - op: replace
        path: /spec/hostnames/0
        value: omnipanel.mydomain.com
```

The overlay changes the image tag and hostname without modifying the base manifests.

## Deploying

### Prerequisites

- Kubernetes cluster (v1.25+)
- `kubectl` configured
- Container image pushed to a registry (see [Chapter 21](21-container-build.md))
- Either a Gateway controller (for HTTPRoute) or nginx ingress controller (for Ingress)

### Quick Deploy

```bash
# Edit the hostname in k8s/base/httproute.yaml or k8s/base/ingress.yaml
# Edit the image tag in k8s/base/kustomization.yaml

kubectl apply -k k8s/base/
```

### Verify Deployment

```bash
# Check pod status
kubectl get pods -l app=omnipanel-go

# Check logs
kubectl logs -l app=omnipanel-go

# Check service
kubectl get svc omnipanel-go
```

### First Run

When the pod starts with a fresh PVC, the `starter` package copies default panels, blocks, themes, and assets into `/var/run/ko/user`. Subsequent restarts preserve existing user data.

## Customization

### Change Image Tag

```bash
kubectl kustomize k8s/base/ | sed 's/image: omnipanel-go:latest/image: your.registry.io/omnipanel-go:v1.0.0/' | kubectl apply -f -
```

Or edit `k8s/base/kustomization.yaml`:

```yaml
images:
  - name: omnipanel-go
    newTag: v1.0.0
```

### Change Hostname

Edit `k8s/base/httproute.yaml` (HTTPRoute) or `k8s/base/ingress.yaml` (Ingress):

```yaml
hostnames:
  - omnipanel.mydomain.com
```

### Change PVC Size

Edit `k8s/base/pvc.yaml`:

```yaml
resources:
  requests:
    storage: 5Gi
```

### Add TLS

For HTTPRoute, add a TLS section to the Gateway (not the HTTPRoute itself). For Ingress:

```yaml
spec:
  tls:
    - hosts:
        - omnipanel.mydomain.com
      secretName: omnipanel-tls
```

## Key Takeaways

- kustomize manifests in `k8s/` provide a complete Kubernetes deployment for serve mode
- `Recreate` strategy is required because the PVC is `ReadWriteOnce`
- `workingDir: /var/run/ko` is required for ko container file resolution
- Starter files auto-populate the PVC on first run
- HTTPRoute (Gateway API) and Ingress are both supported — choose one
- WebSocket timeout annotations prevent premature connection termination
- Production overlay demonstrates how to customize hostname and image tag
- Liveness and readiness probes ensure healthy pod lifecycle management

[← Back: Chapter 21](21-container-build.md)
