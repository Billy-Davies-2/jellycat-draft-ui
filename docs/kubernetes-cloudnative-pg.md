# Kubernetes Deployment with CloudNativePG

This guide provides detailed instructions for deploying the Jellycat Fantasy Draft application on Kubernetes using the [CloudNativePG](https://cloudnative-pg.io/) operator for PostgreSQL database management.

## Table of Contents

- [Overview](#overview)
- [PostgreSQL Optimizations](#postgresql-optimizations)
- [Prerequisites](#prerequisites)
- [Installing CloudNativePG Operator](#installing-cloudnativepg-operator)
- [Creating a PostgreSQL Cluster](#creating-a-postgresql-cluster)
- [Connection Setup](#connection-setup)
- [Deploying the Application](#deploying-the-application)
- [Complete Deployment Example](#complete-deployment-example)
- [Monitoring and Management](#monitoring-and-management)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)
- [Additional Resources](#additional-resources)

## Overview

**CloudNativePG** (formerly known as CloudNative PostgreSQL) is a Kubernetes operator that manages the full lifecycle of PostgreSQL database clusters. It provides:

- **High Availability**: Automatic failover and replica management
- **Backup and Recovery**: Integrated backup to S3, Azure, GCS, and more
- **Connection Pooling**: Built-in PgBouncer support
- **Monitoring**: Prometheus metrics out of the box
- **Day-2 Operations**: Rolling updates, scaling, and maintenance

This application is fully compatible with CloudNativePG and requires PostgreSQL 12 or higher.

## PostgreSQL Optimizations

The application includes several CloudNativePG-specific optimizations:

### Connection Pool Configuration
- **MaxOpenConns: 25** - Prevents connection exhaustion (CloudNativePG default max_connections is 100)
- **MaxIdleConns: 5** - Maintains ready connections for quick reuse
- **ConnMaxLifetime: 5 minutes** - Gracefully handles failovers by recycling connections
- **ConnMaxIdleTime: 1 minute** - Reduces load by closing idle connections

### Query Optimizations
- **Eliminated N+1 Queries**: Teams and players are fetched using a single JOIN query
- **Batch Inserts**: Seed data uses prepared statements and transactions
- **Context Timeouts**: All operations include context timeouts for failover handling
- **Strategic Indexes**: Added indexes for common query patterns (points DESC, created_at, etc.)

### Transaction Improvements
- **Combined Queries**: DraftPlayer uses CTEs to reduce round trips
- **Batch Operations**: Seed operations use transactions with prepared statements
- **Failover Handling**: Context timeouts ensure operations complete or fail quickly

These optimizations significantly improve performance, especially with CloudNativePG's high-availability features and read replicas.

## Prerequisites

Before you begin, ensure you have:

1. **Kubernetes Cluster**: Version 1.25 or higher
   ```bash
   kubectl version --short
   ```

2. **kubectl**: Configured to communicate with your cluster
   ```bash
   kubectl cluster-info
   ```

3. **Helm** (optional but recommended): Version 3.x
   ```bash
   helm version
   ```

4. **Cluster Permissions**: Ability to create namespaces, deployments, and custom resources

5. **Storage**: A StorageClass that supports dynamic provisioning
   ```bash
   kubectl get storageclass
   ```

## Installing CloudNativePG Operator

### Option 1: Using Helm (Recommended)

```bash
# Add the CloudNativePG Helm repository
helm repo add cnpg https://cloudnative-pg.github.io/charts
helm repo update

# Install the operator in the cnpg-system namespace
helm install cnpg \
  --namespace cnpg-system \
  --create-namespace \
  cnpg/cloudnative-pg
```

### Option 2: Using kubectl

```bash
# Install the latest stable version
kubectl apply -f \
  https://raw.githubusercontent.com/cloudnative-pg/cloudnative-pg/release-1.21/releases/cnpg-1.21.0.yaml
```

### Verify Installation

```bash
# Check operator pods are running
kubectl get pods -n cnpg-system

# Expected output:
# NAME                                READY   STATUS    RESTARTS   AGE
# cnpg-cloudnative-pg-xxxxx-xxxxx    1/1     Running   0          1m
```

## Creating a PostgreSQL Cluster

### Basic Cluster Configuration

Create a file named `postgres-cluster.yaml`:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: jellycat-postgres
  namespace: jellycat-draft
spec:
  # Number of PostgreSQL instances
  instances: 3
  
  # PostgreSQL version (12-17 supported)
  imageName: ghcr.io/cloudnative-pg/postgresql:16.1
  
  # Storage configuration
  storage:
    size: 10Gi
    storageClass: standard  # Use your StorageClass name
  
  # Bootstrap a new database
  bootstrap:
    initdb:
      database: jellycatdraft
      owner: jellycatuser
      secret:
        name: jellycat-postgres-app
  
  # Monitoring (Prometheus)
  monitoring:
    enablePodMonitor: true
  
  # Resource limits
  resources:
    requests:
      memory: "256Mi"
      cpu: "250m"
    limits:
      memory: "1Gi"
      cpu: "1000m"
  
  # PostgreSQL configuration
  postgresql:
    parameters:
      max_connections: "100"
      shared_buffers: "256MB"
      effective_cache_size: "1GB"
      work_mem: "4MB"
      maintenance_work_mem: "64MB"
```

### Create the Namespace and Secret

```bash
# Create namespace
kubectl create namespace jellycat-draft

# Create a secret for the database credentials
# CloudNativePG will use this to create the user
kubectl create secret generic jellycat-postgres-app \
  --namespace jellycat-draft \
  --from-literal=username=jellycatuser \
  --from-literal=password=$(openssl rand -base64 32)
```

### Deploy the Cluster

```bash
# Apply the cluster configuration
kubectl apply -f postgres-cluster.yaml

# Watch the cluster come online
kubectl get cluster -n jellycat-draft -w

# Expected output after a few minutes:
# NAME                 AGE   INSTANCES   READY   STATUS                     PRIMARY
# jellycat-postgres   2m    3           3       Cluster in healthy state   jellycat-postgres-1
```

### Verify Database Pods

```bash
# Check PostgreSQL pods
kubectl get pods -n jellycat-draft -l cnpg.io/cluster=jellycat-postgres

# Expected output:
# NAME                  READY   STATUS    RESTARTS   AGE
# jellycat-postgres-1   1/1     Running   0          3m
# jellycat-postgres-2   1/1     Running   0          2m
# jellycat-postgres-3   1/1     Running   0          1m
```

## Connection Setup

### Understanding Connection Secrets

CloudNativePG automatically creates connection secrets when the cluster is ready:

- **`jellycat-postgres-app`**: Application credentials (read-write)
- **`jellycat-postgres-superuser`**: Superuser credentials (admin only)
- **`jellycat-postgres-rw`**: Read-write service endpoint
- **`jellycat-postgres-ro`**: Read-only service endpoint (replicas)
- **`jellycat-postgres-r`**: Any instance endpoint

### View Connection Details

```bash
# Get the application secret
kubectl get secret jellycat-postgres-app -n jellycat-draft -o yaml

# Decode and view the password (for debugging)
kubectl get secret jellycat-postgres-app -n jellycat-draft \
  -o jsonpath='{.data.password}' | base64 -d
echo

# Get connection URI
kubectl get secret jellycat-postgres-app -n jellycat-draft \
  -o jsonpath='{.data.uri}' | base64 -d
echo
```

### Connection String Format

CloudNativePG provides the following service endpoints:

- **Read-Write Service**: `jellycat-postgres-rw.jellycat-draft.svc.cluster.local:5432`
- **Read-Only Service**: `jellycat-postgres-ro.jellycat-draft.svc.cluster.local:5432`
- **Primary Service**: `jellycat-postgres-r.jellycat-draft.svc.cluster.local:5432`

For this application, use the **read-write service** (`-rw`):

```
postgres://jellycatuser:[PASSWORD]@jellycat-postgres-rw.jellycat-draft.svc.cluster.local:5432/jellycatdraft?sslmode=require&connect_timeout=60
```

**Important Connection Parameters**:
- `sslmode=require` - Enforces SSL/TLS connections (required for CloudNativePG)
- `connect_timeout=60` - Sets 60-second timeout for initial connection (recommended for Kubernetes DNS resolution delays)

**Example for cluster named `jellycat-draft-db`**:
```
postgres://jellycat-draft:PASSWORD@jellycat-draft-db-rw.default.svc.cluster.local:5432/jellycat-draft?sslmode=require&connect_timeout=60
```

**Password Encoding**: If your password contains special characters, ensure they are URL-encoded:
- `/` becomes `%2F`
- `@` becomes `%40`
- `=` becomes `%3D`
- `:` becomes `%3A`

**Note**: Replace cluster name, namespace, database name, and credentials based on your CloudNativePG cluster configuration:
- Cluster name `jellycat-postgres` → service name `jellycat-postgres-rw`
- Cluster name `jellycat-draft-db` → service name `jellycat-draft-db-rw`

## Deploying the Application

### Create Application Deployment

Create `jellycat-deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jellycat-draft
  namespace: jellycat-draft
  labels:
    app: jellycat-draft
spec:
  replicas: 2
  selector:
    matchLabels:
      app: jellycat-draft
  template:
    metadata:
      labels:
        app: jellycat-draft
    spec:
      containers:
      - name: jellycat-draft
        image: your-registry/jellycat-draft:latest
        ports:
        - name: http
          containerPort: 3000
          protocol: TCP
        - name: grpc
          containerPort: 50051
          protocol: TCP
        env:
        # Database Configuration
        - name: DB_DRIVER
          value: "postgres"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: jellycat-postgres-app
              key: uri
        
        # Environment Mode
        - name: ENVIRONMENT
          value: "production"
        
        # NATS Configuration
        - name: NATS_URL
          value: "nats://nats.jellycat-draft.svc.cluster.local:4222"
        - name: NATS_SUBJECT
          value: "draft.events"
        
        # ClickHouse Configuration
        - name: CLICKHOUSE_ADDR
          value: "clickhouse.jellycat-draft.svc.cluster.local:9000"
        - name: CLICKHOUSE_DB
          value: "default"
        - name: CLICKHOUSE_USER
          value: "default"
        - name: CLICKHOUSE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: clickhouse-credentials
              key: password
              optional: true
        
        # Authentik OAuth2
        - name: AUTHENTIK_BASE_URL
          value: "https://auth.yourdomain.com"
        - name: AUTHENTIK_CLIENT_ID
          valueFrom:
            secretKeyRef:
              name: authentik-oauth
              key: client-id
        - name: AUTHENTIK_CLIENT_SECRET
          valueFrom:
            secretKeyRef:
              name: authentik-oauth
              key: client-secret
        - name: AUTHENTIK_REDIRECT_URL
          value: "https://jellycat.yourdomain.com/auth/callback"
        
        # Health checks
        livenessProbe:
          httpGet:
            path: /healthz
            port: 3000
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /readyz
            port: 3000
          initialDelaySeconds: 5
          periodSeconds: 5
        
        # Resource limits
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: jellycat-draft
  namespace: jellycat-draft
  labels:
    app: jellycat-draft
spec:
  type: ClusterIP
  ports:
  - name: http
    port: 80
    targetPort: 3000
    protocol: TCP
  - name: grpc
    port: 50051
    targetPort: 50051
    protocol: TCP
  selector:
    app: jellycat-draft
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: jellycat-draft
  namespace: jellycat-draft
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - jellycat.yourdomain.com
    secretName: jellycat-tls
  rules:
  - host: jellycat.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: jellycat-draft
            port:
              number: 80
```

### Create OAuth Secret

```bash
# Create Authentik OAuth credentials
kubectl create secret generic authentik-oauth \
  --namespace jellycat-draft \
  --from-literal=client-id="your-client-id" \
  --from-literal=client-secret="your-client-secret"
```

### Deploy Application

```bash
# Apply the deployment
kubectl apply -f jellycat-deployment.yaml

# Check deployment status
kubectl get deployments -n jellycat-draft

# Check pods
kubectl get pods -n jellycat-draft -l app=jellycat-draft

# View logs
kubectl logs -n jellycat-draft -l app=jellycat-draft --tail=50 -f
```

## Complete Deployment Example

Here's a complete manifest that deploys everything in one go:

```yaml
---
# Namespace
apiVersion: v1
kind: Namespace
metadata:
  name: jellycat-draft

---
# PostgreSQL User Secret
apiVersion: v1
kind: Secret
metadata:
  name: jellycat-postgres-app
  namespace: jellycat-draft
type: kubernetes.io/basic-auth
stringData:
  username: jellycatuser
  # Generate a strong password in production
  password: "CHANGE_ME_IN_PRODUCTION"

---
# PostgreSQL Cluster
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: jellycat-postgres
  namespace: jellycat-draft
spec:
  instances: 3
  imageName: ghcr.io/cloudnative-pg/postgresql:16.1
  
  storage:
    size: 10Gi
  
  bootstrap:
    initdb:
      database: jellycatdraft
      owner: jellycatuser
      secret:
        name: jellycat-postgres-app
  
  monitoring:
    enablePodMonitor: true
  
  postgresql:
    parameters:
      max_connections: "100"
      shared_buffers: "256MB"

---
# OAuth Credentials (update with your values)
apiVersion: v1
kind: Secret
metadata:
  name: authentik-oauth
  namespace: jellycat-draft
type: Opaque
stringData:
  client-id: "your-client-id-here"
  client-secret: "your-client-secret-here"

---
# Application Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jellycat-draft
  namespace: jellycat-draft
spec:
  replicas: 2
  selector:
    matchLabels:
      app: jellycat-draft
  template:
    metadata:
      labels:
        app: jellycat-draft
    spec:
      containers:
      - name: jellycat-draft
        image: your-registry/jellycat-draft:latest
        ports:
        - containerPort: 3000
        - containerPort: 50051
        env:
        - name: DB_DRIVER
          value: "postgres"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: jellycat-postgres-app
              key: uri
        - name: ENVIRONMENT
          value: "development"  # Use "production" for real deployments
        - name: AUTHENTIK_BASE_URL
          value: "https://auth.yourdomain.com"
        - name: AUTHENTIK_CLIENT_ID
          valueFrom:
            secretKeyRef:
              name: authentik-oauth
              key: client-id
        - name: AUTHENTIK_CLIENT_SECRET
          valueFrom:
            secretKeyRef:
              name: authentik-oauth
              key: client-secret
        - name: AUTHENTIK_REDIRECT_URL
          value: "http://jellycat.yourdomain.com/auth/callback"
        livenessProbe:
          httpGet:
            path: /healthz
            port: 3000
          initialDelaySeconds: 10
        readinessProbe:
          httpGet:
            path: /readyz
            port: 3000
          initialDelaySeconds: 5
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"

---
# Service
apiVersion: v1
kind: Service
metadata:
  name: jellycat-draft
  namespace: jellycat-draft
spec:
  type: ClusterIP
  ports:
  - name: http
    port: 80
    targetPort: 3000
  - name: grpc
    port: 50051
    targetPort: 50051
  selector:
    app: jellycat-draft
```

Save this as `complete-deployment.yaml` and deploy:

```bash
# Deploy everything
kubectl apply -f complete-deployment.yaml

# Wait for cluster to be ready (2-5 minutes)
kubectl wait --for=condition=Ready cluster/jellycat-postgres \
  -n jellycat-draft \
  --timeout=300s

# Check status
kubectl get all -n jellycat-draft
```

## Monitoring and Management

### Check Cluster Status

```bash
# View cluster details
kubectl describe cluster jellycat-postgres -n jellycat-draft

# Check cluster status
kubectl get cluster -n jellycat-draft

# View logs from primary
kubectl logs -n jellycat-draft jellycat-postgres-1 -f
```

### Database Access

```bash
# Connect to the primary database
kubectl exec -it jellycat-postgres-1 -n jellycat-draft -- psql -U jellycatuser jellycatdraft

# Run a query
kubectl exec -it jellycat-postgres-1 -n jellycat-draft -- \
  psql -U jellycatuser -d jellycatdraft -c "SELECT COUNT(*) FROM players;"
```

### Monitoring with Prometheus

If you have Prometheus Operator installed:

```bash
# CloudNativePG automatically creates PodMonitor
kubectl get podmonitor -n jellycat-draft

# View metrics endpoint
kubectl port-forward -n jellycat-draft jellycat-postgres-1 9187:9187

# Access metrics at http://localhost:9187/metrics
```

### Backup Configuration

Add backup configuration to your cluster:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: jellycat-postgres
  namespace: jellycat-draft
spec:
  # ... existing spec ...
  
  backup:
    # Backup schedule (daily at 2 AM)
    retentionPolicy: "30d"
    barmanObjectStore:
      destinationPath: s3://your-bucket/postgres-backups/
      endpointURL: https://s3.amazonaws.com
      s3Credentials:
        accessKeyId:
          name: aws-credentials
          key: ACCESS_KEY_ID
        secretAccessKey:
          name: aws-credentials
          key: ACCESS_SECRET_KEY
      wal:
        compression: gzip
    
  # Scheduled backups
  scheduledBackup:
    - name: daily-backup
      schedule: "0 2 * * *"  # 2 AM daily
      backupOwnerReference: self
```

Create a backup manually:

```bash
# Create an on-demand backup
kubectl create -f - <<EOF
apiVersion: postgresql.cnpg.io/v1
kind: Backup
metadata:
  name: jellycat-backup-$(date +%Y%m%d-%H%M%S)
  namespace: jellycat-draft
spec:
  cluster:
    name: jellycat-postgres
EOF

# List backups
kubectl get backups -n jellycat-draft
```

## Best Practices

### 1. **High Availability**

- **Run at least 3 instances** for production to ensure quorum
- Use **anti-affinity** to spread pods across nodes:

```yaml
spec:
  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
          - key: cnpg.io/cluster
            operator: In
            values:
            - jellycat-postgres
        topologyKey: kubernetes.io/hostname
```

### 2. **Resource Management**

- Set appropriate **resource requests and limits**
- Monitor memory usage - PostgreSQL can be memory-intensive
- Use **connection pooling** (PgBouncer) for high-traffic scenarios

### 3. **Security**

- **Always use SSL/TLS** for database connections (`sslmode=require`)
- Store credentials in **Kubernetes Secrets**, never in code
- Use **RBAC** to restrict access to database pods
- Enable **Pod Security Standards**
- Rotate passwords regularly using CloudNativePG's password rotation feature

### 4. **Backup and Recovery**

- Configure **automated backups** to object storage (S3, GCS, Azure Blob)
- Test recovery procedures regularly
- Keep backups for compliance requirements (typically 30-90 days)
- Use **Point-in-Time Recovery (PITR)** for granular recovery

### 5. **Monitoring**

- Enable **Prometheus metrics** for monitoring
- Set up alerts for:
  - Database replication lag
  - Connection pool exhaustion
  - Storage space running low
  - Failed backups
- Monitor application logs for database connection errors

### 6. **Upgrades**

- Use CloudNativePG's **rolling update** feature for zero-downtime upgrades
- Test upgrades in a non-production environment first
- Review the [PostgreSQL release notes](https://www.postgresql.org/docs/release/) before upgrading

### 7. **Network Policies**

Restrict network access to the database:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: postgres-network-policy
  namespace: jellycat-draft
spec:
  podSelector:
    matchLabels:
      cnpg.io/cluster: jellycat-postgres
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: jellycat-draft
    ports:
    - protocol: TCP
      port: 5432
```

## Troubleshooting

### Cluster Won't Start

**Symptoms**: Pods stuck in `CrashLoopBackOff` or `Pending`

**Solutions**:
```bash
# Check pod events
kubectl describe pod jellycat-postgres-1 -n jellycat-draft

# Common issues:
# 1. Insufficient storage
kubectl get pvc -n jellycat-draft

# 2. StorageClass not available
kubectl get storageclass

# 3. Resource constraints
kubectl describe nodes | grep -A 5 "Allocated resources"
```

### Connection Refused / DNS Timeout

**Symptoms**: Application can't connect to database, DNS lookup timeouts like `lookup jellycat-postgres-rw.default.svc.cluster.local on 10.43.0.10:53: i/o timeout`

**Common Causes**:
1. **Incorrect service name**: Ensure you're using the correct CloudNativePG service name
2. **DNS propagation delay**: DNS records may take time to propagate in Kubernetes
3. **Network policies**: Restrictive network policies blocking DNS or database traffic
4. **CoreDNS issues**: DNS server issues in the cluster

**Solutions**:
```bash
# 1. Verify the correct service name from CloudNativePG cluster
kubectl get cluster <cluster-name> -n <namespace> -o yaml | grep writeService
# Output example: writeService: jellycat-draft-db-rw

# 2. Verify service exists and has endpoints
kubectl get svc -n jellycat-draft | grep postgres
kubectl get endpoints -n jellycat-draft | grep postgres

# 3. Check if DNS is resolving from the application pod
kubectl exec -it <app-pod> -n jellycat-draft -- \
  nslookup jellycat-postgres-rw.jellycat-draft.svc.cluster.local

# 4. Test connectivity from application pod
kubectl exec -it <app-pod> -n jellycat-draft -- \
  nc -zv jellycat-postgres-rw.jellycat-draft.svc.cluster.local 5432

# 5. Check database logs
kubectl logs jellycat-postgres-1 -n jellycat-draft

# 6. Verify DATABASE_URL format in secret
# Correct format: postgres://user:password@service-name.namespace.svc.cluster.local:5432/database?sslmode=require&connect_timeout=60
kubectl get secret jellycat-postgres-app -n jellycat-draft \
  -o jsonpath='{.data.uri}' | base64 -d
echo

# 7. If using a custom secret, ensure DATABASE_URL includes connect_timeout
# Example: postgres://jellycatuser:password@jellycat-draft-db-rw.default.svc.cluster.local:5432/jellycat-draft?sslmode=require&connect_timeout=60

# 8. Check CoreDNS logs for DNS issues
kubectl logs -n kube-system -l k8s-app=kube-dns --tail=50

# 9. Verify network policies aren't blocking traffic
kubectl get networkpolicies -n jellycat-draft
```

**Fix**: The application now includes retry logic with 60-second timeout per attempt and up to 5 retries to handle DNS propagation delays in Kubernetes. Ensure your DATABASE_URL includes `connect_timeout=60` parameter.

### Authentication Failed

**Symptoms**: `password authentication failed for user`

**Solutions**:
```bash
# Verify credentials in secret
kubectl get secret jellycat-postgres-app -n jellycat-draft \
  -o jsonpath='{.data.username}' | base64 -d
echo

# Recreate user if needed (connect as superuser)
kubectl exec -it jellycat-postgres-1 -n jellycat-draft -- \
  psql -U postgres -d jellycatdraft -c \
  "ALTER USER jellycatuser WITH PASSWORD 'newpassword';"

# Update secret
kubectl create secret generic jellycat-postgres-app \
  --namespace jellycat-draft \
  --from-literal=username=jellycatuser \
  --from-literal=password=newpassword \
  --dry-run=client -o yaml | kubectl apply -f -
```

### Replication Lag

**Symptoms**: Replicas falling behind primary

**Solutions**:
```bash
# Check replication status
kubectl exec -it jellycat-postgres-1 -n jellycat-draft -- \
  psql -U postgres -c "SELECT * FROM pg_stat_replication;"

# Check cluster status
kubectl get cluster jellycat-postgres -n jellycat-draft -o yaml

# Review PostgreSQL configuration
kubectl exec -it jellycat-postgres-1 -n jellycat-draft -- \
  psql -U postgres -c "SHOW wal_sender_timeout;"
```

### Backup Failures

**Symptoms**: Scheduled backups failing

**Solutions**:
```bash
# Check backup status
kubectl get backups -n jellycat-draft

# View backup logs
kubectl describe backup <backup-name> -n jellycat-draft

# Common issues:
# 1. S3 credentials incorrect
kubectl get secret aws-credentials -n jellycat-draft -o yaml

# 2. Network connectivity to S3
kubectl exec -it jellycat-postgres-1 -n jellycat-draft -- \
  curl -v https://s3.amazonaws.com

# 3. Insufficient permissions
# Ensure IAM role/user has s3:PutObject, s3:GetObject permissions
```

### Database Performance Issues

**Symptoms**: Slow queries, high CPU/memory usage

**Solutions**:
```bash
# Check current connections
kubectl exec -it jellycat-postgres-1 -n jellycat-draft -- \
  psql -U postgres -c "SELECT count(*) FROM pg_stat_activity;"

# Find slow queries
kubectl exec -it jellycat-postgres-1 -n jellycat-draft -- \
  psql -U postgres -c "SELECT pid, now() - query_start AS duration, query 
  FROM pg_stat_activity WHERE state = 'active' AND now() - query_start > interval '5 seconds';"

# Check database size
kubectl exec -it jellycat-postgres-1 -n jellycat-draft -- \
  psql -U jellycatuser -d jellycatdraft -c "\l+"

# Analyze tables
kubectl exec -it jellycat-postgres-1 -n jellycat-draft -- \
  psql -U jellycatuser -d jellycatdraft -c "ANALYZE VERBOSE;"
```

### Pod Evicted or OOM Killed

**Symptoms**: Pods restarting with `OOMKilled` status

**Solutions**:
```bash
# Increase memory limits
kubectl edit cluster jellycat-postgres -n jellycat-draft

# Update resources:
#   resources:
#     limits:
#       memory: "2Gi"  # Increase from 1Gi

# Reduce shared_buffers if needed
kubectl edit cluster jellycat-postgres -n jellycat-draft

# Update postgresql parameters:
#   postgresql:
#     parameters:
#       shared_buffers: "128MB"  # Reduce from 256MB
```

## Additional Resources

### Official Documentation

- **CloudNativePG**: https://cloudnative-pg.io/documentation/
- **PostgreSQL**: https://www.postgresql.org/docs/
- **Kubernetes**: https://kubernetes.io/docs/

### CloudNativePG Resources

- **GitHub Repository**: https://github.com/cloudnative-pg/cloudnative-pg
- **Architecture**: https://cloudnative-pg.io/documentation/current/architecture/
- **API Reference**: https://cloudnative-pg.io/documentation/current/api_reference/
- **Backup and Recovery**: https://cloudnative-pg.io/documentation/current/backup_recovery/
- **Monitoring**: https://cloudnative-pg.io/documentation/current/monitoring/

### Application Resources

- **PostgreSQL Setup Guide**: [postgres-setup.md](postgres-setup.md)
- **Application README**: [../README.md](../README.md)
- **Authentication Setup**: [AUTH-MIGRATION.md](AUTH-MIGRATION.md)

### Community

- **CloudNativePG Slack**: [#cloudnative-pg on CNCF Slack](https://cloud-native.slack.com/)
- **GitHub Discussions**: https://github.com/cloudnative-pg/cloudnative-pg/discussions

## Quick Reference

### CloudNativePG kubectl Plugin (Optional)

The CloudNativePG kubectl plugin provides convenient commands for managing clusters. Install it with:

```bash
# Install via krew
kubectl krew install cnpg

# Or download from GitHub releases
curl -sSfL \
  https://github.com/cloudnative-pg/cloudnative-pg/raw/main/hack/install-kubectl-cnpg-plugin.sh | \
  sudo sh -s -- -b /usr/local/bin
```

### Common Commands

```bash
# Install operator
helm install cnpg cnpg/cloudnative-pg -n cnpg-system --create-namespace

# Create cluster
kubectl apply -f postgres-cluster.yaml

# Check cluster status
kubectl get cluster -n jellycat-draft

# Connect to database
kubectl exec -it jellycat-postgres-1 -n jellycat-draft -- psql -U jellycatuser jellycatdraft

# View logs
kubectl logs -n jellycat-draft jellycat-postgres-1 -f

# Create backup (requires kubectl-cnpg plugin OR use kubectl create -f backup.yaml)
kubectl cnpg backup jellycat-postgres -n jellycat-draft
# Alternative without plugin:
# kubectl create -f backup-manifest.yaml

# List backups
kubectl get backups -n jellycat-draft

# Promote replica to primary (requires kubectl-cnpg plugin OR manually patch the cluster)
kubectl cnpg promote jellycat-postgres-2 -n jellycat-draft
# Alternative without plugin:
# kubectl patch cluster jellycat-postgres -n jellycat-draft --type merge \
#   -p '{"spec":{"primaryUpdateStrategy":"unsupervised","primaryUpdateMethod":"switchover"}}'

# Scale cluster (standard kubectl command)
kubectl patch cluster jellycat-postgres -n jellycat-draft --type merge \
  -p '{"spec":{"instances":5}}'
```

### Environment Variables for Application

| Variable | Description | Example |
|----------|-------------|---------|
| `DB_DRIVER` | Database driver | `postgres` |
| `DATABASE_URL` | PostgreSQL connection string | Use secret reference |
| `ENVIRONMENT` | Environment mode | `production` |

### Connection Endpoints

| Service | Endpoint | Use Case |
|---------|----------|----------|
| Read-Write | `jellycat-postgres-rw` | Application writes |
| Read-Only | `jellycat-postgres-ro` | Read replicas (reporting) |
| Primary | `jellycat-postgres-r` | Any instance |

---

**Note**: This application is fully compatible with CloudNativePG without any code changes. The standard PostgreSQL driver (`lib/pq`) and SQL features used work seamlessly with CloudNativePG-managed clusters.

For questions or issues, please refer to the [main README](../README.md) or open an issue on GitHub.
