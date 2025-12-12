# Kubernetes Deployment Troubleshooting Guide

This guide provides solutions to common issues when deploying the Jellycat Draft application on Kubernetes with CloudNativePG.

## Table of Contents

- [PostgreSQL Connection Issues](#postgresql-connection-issues)
- [DNS Resolution Timeouts](#dns-resolution-timeouts)
- [Pod Startup Failures](#pod-startup-failures)
- [Network Policy Issues](#network-policy-issues)

## PostgreSQL Connection Issues

### Symptom: DNS Lookup Timeout

```
{"time":"2025-12-12T11:47:54.04937449Z","level":"ERROR","msg":"Failed to initialize Postgres","error":"failed to ping postgres: dial tcp: lookup jellycat-draft-db-rw.default.svc.cluster.local on 10.43.0.10:53: read udp 10.42.0.141:57013->10.43.0.10:53: i/o timeout"}
```

**Root Causes**:
1. Missing `connect_timeout` parameter in DATABASE_URL
2. DNS propagation delay when pod starts before DNS records are ready
3. CoreDNS performance issues in the cluster
4. Network policies blocking DNS queries

**Solution 1: Add connect_timeout to DATABASE_URL**

Update your DATABASE_URL secret to include `connect_timeout=60`:

```bash
# Check current DATABASE_URL
kubectl get secret jellycat-draft-auth-secrets -n default \
  -o jsonpath='{.data.DATABASE_URL}' | base64 -d
echo

# Update the secret with connect_timeout parameter
# Example format: postgres://user:password@host:5432/db?sslmode=require&connect_timeout=60
kubectl patch secret jellycat-draft-auth-secrets -n default --type='json' \
  -p='[{"op": "replace", "path": "/data/DATABASE_URL", "value": "'$(echo -n "postgres://jellycat-draft:PASSWORD@jellycat-draft-db-rw.default.svc.cluster.local:5432/jellycat-draft?sslmode=require&connect_timeout=60" | base64 -w0)'"}]'
```

**Solution 2: Verify CloudNativePG Service Name**

The service name must match your CloudNativePG cluster name:

```bash
# Get the correct service name from CloudNativePG
kubectl get cluster -n default -o yaml | grep writeService

# Example output:
# writeService: jellycat-draft-db-rw

# Verify service exists and has endpoints
kubectl get svc jellycat-draft-db-rw -n default
kubectl get endpoints jellycat-draft-db-rw -n default
```

**Solution 3: Test DNS Resolution**

```bash
# Get application pod name
APP_POD=$(kubectl get pods -n default -l app.kubernetes.io/name=jellycat-ui -o jsonpath='{.items[0].metadata.name}')

# Test DNS resolution from the pod
kubectl exec -n default $APP_POD -- nslookup jellycat-draft-db-rw.default.svc.cluster.local

# Test direct connection
kubectl exec -n default $APP_POD -- nc -zv jellycat-draft-db-rw.default.svc.cluster.local 5432
```

**Solution 4: Increase Startup Probe Delays**

Update your deployment to give more time for DNS resolution:

```yaml
spec:
  template:
    spec:
      containers:
      - name: ui
        readinessProbe:
          httpGet:
            path: /readyz
            port: http
          initialDelaySeconds: 30  # Increase from 5 to 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        livenessProbe:
          httpGet:
            path: /healthz
            port: http
          initialDelaySeconds: 45  # Increase from 15 to 45
          periodSeconds: 20
          timeoutSeconds: 5
          failureThreshold: 3
```

**Solution 5: Check CoreDNS**

```bash
# Check CoreDNS pods are running
kubectl get pods -n kube-system -l k8s-app=kube-dns

# Check CoreDNS logs for errors
kubectl logs -n kube-system -l k8s-app=kube-dns --tail=100

# Restart CoreDNS if needed
kubectl rollout restart deployment/coredns -n kube-system
```

**Solution 6: Application-Level Retry (Already Implemented)**

The application now includes automatic retry logic:
- 5 retry attempts with 5-second delays between retries
- 60-second timeout per connection attempt
- Total maximum wait time: ~5 minutes

This handles most DNS propagation delays automatically.

## DATABASE_URL Format Reference

### Correct Format

```
postgres://USERNAME:PASSWORD@SERVICE.NAMESPACE.svc.cluster.local:5432/DATABASE?sslmode=require&connect_timeout=60
```

### Example for jellycat-draft-db cluster

```
postgres://jellycat-draft:PASSWORD@jellycat-draft-db-rw.default.svc.cluster.local:5432/jellycat-draft?sslmode=require&connect_timeout=60
```

### URL Encoding Special Characters

If your password contains special characters, they must be URL-encoded:

| Character | Encoding | Example |
|-----------|----------|---------|
| `/` | `%2F` | `/dnt7` → `%2Fdnt7` |
| `@` | `%40` | `user@host` → `user%40host` |
| `=` | `%3D` | `key=val` → `key%3Dval` |
| `:` | `%3A` | `part:1` → `part%3A1` |
| `?` | `%3F` | `what?` → `what%3F` |
| `#` | `%23` | `tag#1` → `tag%231` |

Example with encoded password:
```
postgres://jellycat-draft:%2Fdnt7b6tPjM%3D@jellycat-draft-db-rw.default.svc.cluster.local:5432/jellycat-draft?sslmode=require&connect_timeout=60
```

## Pod Startup Failures

### Symptom: CrashLoopBackOff

```bash
# Check pod status
kubectl get pods -n default -l app.kubernetes.io/name=jellycat-ui

# View pod events
kubectl describe pod <pod-name> -n default

# Check logs
kubectl logs <pod-name> -n default --previous
```

**Common Causes**:
1. Missing required environment variables
2. Invalid DATABASE_URL
3. Database not ready
4. Missing secrets

**Solution**:

```bash
# Verify all required secrets exist
kubectl get secret jellycat-draft-auth-secrets -n default

# Check environment variables are set correctly
kubectl get deployment jellycat-draft-jellycat-ui -n default -o yaml | grep -A 20 "env:"

# Verify CloudNativePG cluster is ready
kubectl get cluster -n default
# Should show "Cluster in healthy state"
```

## Network Policy Issues

### Symptom: Connection Blocked

**Check for Network Policies**:

```bash
# List network policies in namespace
kubectl get networkpolicies -n default

# View network policy details
kubectl describe networkpolicy <policy-name> -n default
```

**Solution**: Ensure network policy allows:
1. Egress to DNS (kube-dns on port 53)
2. Egress to PostgreSQL service (port 5432)
3. Ingress for health checks (port 3000)

Example network policy:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: jellycat-draft-app
  namespace: default
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: jellycat-ui
  policyTypes:
  - Egress
  - Ingress
  egress:
  # Allow DNS
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: UDP
      port: 53
  # Allow PostgreSQL
  - to:
    - podSelector:
        matchLabels:
          cnpg.io/cluster: jellycat-draft-db
    ports:
    - protocol: TCP
      port: 5432
  # Allow all other egress (NATS, ClickHouse, Authentik, etc.)
  - to:
    - namespaceSelector: {}
  ingress:
  # Allow health checks
  - from:
    - podSelector: {}
    ports:
    - protocol: TCP
      port: 3000
```

## Quick Diagnostic Commands

```bash
# Check everything is running
kubectl get all -n default

# Check CloudNativePG cluster status
kubectl get cluster -n default
kubectl describe cluster jellycat-draft-db -n default

# Check PostgreSQL pods
kubectl get pods -n default -l cnpg.io/cluster=jellycat-draft-db

# Check application pods
kubectl get pods -n default -l app.kubernetes.io/name=jellycat-ui

# View application logs
kubectl logs -n default -l app.kubernetes.io/name=jellycat-ui --tail=100

# Check database connectivity from app pod
APP_POD=$(kubectl get pods -n default -l app.kubernetes.io/name=jellycat-ui -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n default $APP_POD -- nc -zv jellycat-draft-db-rw.default.svc.cluster.local 5432

# Test database login
kubectl exec -n default $APP_POD -- psql "postgres://jellycat-draft:PASSWORD@jellycat-draft-db-rw.default.svc.cluster.local:5432/jellycat-draft?sslmode=require" -c "SELECT 1"
```

## Getting More Help

If issues persist:

1. Collect diagnostic information:
```bash
# Save cluster info
kubectl cluster-info dump > cluster-dump.txt

# Save namespace resources
kubectl get all -n default -o yaml > namespace-resources.yaml

# Save logs
kubectl logs -n default -l app.kubernetes.io/name=jellycat-ui --all-containers=true > app-logs.txt
```

2. Check the main documentation:
   - [README.md](../README.md)
   - [Kubernetes CloudNativePG Guide](kubernetes-cloudnative-pg.md)
   - [PostgreSQL Setup](postgres-setup.md)

3. Review CloudNativePG documentation:
   - [CloudNativePG Troubleshooting](https://cloudnative-pg.io/documentation/current/troubleshooting/)

## Summary Checklist

When deploying to Kubernetes with CloudNativePG:

- [ ] CloudNativePG operator is installed and running
- [ ] CloudNativePG cluster shows "Cluster in healthy state"
- [ ] DATABASE_URL includes `connect_timeout=60` parameter
- [ ] DATABASE_URL service name matches CloudNativePG cluster name + `-rw` suffix
- [ ] Special characters in password are URL-encoded
- [ ] All required secrets exist in the namespace
- [ ] Network policies allow DNS and PostgreSQL traffic
- [ ] Readiness probe initialDelaySeconds is at least 30 seconds
- [ ] Application image version is up to date with retry logic
