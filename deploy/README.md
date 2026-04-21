# New-API Deployment Guide

This project supports two production deployment methods: **Docker Compose** (single-host) and **Kubernetes** (cluster). Both provide full-featured deployments with PostgreSQL, Redis, health checks, and Prometheus monitoring.

---

## Quick Comparison

| Feature | Docker Compose | Kubernetes (Helm) | Kubernetes (Standalone) |
|---------|---------------|-------------------|------------------------|
| Complexity | Low | Medium | Medium |
| Multi-node scaling | No (use Swarm) | Yes (HPA) | Yes (HPA) |
| Auto-scaling | No | Yes (HPA 2-20 pods) | Yes (2-20 pods) |
| Self-healing | Restart only | Pod recreation + PDB | Pod recreation + PDB |
| Database included | Yes (PostgreSQL/MySQL) | Via Bitnami subchart | Yes (in-cluster) |
| Redis included | Yes | Via Bitnami subchart | Yes (in-cluster) |
| Ingress/TLS | Manual | Built-in | Manual |
| Monitoring | Port exposed | ServiceMonitor | ServiceMonitor |
| Best for | Dev / single server / PoC | Production cluster | Production without Helm |

---

## Method 1: Docker Compose

### Prerequisites

- Docker >= 24.0
- Docker Compose v2 (`docker compose` plugin)
- 4GB+ RAM available on host
- Ports 3000, 9090 (and optionally 5432/3306 for DB access)

### Step-by-step

```bash
# 1. Clone repository
git clone https://github.com/QuantumNous/new-api.git
cd new-api

# 2. Create environment file from template
cp deploy/.env.example .env
vim .env   # ⚠️ EDIT: Change ALL passwords!

# 3. Start with PostgreSQL (default)
docker compose --profile postgres up -d

# Or start with MySQL instead
docker compose --profile mysql up -d

# 4. Wait for services to become healthy
docker compose ps

# 5. Access the application
open http://localhost:3000
```

### Environment Variables (.env)

All configuration is managed via `.env` file. Key variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_PASSWORD` | Database password (PostgreSQL or MySQL) | `123456` — MUST change |
| `REDIS_PASSWORD` | Redis auth password | `123456` — MUST change |
| `SESSION_SECRET` | Cluster session secret (generate with `openssl rand -hex 32`) | Auto-generated per start |
| `NEW_API_PORT` | Application HTTP port | 3000 |
| `METRICS_PORT` | Prometheus metrics port | 9090 |
| `STREAMING_TIMEOUT` | Streaming response timeout (seconds) | 300 |
| `METRICS_ENABLED` | Enable Prometheus metrics endpoint | true |
| `CANARY_ENABLED` | Enable canary/gray release routing | false |

Full variable list: see `deploy/.env.example`.

### Common Operations

```bash
# View logs
docker compose logs -f new-api

# Scale application (requires load balancer in front)
docker compose up -d --scale new-api=3

# Stop everything
docker compose down

# Stop and remove volumes (data will be lost!)
docker compose down -v

# Rebuild with local image
docker build -t new-api-local:latest .
# Then edit docker-compose.yml to use image: new-api-local:latest
docker compose up -d
```

### Switching Databases

The compose file uses Docker Compose profiles to manage database selection:

```bash
# PostgreSQL (default)
docker compose --profile postgres up -d

# MySQL
docker compose --profile mysql up -d
```

When using MySQL, update `.env`:
```
SQL_DSN=root:${DB_PASSWORD}@tcp(mysql:3306)/${DB_NAME}
```

---

## Method 2: Kubernetes (Helm Chart)

### Prerequisites

- Kubernetes cluster v1.24+
- kubectl configured with admin access
- Helm 3.12+
- StorageClass provisioned for PVCs (or pre-create PVs)

### Install from Repository

```bash
# Add Helm repository (when published)
helm repo add new-api https://charts.example.com/new-api
helm repo update

# Create namespace
kubectl create namespace new-api

# Install with default values (includes Bitnami PostgreSQL + Redis)
helm install new-api new-api/new-api \
  -n new-api \
  --set sessionSecret=$(openssl rand -hex 32) \
  --set database.password="your-db-password" \
  --set redis.auth.password="your-redis-password"
```

### Install from Local Source

```bash
cd deploy/k8s/helm

# Install with defaults
helm install new-api . \
  -n new-api \
  --set sessionSecret=$(openssl rand -hex 32) \
  --set database.password="your-db-password" \
  --set redis.auth.password="your-redis-password"

# Install with custom values file
cat > my-values.yaml << 'EOF'
replicaCount: 3
sessionSecret: "your-generated-secret"
database.password: "your-db-password"
redis.auth.password: "your-redis-password"
ingress.enabled: true
ingress.className: nginx
ingress.hosts:
  - host: api.yourdomain.com
    paths:
      - path: /
        pathType: Prefix
resources:
  limits:
    cpu: "2"
    memory: 1Gi
  requests:
    cpu: 200m
    memory: 256Mi
EOF

helm install new-api . -n new-api -f my-values.yaml
```

### Using External Database / Redis

When your cluster already has managed databases:

```bash
helm install new-api . -n newapi \
  --set sessionSecret=$(openssl rand -hex 32) \
  --set postgresql.enabled=false \
  --set redis.enabled=false \
  --set redis.external.enabled=true \
  --set redis.external.host="your-redis-host" \
  --set redis.external.port=6379 \
  --set redis.external.password="your-redis-pwd" \
  --set database.dsn="postgresql://user:pwd@external-db-host:5432/newapi?sslmode=require"
