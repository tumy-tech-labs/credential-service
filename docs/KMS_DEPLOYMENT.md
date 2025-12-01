# Production KMS Deployment Guide

This guide covers deploying the credential service with external key management systems (KMS) for production security.

## Overview

The credential service supports multiple key management backends:
- **HashiCorp Vault** (Recommended for most deployments)
- **Google Cloud KMS** (For GCP environments)
- **Memory Store** (Development only - not for production)

## Quick Start

Enable production keystore by setting:
```bash
export USE_PRODUCTION_KEYSTORE=true
```

The system will automatically detect and use the best available KMS backend.

## HashiCorp Vault Setup

### 1. Vault Server Setup

```bash
# Start Vault in development mode (for testing)
vault server -dev

# In production, use proper Vault configuration
vault server -config=/path/to/vault.hcl
```

### 2. Enable Transit Engine

```bash
# Enable transit secrets engine
vault secrets enable transit

# Create a key for the credential service
vault write -f transit/keys/credential-service-main type=ed25519
```

### 3. Environment Configuration

```bash
export USE_PRODUCTION_KEYSTORE=true
export VAULT_ADDR=https://vault.example.com:8200
export VAULT_TOKEN=your-vault-token
export VAULT_KEY_NAME=credential-service-main
```

### 4. Optional Configuration

```bash
# Custom mount path (default: transit)
export VAULT_MOUNT_PATH=credential-transit

# Auto-create keys if they don't exist
export VAULT_AUTO_CREATE_KEYS=true
```

## Google Cloud KMS Setup

### 1. Enable KMS API

```bash
gcloud services enable cloudkms.googleapis.com
```

### 2. Create Key Ring and Key

```bash
# Create key ring
gcloud kms keyrings create credential-service-ring \
    --location=global

# Create signing key
gcloud kms keys create credential-service-key \
    --location=global \
    --keyring=credential-service-ring \
    --purpose=asymmetric-signing \
    --default-algorithm=ec-sign-ed25519
```

### 3. Environment Configuration

```bash
export USE_PRODUCTION_KEYSTORE=true
export ENABLE_KMS=true
export GCP_PROJECT=your-project-id
export KMS_LOCATION=global
export KMS_KEYRING=credential-service-ring
export KMS_KEY_ID=credential-service-key
```

## Backend Selection Priority

The system automatically selects the backend in this order:

1. **Vault** - If `ENABLE_VAULT=true` and Vault is configured
2. **Google KMS** - If `ENABLE_KMS=true` and KMS is configured  
3. **Memory** - Fallback for development

### Force Specific Backend

```bash
# Force Vault backend
export KEY_BACKEND=vault

# Force Google KMS backend
export KEY_BACKEND=gcp-kms

# Force memory backend (development)
export KEY_BACKEND=memory
```

## Docker Deployment

### Vault Integration

```yaml
version: '3.8'
services:
  credential-service:
    image: credential-service:latest
    environment:
      - USE_PRODUCTION_KEYSTORE=true
      - VAULT_ADDR=http://vault:8200
      - VAULT_TOKEN=${VAULT_TOKEN}
      - VAULT_KEY_NAME=credential-service-main
    depends_on:
      - vault

  vault:
    image: vault:latest
    cap_add:
      - IPC_LOCK
    environment:
      - VAULT_DEV_ROOT_TOKEN_ID=${VAULT_TOKEN}
```

### GCP KMS Integration

```yaml
version: '3.8'
services:
  credential-service:
    image: credential-service:latest
    environment:
      - USE_PRODUCTION_KEYSTORE=true
      - ENABLE_KMS=true
      - GCP_PROJECT=${GCP_PROJECT}
      - KMS_LOCATION=global
      - KMS_KEYRING=credential-service-ring
    volumes:
      - ./service-account.json:/app/service-account.json
      - GOOGLE_APPLICATION_CREDENTIALS=/app/service-account.json
```

## Kubernetes Deployment

### Vault Integration with Secrets

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: credential-service
spec:
  template:
    spec:
      containers:
      - name: credential-service
        image: credential-service:latest
        env:
        - name: USE_PRODUCTION_KEYSTORE
          value: "true"
        - name: VAULT_ADDR
          value: "https://vault.default.svc.cluster.local:8200"
        - name: VAULT_TOKEN
          valueFrom:
            secretKeyRef:
              name: vault-secrets
              key: token
        - name: VAULT_KEY_NAME
          value: "credential-service-main"
