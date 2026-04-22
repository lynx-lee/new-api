# AI-Bridge Kubernetes Deployment Guide

## Target Architecture

```
                    ┌──────────────┐   DNS/LB (CloudFlare/ALB)
                    │              │
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐   Ingress (K8s Nginx Ingress)
                    │   Controller │
                    └──────┬───────┘
                           │
          ┌────────┬───────┼───────┬────────┐
          ▼        ▼       ▼       ▼        ▼
     Replica 1  Replica 2 ... Replica N    ← ai-bridge Pods
          │        │       │       │        │
          └────────┴───────┴───────┴────────┘
                           │
         ┌─────────────────┼─────────────────┐
         ▼                 ▼                 ▼
   Redis Cluster    PostgreSQL Cluster   Prometheus
```

## Quick Start

### Prerequisites

1. **Kubernetes Cluster** (v1.28+) with [Nginx Ingress Controller](https://kubernetes.github.io/ingress-nginx/deploy/) installed
2. **External PostgreSQL Cluster** (or enable bitnami postgresql sub-chart)
3. **External Redis** (standalone or cluster) — or enable bitnami redis sub-chart
4. **[Prometheus Operator](https://prometheus-operator.dev/docs/prologue/introduction/)** (optional, for ServiceMonitor)

### Option A: Helm Chart (Recommended)

```bash
# 1. Generate secrets
export SESSION_SECRET=$(openssl rand -hex 32)
export DB_PASSWORD="your-db-password"
export REDIS_PASSWORD="your-redis-password"

# 2. Install (development — uses embedded redis/postgresql via bitnami charts)
helm install ai-bridge ./deploy/k8s/helm \
  --namespace ai-bridge \
  --create-namespace \
  --set sessionSecret=$SESSION_SECRET \
  --set database.password=$DB_PASSWORD \
  --set redis.auth.password=$REDIS_PASSWORD \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=api.yourdomain.com

# 3. Install (production — external dependencies)
helm install ai-bridge ./deploy/k8s/helm \
  -f deploy/k8s/helm/values-prod.yaml \
  --namespace ai-bridge \
  --create-namespace \
  --set sessionSecret=$SESSION_SECRET \
  --set "database.external.host=pg-cluster.yourdomain.com" \
  --set "database.external.password=$DB_PASSWORD" \
  --set "redis.external.host=redis-cluster.yourdomain.com" \
  --set "redis.external.password=$REDIS_PASSWORD" \
  --set image.tag=v1.0.0
```

### Option B: Standalone Manifests (Kustomize)

```bash
# 1. Prepare environment file
cp deploy/k8s/standalone/env.example deploy/k8s/standalone/.env.k8s
# Edit .env.k8s with your actual values!

# 2. Create namespace and secrets
kubectl create namespace ai-bridge
kubectl -n ai-bridge create secret generic ai-bridge-secret \
  --from-literal=session-secret=$(openssl rand -hex 32) \
  --from-literal=db-password=YOUR_DB_PASSWORD \
  --from-literal=redis-password=YOUR_REDIS_PASSWORD

# 3. Apply
kubectl apply -k deploy/k8s/standalone/
```

### Option C: Plain YAML (No Kustomize)

```bash
kubectl apply -f deploy/k8s/standalone/k8s-deployment.yaml
# Then manually edit secrets/configmap values in the cluster:
kubectl -n ai-bridge edit secret ai-bridge-secret
kubectl -n ai-bridge edit configmap ai-bridge-config
```

---

## Configuration Reference

### Values (`values.yaml`)

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Initial pod replicas | `2` |
| `image.repository` | Container image | `calciumion/ai-bridge` |
| `image.tag` | Image tag | `latest` |
| `ingress.enabled` | Enable Nginx Ingress | `false` |
| `ingress.className` | Ingress class | `"nginx"` |
| `autoscaling.enabled` | Enable HPA | `true` |
| `autoscaling.minReplicas` | Min pods | `2` |
| `autoscaling.maxReplicas` | Max pods | `20` |
| `database.type` | `postgresql` / `mysql` | `postgresql` |
| `database.external.enabled` | Use external PG cluster | `false` |
| `redis.architecture` | `standalone` / `cluster` | `standalone` |
| `redis.external.enabled` | Use external Redis | `false` |
| `monitoring.prometheus.enabled` | Enable metrics endpoint | `true` |

### Production Checklist

- [ ] Set `sessionSecret` (required for multi-pod session consistency)
- [ ] Pin `image.tag` to a specific version (never use `latest` in prod)
- [ ] Enable TLS on Ingress (`cert-manager` recommended)
- [ ] Configure resource limits (`resources.limits`)
- [ ] Enable `podAntiAffinity` for HA spread across nodes
- [ ] Use external PostgreSQL Cluster (not embedded bitnami chart)
- [ ] Use external Redis Cluster for production scale
- [ ] Set up Grafana dashboard (`deploy/k8s/grafana/dashboard.json`)
- [ ] Enable OpenTelemetry tracing to collector
- [ ] Review NetworkPolicy rules for your environment

---

## Monitoring

### Prometheus Metrics

Available at `:9090/metrics` when `monitoring.prometheus.enabled=true`.

Key metrics exposed by the application:
- HTTP request rate & latency histograms (per endpoint)
- Active connections / goroutines count
- Relay request duration (to upstream AI providers)
- Error rate by status code
- Token usage counters
- Cache hit/miss rates (Redis)

### Grafana Dashboard

Import `deploy/k8s/grafana/dashboard.json` into your Grafana instance.

Dashboard includes:
- Request rate over time (req/s)
- System overview table (pod status, CPU/MEM %)
- API latency P95/P99
- Top error endpoints
- Network traffic throughput

### ServiceMonitor

Auto-created when `monitoring.prometheus.enabled=true`. Ensure Prometheus Operator is installed with label selector matching `release: prometheus`.

---

## Scaling Guide

### Horizontal Scaling (HPA)

The HPA is enabled by default, scaling based on CPU (75%) and memory (80%) utilization:

```bash
# Check HPA status
kubectl get hpa -n ai-bridge

# Manually trigger scale-up test
kubectl scale deployment ai-bridge -n ai-bridge --replicas=10
```

### Vertical Scaling

Adjust resource requests/limits:

```bash
# Via Helm upgrade
helm upgrade ai-bridge ./deploy/k8s/helm \
  --set resources.limits.cpu=4 \
  --set resources.limits.memory=2Gi \
  --set resources.requests.cpu=500m \
  --set resources.requests.memory=512Mi
```

---

## Troubleshooting

| Issue | Diagnosis | Fix |
|-------|-----------|-----|
| Pod CrashLoopBackOff | `kubectl logs -n ai-bridge <pod>` | Check SQL_DSN / REDIS_CONN_STRING validity |
| 502 from Ingress | `kubectl get ep -n ai-bridge` | No ready endpoints — check readiness probe |
| Session lost between pods | Missing `SESSION_SECRET` | Must set consistent secret across all pods |
| High latency to upstream | `kubectl top pods -n ai-bridge` | CPU throttled → increase limits; network → check DNS |
| Redis connection refused | `kubectl exec -it <pod> -- nslookup redis-host` | External Redis not reachable from cluster network |
