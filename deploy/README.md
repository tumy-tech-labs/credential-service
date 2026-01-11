# Kubernetes Deployment

This directory contains Kubernetes manifests for deploying the credential service with independent scaling.

## Quick Start (Local with Kind)

```bash
# 1. Deploy to Kind cluster
./deploy/local-deploy.sh

# 2. Port forward services
kubectl port-forward -n credential-service svc/credential-issuer 8080:8080 &
kubectl port-forward -n credential-service svc/credential-verifier 8081:8081 &

# 3. Test the services
curl http://localhost:8080/healthz
curl http://localhost:8081/readyz
```

## Production Deployment

### Prerequisites

1. **Build and push Docker images:**
   ```bash
   # Build
   docker build -f build/issuer.Dockerfile -t ghcr.io/bradtumy/credential-issuer:v1.0.0 .
   docker build -f build/verifier.Dockerfile -t ghcr.io/bradtumy/credential-verifier:v1.0.0 .
   
   # Push to registry
   docker push ghcr.io/bradtumy/credential-issuer:v1.0.0
   docker push ghcr.io/bradtumy/credential-verifier:v1.0.0
   ```

2. **Update image tags in manifests:**
   ```bash
   # k8s/issuer.yaml
   image: ghcr.io/bradtumy/credential-issuer:v1.0.0
   
   # k8s/verifier.yaml
   image: ghcr.io/bradtumy/credential-verifier:v1.0.0
   ```

### Deploy to Kubernetes

```bash
# Apply all manifests
kubectl apply -k deploy/k8s/

# Or apply individually
kubectl apply -f deploy/k8s/namespace.yaml
kubectl apply -f deploy/k8s/postgres.yaml
kubectl apply -f deploy/k8s/issuer.yaml
kubectl apply -f deploy/k8s/verifier.yaml
kubectl apply -f deploy/k8s/ingress.yaml
```

## Independent Scaling

### Manual Scaling

Scale each service independently based on load:

```bash
# Scale issuer to 10 replicas (CPU-bound workload)
kubectl scale deployment credential-issuer -n credential-service --replicas=10

# Scale verifier to 20 replicas (I/O-bound workload)
kubectl scale deployment credential-verifier -n credential-service --replicas=20

# Check status
kubectl get pods -n credential-service -l app=credential-issuer
kubectl get pods -n credential-service -l app=credential-verifier
```

### Automatic Scaling (HPA)

Both services have HorizontalPodAutoscalers configured:

**Issuer HPA:**
- Min: 3 replicas
- Max: 20 replicas
- Triggers: CPU > 70%, Memory > 80%

**Verifier HPA:**
- Min: 5 replicas
- Max: 50 replicas
- Triggers: CPU > 70%, Memory > 80%

```bash
# View HPA status
kubectl get hpa -n credential-service

# Describe HPA for details
kubectl describe hpa credential-issuer-hpa -n credential-service
kubectl describe hpa credential-verifier-hpa -n credential-service
```

### Custom Metrics (Advanced)

For production, scale based on custom metrics:

```bash
# Scale issuer based on credential issuance rate
kubectl autoscale deployment credential-issuer \
  -n credential-service \
  --cpu-percent=70 \
  --min=5 \
  --max=30 \
  --name=issuer-custom-hpa

# Scale verifier based on gateway authorization requests
kubectl autoscale deployment credential-verifier \
  -n credential-service \
  --cpu-percent=70 \
  --min=10 \
  --max=50 \
  --name=verifier-custom-hpa
```

## Monitoring

### View Logs

```bash
# Issuer logs
kubectl logs -n credential-service -l app=credential-issuer --tail=100 -f

# Verifier logs
kubectl logs -n credential-service -l app=credential-verifier --tail=100 -f

# Specific pod
kubectl logs -n credential-service <pod-name> -f
```

### Metrics

Prometheus scrapes metrics from both services:

```bash
# Port forward to view metrics
kubectl port-forward -n credential-service svc/credential-issuer 8080:8080
curl http://localhost:8080/metrics

kubectl port-forward -n credential-service svc/credential-verifier 8081:8081
curl http://localhost:8081/metrics
```

