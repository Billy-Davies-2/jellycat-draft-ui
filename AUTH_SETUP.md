# Authentication Setup Guide

## Overview

The Jellycat Draft microservice supports authentication via **Authentik** (OAuth2/OIDC) for production environments and provides a **mock authentication** system for local development.

## Architecture

- **Production**: Authentik OAuth2/OIDC flow
- **Development**: Automatic mock authentication (no external dependencies)
- **Session Management**: Server-side sessions with secure HTTP-only cookies
- **User Context**: Authenticated user available in request context and templates

---

## Local Development (Mock Authentication)

### Configuration

No configuration needed! Just set `ENVIRONMENT=development`:

```bash
export ENVIRONMENT=development
./jellycat-draft
```

### Mock User Details

The mock authentication automatically creates a test user:

- **ID**: `dev-user-123`
- **Email**: `dev@jellycat.local`
- **Name**: `Dev User`
- **Username**: `devuser`
- **Groups**: `users`, `admins`

### How It Works

1. Navigate to any protected route (e.g., `/start`, `/draft`, `/admin`)
2. You're redirected to `/auth/login`
3. Mock auth automatically creates a session and redirects back
4. Session lasts 24 hours

### Mock Authentication Flow

```
User requests /draft
   ↓
Middleware checks session cookie
   ↓
No session → Redirect to /auth/login
   ↓
Mock auth creates session immediately
   ↓
Sets session_id cookie
   ↓
Redirects to original URL
   ↓
User is authenticated
```

---

## Production (Authentik OAuth2/OIDC)

### Prerequisites

1. **Authentik instance** running and accessible
2. **Application** created in Authentik with:
   - Provider type: OAuth2/OpenID
   - Client type: Confidential
   - Redirect URIs configured
   - Scopes: `openid`, `profile`, `email`

### Authentik Configuration

#### 1. Create Application in Authentik

Navigate to **Applications** → **Create Application**:

- **Name**: `Jellycat Draft`
- **Slug**: `jellycat-draft`
- **Provider**: Create new OAuth2/OpenID Provider

#### 2. Configure OAuth2 Provider

- **Client type**: Confidential
- **Client ID**: Generate or use custom (e.g., `jellycat-draft-client`)
- **Client Secret**: Generate secure secret
- **Redirect URIs**: 
  ```
  http://localhost:3000/auth/callback
  https://draft.yourdomain.com/auth/callback
  ```
- **Scopes**: `openid`, `profile`, `email`
- **Subject mode**: Based on User's UUID
- **Include claims in ID token**: ✓

#### 3. Configure Groups (Optional)

In **Directory** → **Groups**, create:
- `draft-users` - Can access draft
- `draft-admins` - Can access admin panel

Assign groups to users in **Directory** → **Users**.

### Environment Variables

```bash
# Required for production
export ENVIRONMENT=production
export AUTHENTIK_BASE_URL="https://auth.yourdomain.com"
export AUTHENTIK_CLIENT_ID="jellycat-draft-client"
export AUTHENTIK_CLIENT_SECRET="your-secret-here"
export AUTHENTIK_REDIRECT_URL="https://draft.yourdomain.com/auth/callback"

# Optional (defaults shown)
# AUTHENTIK_SCOPES="openid,profile,email"
```

### Production Flow

```
User requests /draft
   ↓
Middleware checks session cookie
   ↓
No session → Redirect to /auth/login
   ↓
Server generates OAuth2 state (CSRF protection)
   ↓
Redirect to Authentik authorize endpoint
   ↓
User logs in at Authentik
   ↓
Authentik redirects to /auth/callback?code=xxx&state=xxx
   ↓
Server validates state
   ↓
Exchange code for access token
   ↓
Fetch user info from Authentik
   ↓
Create server-side session
   ↓
Set session_id cookie
   ↓
Redirect to /draft
   ↓
User is authenticated
```

---

## API Endpoints

### Public Endpoints

- `GET /auth/login` - Initiate OAuth2 login flow
- `GET /auth/callback` - OAuth2 callback handler
- `GET /auth/logout` - Logout and clear session

### Protected Endpoints

All require valid session cookie:

