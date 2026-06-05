package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/auth"
)

func TestRequireAdminAPIRequiresLogin(t *testing.T) {
	called := false
	handler := requireAdminAPI(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/players/update", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if called {
		t.Fatal("handler should not be called without a logged-in user")
	}
}

func TestRequireAdminAPIRequiresAdmin(t *testing.T) {
	t.Setenv("AUTH_ADMIN_CLAIM", "email")
	t.Setenv("AUTH_ADMIN_VALUE", "billy.davies.10@icloud.com")

	called := false
	handler := requireAdminAPI(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	request := requestWithUser(httptest.NewRequest(http.MethodPost, "/api/players/update", nil), &auth.User{Email: "not-admin@example.com"})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if called {
		t.Fatal("handler should not be called for a non-admin user")
	}
}

func TestRequireAdminAPIAllowsAdmin(t *testing.T) {
	t.Setenv("AUTH_ADMIN_CLAIM", "email")
	t.Setenv("AUTH_ADMIN_VALUE", "billy.davies.10@icloud.com")

	called := false
	handler := requireAdminAPI(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	})

	request := requestWithUser(httptest.NewRequest(http.MethodPost, "/api/players/update", nil), &auth.User{Email: "billy.davies.10@icloud.com"})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
	if !called {
		t.Fatal("handler should be called for the configured admin user")
	}
}

func requestWithUser(request *http.Request, user *auth.User) *http.Request {
	return request.WithContext(context.WithValue(request.Context(), "user", user)) //nolint:staticcheck
}
