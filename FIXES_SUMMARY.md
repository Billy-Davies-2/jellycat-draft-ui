# Repository Fixes - Implementation Summary

This document summarizes all the changes made to address the issues in the problem statement.

## Issues Addressed

### ✅ 1. PostgreSQL Connection Timeout in Kubernetes

**Problem**: Application failing to connect to CloudNativePG database with DNS timeout errors:
```
lookup jellycat-draft-db-rw.default.svc.cluster.local on 10.43.0.10:53: i/o timeout
```

**Solution Implemented**:
- Added retry logic with 5 attempts and 60-second timeout per attempt in `internal/dal/postgres.go`
- Total maximum wait time: ~5 minutes (handles DNS propagation delays)
- Created comprehensive troubleshooting guide: `docs/kubernetes-troubleshooting.md`
- Updated CloudNativePG documentation with proper DATABASE_URL format

**Action Required**:
1. Update your DATABASE_URL secret to include `connect_timeout=60`:
   ```bash
   # Current format (missing connect_timeout):
   postgres://jellycat-draft:PASSWORD@jellycat-draft-db-rw.default.svc.cluster.local:5432/jellycat-draft?sslmode=require
   
   # Updated format (add connect_timeout=60):
   postgres://jellycat-draft:PASSWORD@jellycat-draft-db-rw.default.svc.cluster.local:5432/jellycat-draft?sslmode=require&connect_timeout=60
   ```

2. Update the secret in Kubernetes:
   ```bash
   kubectl patch secret jellycat-draft-auth-secrets -n default --type='json' \
     -p='[{"op": "replace", "path": "/data/DATABASE_URL", "value": "'$(echo -n "postgres://jellycat-draft:YOUR_PASSWORD@jellycat-draft-db-rw.default.svc.cluster.local:5432/jellycat-draft?sslmode=require&connect_timeout=60" | base64 -w0)'"}]'
   ```

3. Rebuild and redeploy the application image with the new retry logic

4. If issues persist, see `docs/kubernetes-troubleshooting.md` for detailed solutions

---

### ✅ 2. gRPC Endpoints - Keep or Remove?

**Decision**: **KEEP gRPC** - It's essential for the application!

**Why gRPC is Required**:
- ✅ **NATS Event Streaming**: `StreamEvents()` provides real-time updates via NATS pub/sub
- ✅ **Chat Integration**: Chat messages use NATS for distributed messaging
- ✅ **Programmatic API**: Type-safe API for external services and microservices
- ✅ **Dual Interface**: HTTP for browsers, gRPC for services

**Documentation Created**:
- `docs/grpc-nats-architecture.md` - Explains the full gRPC + NATS integration
- Updated README with clarification about gRPC usage

**No Action Required**: Do NOT remove gRPC endpoints. They are actively used for NATS-based real-time messaging.

---

### ✅ 3. Admin Panel - Verification and Documentation

**Status**: Admin panel is properly implemented and distinct!

**Features Verified**:
- ✅ **Access Control**: Requires authentication AND admin role (users in `admins` group)
- ✅ **Add Jellycats**: Form to add new Jellycat plush toys to the database
- ✅ **Modify Cuddle Points**: API endpoint to update player points
- ✅ **Manage Draft Scores**: Automatic draft score adjustments based on pick number
- ✅ **Draft Controls**: Reset draft functionality with confirmation

**Documentation Created**:
- `docs/admin-panel-guide.md` - Comprehensive guide to all admin features
- Updated README with link to admin panel documentation

**Action Required**:
- Review `docs/admin-panel-guide.md` to understand all admin capabilities
- Ensure users who need admin access are in the `admins` group in Authentik

---

### ✅ 4. TailwindCSS in Dockerfile

**Problem**: TailwindCSS was not being compiled during Docker builds

**Solution Implemented**:
- Updated `Dockerfile` to download TailwindCSS standalone CLI
- Added build step to compile `static/css/input.css` → `static/css/styles.css`
- Compiled styles are automatically included in the final Docker image
- No manual CSS compilation needed for Docker deployments

**Changes Made**:
```dockerfile
# Download TailwindCSS standalone CLI
RUN curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 && \
    chmod +x tailwindcss-linux-x64

# Build TailwindCSS
RUN ./tailwindcss-linux-x64 -i static/css/input.css -o static/css/styles.css --minify
```

**Action Required**:
1. Rebuild Docker image to get automatic CSS compilation:
   ```bash
   docker build -t jellycat-draft:latest .
   ```

2. Push to your registry:
   ```bash
   docker tag jellycat-draft:latest ghcr.io/billy-davies-2/jellycat-draft-ui:v7
   docker push ghcr.io/billy-davies-2/jellycat-draft-ui:v7
   ```

3. Update Kubernetes deployment to use new image version

---

## New Documentation Files