- `GET /start` - Team creation page
- `GET /draft` - Main draft page
- `GET /admin` - Admin panel
- All `/api/*` endpoints

---

## Session Management

### Session Cookie

- **Name**: `session_id`
- **HttpOnly**: `true` (prevents XSS)
- **Secure**: `true` (HTTPS only in production)
- **SameSite**: `Lax` (CSRF protection)
- **Expiration**: Matches token expiry (typically 1 hour)

### Session Storage

Sessions stored in-memory on the server:

```go
type Session struct {
    ID        string
    User      *User
    Token     *oauth2.Token
    CreatedAt time.Time
    ExpiresAt time.Time
}
```

**Note**: For multi-instance deployments, consider adding Redis or similar for shared session storage.

### Session Validation

On each request to protected endpoints:
1. Extract `session_id` cookie
2. Lookup session in memory
3. Check expiration
4. Add user to request context
5. Proceed or redirect to login

---

## User Data

### User Object

```go
type User struct {
    ID       string    // Authentik user UUID
    Email    string    // User email
    Name     string    // Full name
    Username string    // Username
    Groups   []string  // Group memberships
}
```

### Accessing User in Code

```go
// In handlers
user := auth.GetUser(r)
if user == nil {
    // Not authenticated
}

log.Printf("User %s (%s) accessed resource", user.Name, user.Email)
```

### Accessing User in Templates

```html
{{ if .User }}
    <p>Welcome, {{ .User.Name }}!</p>
    <p>Email: {{ .User.Email }}</p>
{{ end }}
```

---

## Security Considerations

### CSRF Protection

- OAuth2 state parameter used for CSRF protection
- State stored in secure HTTP-only cookie
- State validated on callback

### Session Security

- HTTP-only cookies prevent XSS attacks
- Secure flag enforces HTTPS in production
- SameSite=Lax prevents CSRF
- Short-lived sessions (1 hour default)

### Token Handling

- Access tokens never exposed to client
- Tokens stored server-side in session
- Refresh tokens can be implemented for long-lived sessions

---

## Customization

### Custom Scopes

```bash
export AUTHENTIK_SCOPES="openid,profile,email,groups"
```

### Group-Based Access Control

Modify middleware to check user groups:

```go
func (a *AuthentikAuth) AdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return a.Middleware(func(w http.ResponseWriter, r *http.Request) {
        user := auth.GetUser(r)
        
        // Check if user is admin
        isAdmin := false
        for _, group := range user.Groups {
            if group == "draft-admins" {
                isAdmin = true
                break
            }
        }
        
        if !isAdmin {
            http.Error(w, "Forbidden", http.StatusForbidden)
            return
        }
        
        next(w, r)
    })
}
```

### Session Expiration

Customize in `internal/auth/authentik.go`:

```go
session := &Session{
    ID:        sessionID,
    User:      user,
    Token:     token,
    CreatedAt: time.Now(),
    ExpiresAt: time.Now().Add(24 * time.Hour), // Custom expiration
}
```

---

## Troubleshooting

### "Missing state cookie" Error

**Cause**: Browser cookies disabled or not being set properly

**Solution**: 
- Ensure cookies are enabled
- Check `Secure` flag matches protocol (HTTP vs HTTPS)
- Verify `SameSite` policy compatibility

### "Invalid state parameter" Error

**Cause**: CSRF validation failed

**Solution**:
- Check clock synchronization between client and server
- Ensure cookies persist between requests
- Verify redirect URL matches configured URL

### "Failed to exchange token" Error

**Cause**: OAuth2 code exchange failed

**Solution**:
- Verify `AUTHENTIK_CLIENT_ID` and `AUTHENTIK_CLIENT_SECRET`
- Check Authentik provider configuration
- Ensure redirect URI matches exactly
- Check Authentik logs for details

### "Failed to get user info" Error

**Cause**: Cannot fetch user details from Authentik

**Solution**:
- Verify access token is valid
- Check Authentik userinfo endpoint is accessible
- Ensure required scopes (`profile`, `email`) are configured

### Session Expires Immediately

**Cause**: Token expiry too short

