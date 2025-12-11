package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// AuthentikConfig holds the configuration for Authentik OAuth2/OIDC
type AuthentikConfig struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// User represents an authenticated user
type User struct {
	ID       string
	Email    string
	Name     string
	Username string
	Groups   []string
}

// AuthentikAuth manages authentication with Authentik
type AuthentikAuth struct {
	config       *AuthentikConfig
	oauth2Config *oauth2.Config
	sessions     map[string]*Session
	sessionMu    sync.RWMutex
}

// Session represents a user session
type Session struct {
	ID        string
	User      *User
	Token     *oauth2.Token
	CreatedAt time.Time
	ExpiresAt time.Time
}

// NewAuthentikAuth creates a new Authentik authentication handler
func NewAuthentikAuth(config *AuthentikConfig) *AuthentikAuth {
	if len(config.Scopes) == 0 {
		config.Scopes = []string{"openid", "profile", "email"}
	}

	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       config.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/application/o/authorize/", config.BaseURL),
			TokenURL: fmt.Sprintf("%s/application/o/token/", config.BaseURL),
		},
	}

	return &AuthentikAuth{
		config:       config,
		oauth2Config: oauth2Config,
		sessions:     make(map[string]*Session),
	}
}

// LoginHandler initiates the OAuth2 login flow
func (a *AuthentikAuth) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Generate state for CSRF protection
	state := generateState()

	// Store state in cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300, // 5 minutes
	})

	// Redirect to Authentik
	authURL := a.oauth2Config.AuthCodeURL(state)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// CallbackHandler handles the OAuth2 callback from Authentik
func (a *AuthentikAuth) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	// Verify state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "Missing state cookie", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state != stateCookie.Value {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	code := r.URL.Query().Get("code")
	token, err := a.oauth2Config.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get user info
	user, err := a.getUserInfo(token)
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create session
	sessionID := generateSessionID()
	session := &Session{
		ID:        sessionID,
		User:      user,
		Token:     token,
		CreatedAt: time.Now(),
		ExpiresAt: token.Expiry,
	}

	a.sessionMu.Lock()
	a.sessions[sessionID] = session
	a.sessionMu.Unlock()

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  token.Expiry,
	})

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// Redirect to app
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// LogoutHandler handles user logout
func (a *AuthentikAuth) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Get session cookie
	cookie, err := r.Cookie("session_id")
	if err == nil {
		// Delete session
		a.sessionMu.Lock()
		delete(a.sessions, cookie.Value)
		a.sessionMu.Unlock()
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "session_id",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// Redirect to Authentik logout
	logoutURL := fmt.Sprintf("%s/application/o/jellycat-draft/end-session/", a.config.BaseURL)
	http.Redirect(w, r, logoutURL, http.StatusSeeOther)
}

// Middleware protects routes requiring authentication
func (a *AuthentikAuth) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get session cookie
		cookie, err := r.Cookie("session_id")
		if err != nil {
			// Not authenticated, redirect to login
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		// Get session
		a.sessionMu.RLock()
		session, exists := a.sessions[cookie.Value]
		a.sessionMu.RUnlock()

		if !exists || time.Now().After(session.ExpiresAt) {
			// Session expired or invalid
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), "user", session.User)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// GetUser retrieves the authenticated user from the request context
func GetUser(r *http.Request) *User {
	user, ok := r.Context().Value("user").(*User)
	if !ok {
		return nil
	}
	return user
}

// IsAdmin checks if the user has admin privileges
func IsAdmin(user *User) bool {
	if user == nil {
		return false
	}
	for _, group := range user.Groups {
		if group == "admins" {
			return true
		}
	}
	return false
}

// getUserInfo fetches user information from Authentik
func (a *AuthentikAuth) getUserInfo(token *oauth2.Token) (*User, error) {
	userInfoURL := fmt.Sprintf("%s/application/o/userinfo/", a.config.BaseURL)

	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: %s - %s", resp.Status, string(body))
	}

	var userInfo struct {
		Sub               string   `json:"sub"`
		Email             string   `json:"email"`
		Name              string   `json:"name"`
		PreferredUsername string   `json:"preferred_username"`
		Groups            []string `json:"groups"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &User{
		ID:       userInfo.Sub,
		Email:    userInfo.Email,
		Name:     userInfo.Name,
		Username: userInfo.PreferredUsername,
		Groups:   userInfo.Groups,
	}, nil
}

// generateState generates a random state string for CSRF protection
func generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// generateSessionID generates a random session ID
func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// MockAuth provides a mock authentication for local development
type MockAuth struct {
	sessions  map[string]*Session
	sessionMu sync.RWMutex
}

// NewMockAuth creates a new mock authentication handler
func NewMockAuth() *MockAuth {
	return &MockAuth{
		sessions: make(map[string]*Session),
	}
}

// LoginHandler for mock auth - auto-creates a session
func (m *MockAuth) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Auto-authenticate as test user
	sessionID := generateSessionID()
	session := &Session{
		ID: sessionID,
		User: &User{
			ID:       "dev-user-123",
			Email:    "dev@jellycat.local",
			Name:     "Dev User",
			Username: "devuser",
			Groups:   []string{"users", "admins"},
		},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	m.sessionMu.Lock()
	m.sessions[sessionID] = session
	m.sessionMu.Unlock()

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Expires:  session.ExpiresAt,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// CallbackHandler is not needed for mock auth
func (m *MockAuth) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// LogoutHandler for mock auth
func (m *MockAuth) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil {
		m.sessionMu.Lock()
		delete(m.sessions, cookie.Value)
		m.sessionMu.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "session_id",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	http.Redirect(w, r, "/start", http.StatusSeeOther)
}

// Middleware for mock auth
func (m *MockAuth) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		m.sessionMu.RLock()
		session, exists := m.sessions[cookie.Value]
		m.sessionMu.RUnlock()

		if !exists || time.Now().After(session.ExpiresAt) {
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), "user", session.User)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// AuthProvider is a common interface for authentication providers
type AuthProvider interface {
	LoginHandler(w http.ResponseWriter, r *http.Request)
	CallbackHandler(w http.ResponseWriter, r *http.Request)
	LogoutHandler(w http.ResponseWriter, r *http.Request)
	Middleware(next http.HandlerFunc) http.HandlerFunc
}