```

### Helm Values Reference

See `deploy/k8s/helm/values.yaml` for complete list. Key groups:

| Group | Description |
|-------|-------------|
| `image.*` | Container image, tag, pull policy |
| `replicaCount` | Initial pod count (default: 2) |
| `autoscaling.*` | HPA settings (min/max replicas, CPU/memory targets) |
| `database.*` | Database type, DSN, credentials |
| `redis.*` | Redis config; set `redis.external.enabled=true` for external |
| `service.*` / `ingress.*` | Network exposure config |
| `monitoring.prometheus` | Metrics port/path toggle |
| `monitoring.openTelemetry` | OTel tracing to collector |
| `resources` | CPU/memory requests and limits |
| `rateLimit.*` | API/web rate limiting thresholds |

### Helm Operations

```bash
# Upgrade after values change
helm upgrade new-api . -n new-api -f my-values.yaml

# Rollback to previous version
helm rollback new-api 1 -n new-api

# Uninstall completely
helm uninstall new-api -n new-api

# Debug: render templates without installing
helm template new-api . -n new-api -f my-values.yaml > rendered.yaml
```

---

## Method 3: Kubernetes (Standalone Manifests)

For environments without Helm, use the self-contained YAML manifests that include PostgreSQL, Redis, and the application.

### Prerequisites

- Kubernetes cluster v1.24+ (for HPA v2, PDB v1)
- kubectl configured
- StorageClass for PVC dynamic provisioning

### Deploy

```bash
# 1. Edit secrets first!
# Open deploy/k8s/standalone/k8s-deployment.yaml and change:
#   - session-secret under new-api-secrets
#   - db-password
#   - redis-password

# 2. Deploy all resources
kubectl apply -f deploy/k8s/standalone/k8s-deployment.yaml

# 3. Watch pods become ready
kubectl get pods -n new-api -w

# 4. Port-forward for local access
kubectl port-forward svc/new-api -n new-api 3000:3000
# Open http://localhost:3000
```

### What's Included

The standalone manifest deploys these resources in order:

| Resource | Purpose |
|----------|---------|
| Namespace (`new-api`) | Isolated deployment namespace |
| Secret (`new-api-secrets`) | Credentials for app, DB, Redis |
| Deployment (`postgresql`) | PostgreSQL 16 with PVC persistence |
| Deployment (`redis`) | Redis 7 with AOF persistence |
| ConfigMap (`new-api-config`) | Application environment variables |
| Deployment (`new-api`) | Application (3 replicas, anti-affinity) |
| ServiceAccount (`new-api`) | Identity for pod service account |
| Service (`new-api`) | ClusterIP exposing ports 3000 + 9090 |
| HPA (`new-api-hpa`) | Auto-scale 2-20 pods by CPU/Memory |
| PDB (`new-api-pdb`) | Guarantee min 1 available during disruption |
| NetworkPolicy (`new-api-network-policy`) | Restrict ingress/egress traffic |
| ServiceMonitor (`new-api`) | Prometheus metrics scraping |

### Cleanup

```bash
kubectl delete -f deploy/k8s/standalone/k8s-deployment.yaml
# This deletes namespace, PVCs, and all resources (DATA WILL BE LOST)
```

---

## Production Checklist

Before deploying to production, ensure you have completed:

### Security

- [ ] Changed all default passwords in `.env`, `my-values.yaml`, or Secrets manifest
- [ ] Generated `SESSION_SECRET` with `openssl rand -hex 32`
- [ ] Enabled TLS (Ingress annotation for K8s, reverse proxy for Docker)
- [ ] Set resource limits to prevent OOM kills
- [ ] Enabled NetworkPolicy (K8s) or non-root container user
- [ ] Restricted database ports to internal network only

### Reliability

- [ ] Configured PersistentVolumes for data durability
- [ ] Enabled HPA with appropriate replica range
- [ ] Set up PodDisruptionBudget for maintenance safety
- [ ] Configured liveness + readiness probes
- [ ] Set up log aggregation (ELK/Loki/Docker driver)

### Monitoring

- [ ] Prometheus scraping `/metrics` endpoint
- [ ] Grafana dashboard for key metrics (latency, throughput, error rate)
- [ ] Alerting rules for high error rate, circuit breaker open, etc.
- [ ] Optional: OpenTelemetry tracing enabled for distributed tracing

### Performance

- [ ] Tuned `RELAY_MAX_IDLE_CONNS` for upstream connection pooling
- [ ] Set appropriate `STREAMING_TIMEOUT` for long-running requests
- [ ] Enabled batch updates for billing efficiency
- [ ] Considered CDN for static frontend assets

---

## Troubleshooting

### Docker Compose

| Issue | Solution |
|-------|----------|
| Container restart loop | `docker compose logs new-api` — check DB connection string |
| Database connection refused | Ensure DB profile is active: `--profile postgres` or `--profile mysql` |
| Permission denied on `./data` | `chmod 777 ./data` or adjust volume mount owner |
| Port already in use | Change `NEW_API_PORT` in `.env` |

### Kubernetes

| Issue | Solution |
|-------|----------|
| Pod stuck in `Pending` | Check PVC binding: `kubectl describe pvc -n new-api` |
| Pod `CrashLoopBackOff` | Check init containers: DB/Redis might not be ready yet |
| 502 from Ingress | Verify Service endpoints: `kubectl get endpoints new-api -n new-api` |
| Metrics not scraped | Confirm ServiceMonitor selector matches pod labels |
| HPA not scaling | Check current CPU/Memory utilization: `kubectl top pods -n new-api` |
