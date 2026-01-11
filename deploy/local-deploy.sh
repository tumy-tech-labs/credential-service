#!/bin/bash
set -e

echo "🚀 Deploying Credential Service to Kind..."

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if kind cluster exists
if ! kind get clusters | grep -q "credential-service"; then
    echo -e "${BLUE}Creating Kind cluster...${NC}"
    kind create cluster --config deploy/kind-config.yaml --name credential-service
else
    echo -e "${GREEN}✓ Kind cluster already exists${NC}"
fi

# Load Docker images into Kind
echo -e "${BLUE}Building and loading Docker images...${NC}"
docker compose build issuer verifier

kind load docker-image credential-service-issuer:latest --name credential-service
kind load docker-image credential-service-verifier:latest --name credential-service

# Update Kubernetes image references for local testing
echo -e "${BLUE}Applying Kubernetes manifests...${NC}"

# Create namespace first
kubectl apply -f deploy/k8s/namespace.yaml

# Apply all manifests with local image references
cat deploy/k8s/postgres.yaml | kubectl apply -f -
cat deploy/k8s/issuer.yaml | sed 's|ghcr.io/bradtumy/credential-issuer:latest|credential-service-issuer:latest|' | kubectl apply -f -
cat deploy/k8s/verifier.yaml | sed 's|ghcr.io/bradtumy/credential-verifier:latest|credential-service-verifier:latest|' | kubectl apply -f -

echo -e "${BLUE}Waiting for pods to be ready...${NC}"
kubectl wait --for=condition=ready pod -l app=postgres -n credential-service --timeout=120s
kubectl wait --for=condition=ready pod -l app=credential-issuer -n credential-service --timeout=120s
kubectl wait --for=condition=ready pod -l app=credential-verifier -n credential-service --timeout=120s

echo -e "${GREEN}✓ Deployment complete!${NC}"
echo ""
echo "📊 Cluster Status:"
kubectl get pods -n credential-service
echo ""
echo "🔗 Port Forwarding:"
echo "  Issuer:   kubectl port-forward -n credential-service svc/credential-issuer 8080:8080"
echo "  Verifier: kubectl port-forward -n credential-service svc/credential-verifier 8081:8081"
echo ""
echo "📈 Scale Services:"
echo "  Issuer:   kubectl scale deployment credential-issuer -n credential-service --replicas=5"
echo "  Verifier: kubectl scale deployment credential-verifier -n credential-service --replicas=10"
echo ""
echo "🔍 View Logs:"
echo "  Issuer:   kubectl logs -n credential-service -l app=credential-issuer --tail=100 -f"
echo "  Verifier: kubectl logs -n credential-service -l app=credential-verifier --tail=100 -f"