### 1. `docs/kubernetes-troubleshooting.md`
Comprehensive troubleshooting guide for Kubernetes deployment issues:
- DNS timeout solutions
- DATABASE_URL format reference
- Network policy issues
- Pod startup failures
- Quick diagnostic commands

### 2. `docs/grpc-nats-architecture.md`
Architecture documentation explaining:
- How gRPC and NATS work together
- Why gRPC is essential for real-time messaging
- Event streaming architecture
- Use cases for HTTP SSE vs gRPC

### 3. `docs/admin-panel-guide.md`
Complete admin panel documentation:
- Access control requirements
- Adding new Jellycats
- Managing cuddle points
- Draft score mechanics
- Best practices

---

## Files Modified

### `internal/dal/postgres.go`
- Added retry logic with 5 attempts
- Increased timeout from 10s to 60s per attempt
- Better error messages showing retry count

### `Dockerfile`
- Added curl to dependencies
- Download TailwindCSS CLI during build
- Compile CSS before copying static files

### `docs/kubernetes-cloudnative-pg.md`
- Added DNS timeout troubleshooting section
- Updated DATABASE_URL format examples
- Added password encoding reference
- Enhanced connection string documentation

### `README.md`
- Added links to new documentation
- Clarified gRPC usage and importance
- Updated Docker build section
- Added note about automatic TailwindCSS compilation

---

## Testing Checklist

Before deploying to production:

- [ ] PostgreSQL connection succeeds with new retry logic
- [ ] DATABASE_URL includes `connect_timeout=60` parameter
- [ ] Docker image builds successfully with TailwindCSS compilation
- [ ] Compiled CSS (`static/css/styles.css`) is included in Docker image
- [ ] Admin panel accessible only to users in `admins` group
- [ ] gRPC endpoints are functional (test with `grpcurl` or client)
- [ ] NATS pub/sub working for real-time events
- [ ] Application starts without DNS timeout errors

---

## Deployment Steps

1. **Update DATABASE_URL Secret** (as shown above)

2. **Build New Docker Image**:
   ```bash
   cd /path/to/jellycat-draft-ui
   docker build -t jellycat-draft:v7 .
   ```

3. **Push to Registry**:
   ```bash
   docker tag jellycat-draft:v7 ghcr.io/billy-davies-2/jellycat-draft-ui:v7
   docker push ghcr.io/billy-davies-2/jellycat-draft-ui:v7
   ```

4. **Update Kubernetes Deployment**:
   ```bash
   kubectl set image deployment/jellycat-draft-jellycat-ui \
     ui=ghcr.io/billy-davies-2/jellycat-draft-ui:v7 \
     -n default
   ```

5. **Monitor Deployment**:
   ```bash
   # Watch pod status
   kubectl get pods -n default -l app.kubernetes.io/name=jellycat-ui -w
   
   # Check logs
   kubectl logs -n default -l app.kubernetes.io/name=jellycat-ui --tail=100 -f
   ```

6. **Verify Connection**:
   ```bash
   # Check health endpoint
   kubectl port-forward -n default svc/jellycat-draft 3000:80
   curl http://localhost:3000/api/health
   ```

---

## Quick Reference

### Fixing DNS Timeout
```bash
# 1. Update DATABASE_URL with connect_timeout=60
kubectl patch secret jellycat-draft-auth-secrets -n default --type='json' \
  -p='[{"op": "replace", "path": "/data/DATABASE_URL", "value": "'$(echo -n "YOUR_CONNECTION_STRING_WITH_CONNECT_TIMEOUT" | base64 -w0)'"}]'

# 2. Restart deployment
kubectl rollout restart deployment/jellycat-draft-jellycat-ui -n default

# 3. Watch for successful startup
kubectl logs -n default -l app.kubernetes.io/name=jellycat-ui --tail=50 -f
```

### Checking gRPC
```bash
# Test gRPC endpoint
grpcurl -plaintext localhost:50051 draft.DraftService/GetState
```

### Admin Panel Access
```
URL: https://your-domain.com/admin
Requires: Authentication + "admins" group membership
```

---

## Support

For issues or questions:
1. Check `docs/kubernetes-troubleshooting.md` for common issues
2. Review `docs/kubernetes-cloudnative-pg.md` for CloudNativePG setup
3. See `docs/grpc-nats-architecture.md` for architecture details
4. Consult `docs/admin-panel-guide.md` for admin features

---

## Summary

All issues from the problem statement have been addressed:

1. ✅ **PostgreSQL connection timeout** - Fixed with retry logic and documentation
2. ✅ **gRPC endpoints** - Documented why they're essential and should be kept
3. ✅ **Admin panel** - Verified and fully documented
4. ✅ **TailwindCSS in Docker** - Now automatically compiled during build

**Next Steps**: 
1. Update your DATABASE_URL secret with `connect_timeout=60`
2. Rebuild and deploy the new Docker image
3. Review the new documentation files
4. Test the deployment in your Kubernetes cluster

No code changes are needed for Podman specifically - the fixes apply to all container runtimes including Docker and Podman.