---
apiVersion: v1
kind: Secret
metadata:
  name: vault-secrets
data:
  token: <base64-encoded-vault-token>
```

### GCP KMS with Workload Identity

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: credential-service
spec:
  template:
    metadata:
      annotations:
        iam.gke.io/gcp-service-account: credential-service@project.iam.gserviceaccount.com
    spec:
      serviceAccountName: credential-service-ksa
      containers:
      - name: credential-service
        image: credential-service:latest
        env:
        - name: USE_PRODUCTION_KEYSTORE
          value: "true"
        - name: ENABLE_KMS
          value: "true"
        - name: GCP_PROJECT
          value: "your-project-id"
        - name: KMS_LOCATION
          value: "global"
        - name: KMS_KEYRING
          value: "credential-service-ring"
        - name: KMS_KEY_ID
          value: "credential-service-key"
```

## Security Best Practices

### 1. Vault Security

```bash
# Use proper Vault authentication (not root tokens)
vault auth enable kubernetes
vault auth enable aws
vault auth enable gcp

# Create dedicated policy for credential service
vault policy write credential-service - <<EOF
path "transit/encrypt/credential-service-*" {
  capabilities = ["update"]
}
path "transit/decrypt/credential-service-*" {
  capabilities = ["update"]  
}
path "transit/sign/credential-service-*" {
  capabilities = ["update"]
}
path "transit/keys/credential-service-*" {
  capabilities = ["read", "create", "update"]
}
EOF
```

### 2. Key Rotation

```bash
# Rotate Vault keys
vault write -f transit/keys/credential-service-main/rotate

# The service automatically uses the latest key version
```

### 3. Monitoring and Auditing

```bash
# Enable Vault audit logging
vault audit enable file file_path=/var/log/vault/audit.log

# Monitor KMS operations in GCP
gcloud logging read 'resource.type="cloudkms_cryptokey"'
```

## Environment Variables Reference

### Global Settings
- `USE_PRODUCTION_KEYSTORE=true` - Enable production keystore
- `KEY_BACKEND=auto|vault|gcp-kms|memory` - Force specific backend
- `KEY_TENANT_PREFIX=prefix-` - Prefix for tenant-specific keys

### Vault Settings
- `ENABLE_VAULT=true` - Enable Vault backend
- `VAULT_ADDR` - Vault server address
- `VAULT_TOKEN` - Vault authentication token
- `VAULT_KEY_NAME` - Primary key name in Vault
- `VAULT_MOUNT_PATH=transit` - Transit engine mount path

### Google KMS Settings
- `ENABLE_KMS=true` - Enable Google KMS backend
- `GCP_PROJECT` - Google Cloud project ID
- `KMS_LOCATION=global` - KMS key location
- `KMS_KEYRING` - KMS key ring name
- `KMS_KEY_ID` - KMS key ID

## Troubleshooting

### Check Backend Status

The service logs will show which backend is active:
```
Using production keystore with backend: vault
Using production keystore with backend: gcp-kms
Using memory keystore (development mode)
```

### Vault Connection Issues

```bash
# Test Vault connectivity
vault status -address=${VAULT_ADDR}

# Test authentication
vault token lookup

# Test transit engine
vault read transit/keys/credential-service-main
```

### KMS Permission Issues

```bash
# Test KMS permissions
gcloud kms keys list --keyring=credential-service-ring --location=global

# Test signing operation
echo "test data" | gcloud kms asymmetric-sign \
    --key=credential-service-key \
    --keyring=credential-service-ring \
    --location=global \
    --digest-algorithm=sha256 \
    --input-file=-
```

### Common Error Messages

- **"Failed to initialize production keystore"** - Check environment variables and connectivity
- **"Vault authentication failed"** - Verify VAULT_TOKEN and permissions
- **"KMS permission denied"** - Check service account permissions
- **"Using memory keystore (development mode)"** - Production keystore not properly configured

## Migration from Memory Store

1. **Backup existing keys** (if needed for compatibility)
2. **Configure KMS backend** using environment variables
3. **Set USE_PRODUCTION_KEYSTORE=true**
4. **Restart services** - new keys will be generated in KMS
5. **Update trust registry** with new issuer DIDs if needed

The service will automatically generate new keys in the configured KMS backend. Existing credentials will need to be re-issued if backward compatibility is required.