**Solution**:
- Check Authentik token lifetime settings
- Implement token refresh in session middleware
- Increase session expiration if appropriate

---

## Alpine.js Integration

The frontend uses Alpine.js for enriched client-side interactions while maintaining the server-rendered architecture.

### Alpine.js Features

1. **Reactive State Management**
   - Client-side filtering and search
   - Real-time UI updates without page reloads
   - Local state for UI elements (dropdowns, modals, etc.)

2. **User Menu Example**
   ```html
   <div x-data="{ showMenu: false }">
       <button @click="showMenu = !showMenu">
           {{ .User.Username }}
       </button>
       <div x-show="showMenu" @click.away="showMenu = false">
           <a href="/auth/logout">Logout</a>
       </div>
   </div>
   ```

3. **Draft Page Enhancements**
   - Real-time search/filter of available players
   - Position-based filtering
   - Notifications for draft events
   - No full page reload required

### Example: Player Filtering

```javascript
function draftApp() {
    return {
        search: '',
        filterPosition: '',
        
        filterPlayers() {
            const players = document.querySelectorAll('#players-grid > div');
            players.forEach(player => {
                const name = player.textContent.toLowerCase();
                const matchesSearch = this.search === '' || 
                                     name.includes(this.search.toLowerCase());
                const position = /* extract from player */;
                const matchesPosition = this.filterPosition === '' || 
                                       position === this.filterPosition;
                
                player.style.display = (matchesSearch && matchesPosition) ? '' : 'none';
            });
        }
    };
}
```

Alpine.js CDN is included in base template with SRI hash for security.

---

## Testing Authentication

### Test Mock Auth

```bash
# Start server
ENVIRONMENT=development ./jellycat-draft

# Open browser
open http://localhost:3000/start

# You'll be auto-logged in as dev user
```

### Test Authentik Auth

```bash
# Configure environment
export ENVIRONMENT=production
export AUTHENTIK_BASE_URL="https://auth.test.local"
export AUTHENTIK_CLIENT_ID="test-client"
export AUTHENTIK_CLIENT_SECRET="test-secret"
export AUTHENTIK_REDIRECT_URL="http://localhost:3000/auth/callback"

# Start server
./jellycat-draft

# Open browser and test login flow
open http://localhost:3000/start
```

### Test Session Persistence

1. Login successfully
2. Close browser tab (not entire browser)
3. Reopen tab to `/draft`
4. Should remain authenticated

### Test Session Expiration

1. Login successfully
2. Wait for session expiry (or manually delete session on server)
3. Try to access `/draft`
4. Should redirect to login

---

## Production Deployment

### Docker with Authentik

```bash
docker run -p 3000:3000 -p 50051:50051 \
  -e ENVIRONMENT=production \
  -e AUTHENTIK_BASE_URL="https://auth.yourdomain.com" \
  -e AUTHENTIK_CLIENT_ID="jellycat-draft-client" \
  -e AUTHENTIK_CLIENT_SECRET="your-secret" \
  -e AUTHENTIK_REDIRECT_URL="https://draft.yourdomain.com/auth/callback" \
  -e DATABASE_URL="postgres://..." \
  -e NATS_URL="nats://..." \
  -e CLICKHOUSE_ADDR="..." \
  jellycat-draft
```

### Kubernetes ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: jellycat-draft-config
data:
  ENVIRONMENT: "production"
  AUTHENTIK_BASE_URL: "https://auth.yourdomain.com"
  AUTHENTIK_REDIRECT_URL: "https://draft.yourdomain.com/auth/callback"
---
apiVersion: v1
kind: Secret
metadata:
  name: jellycat-draft-secrets
type: Opaque
stringData:
  AUTHENTIK_CLIENT_ID: "jellycat-draft-client"
  AUTHENTIK_CLIENT_SECRET: "your-secret-here"
```

---

## References

- [Authentik Documentation](https://goauthentik.io/docs/)
- [OAuth 2.0 RFC](https://tools.ietf.org/html/rfc6749)
- [OpenID Connect](https://openid.net/connect/)
- [Alpine.js Documentation](https://alpinejs.dev/)
- [golang.org/x/oauth2](https://pkg.go.dev/golang.org/x/oauth2)