### Health Checks

```bash
# Check pod health
kubectl get pods -n credential-service

# Describe pod for events
kubectl describe pod <pod-name> -n credential-service

# Check readiness/liveness probes
kubectl get pods -n credential-service -o json | jq '.items[] | {name: .metadata.name, ready: .status.conditions[] | select(.type=="Ready")}'
```

## Load Testing

Test scaling behavior with load:

```bash
# Install k6 or use existing load testing tool
brew install k6  # macOS

# Create load test script
cat > loadtest.js <<'EOF'
import http from 'k6/http';
import { check } from 'k6';

export let options = {
  stages: [
    { duration: '2m', target: 100 },  // Ramp up
    { duration: '5m', target: 100 },  // Stay at 100 users
    { duration: '2m', target: 200 },  // Spike
    { duration: '5m', target: 200 },  // Stay at 200
    { duration: '2m', target: 0 },    // Ramp down
  ],
};

export default function () {
  // Test issuer
  let issuerRes = http.post('http://issuer.example.com/v1/credentials/issue', JSON.stringify({
    subject_did: 'did:jwk:...',
    ttl_seconds: 600,
    claims: { role: 'tester', aud: 'api' }
  }), { headers: { 'Content-Type': 'application/json' } });
  
  check(issuerRes, { 'issuer status is 200': (r) => r.status === 200 });
  
  // Test verifier
  let verifierRes = http.post('http://verifier.example.com/v1/gateway/authorize', JSON.stringify({
    credential: issuerRes.json('credential'),
    expected_audience: 'api',
    resource: 'orders',
    action: 'read'
  }), { headers: { 'Content-Type': 'application/json' } });
  
  check(verifierRes, { 'verifier status is 200': (r) => r.status === 200 });
}
EOF

# Run load test
k6 run loadtest.js

# Watch HPA during test
watch kubectl get hpa -n credential-service
```

## Cleanup

```bash
# Delete everything in namespace
kubectl delete namespace credential-service

# Or delete Kind cluster
kind delete cluster --name credential-service
```

## Architecture Notes

### Why Separate Scaling?

1. **Different Load Patterns:**
   - Issuer: Bursty, CPU-intensive (crypto operations)
   - Verifier: Steady, I/O-intensive (DB lookups, gateway traffic)

2. **Cost Optimization:**
   - Scale issuer only during peak issuance hours
   - Scale verifier for 24/7 gateway authorization traffic

3. **Fault Isolation:**
   - Verifier outage doesn't affect new credential issuance
   - Issuer outage doesn't affect existing credential verification

### Scaling Guidelines

| Service | Min Replicas | Max Replicas | Typical Load | Scale Trigger |
|---------|--------------|--------------|--------------|---------------|
| Issuer | 3 | 20 | 100 VCs/sec | CPU > 70% |
| Verifier | 5 | 50 | 1000 verifications/sec | CPU > 70%, DB connections |

### Resource Requests vs Limits

Current settings:
- **Requests:** 256Mi memory, 250m CPU (guaranteed)
- **Limits:** 512Mi memory, 500m CPU (burst capacity)

Tune based on metrics:
```bash
# View actual resource usage
kubectl top pods -n credential-service
```

## Troubleshooting

### Pods Not Starting

```bash
# Check events
kubectl get events -n credential-service --sort-by='.lastTimestamp'

# Check pod status
kubectl describe pod <pod-name> -n credential-service

# Check logs
kubectl logs <pod-name> -n credential-service --previous
```

### Scaling Not Working

```bash
# Check metrics server
kubectl top nodes
kubectl top pods -n credential-service

# If metrics unavailable, install metrics-server
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

### Database Connection Issues

```bash
# Check postgres pod
kubectl get pods -n credential-service -l app=postgres

# Test connection from verifier
kubectl exec -it -n credential-service deployment/credential-verifier -- sh
# (if shell available, try: nc -zv postgres 5432)

# Check database logs
kubectl logs -n credential-service -l app=postgres
```
