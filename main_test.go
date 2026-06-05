package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/auth"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/models"
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

func TestBuildFeaturedProspectsSpreadsCategories(t *testing.T) {
	players := []models.Player{
		{ID: "space-1", Name: "Space One", Team: "Space", Position: "CC", Image: "/images/space-1.png"},
		{ID: "space-2", Name: "Space Two", Team: "Space", Position: "SS", Image: "/images/space-2.png"},
		{ID: "ocean-1", Name: "Ocean One", Team: "Ocean", Position: "HH", Image: "/images/ocean-1.png"},
		{ID: "garden-1", Name: "Garden One", Team: "Garden", Position: "DD", Image: "/images/garden-1.png"},
	}

	prospects := buildFeaturedProspects(players, "ROOM|standard")
	if len(prospects) != 3 {
		t.Fatalf("featured prospects length = %d, want 3", len(prospects))
	}

	categories := map[string]bool{}
	for _, prospect := range prospects {
		categories[prospect.Category] = true
		if prospect.Image == "" || prospect.Name == "" || prospect.Label == "" || prospect.FrameClass == "" {
			t.Fatalf("featured prospect missing display fields: %+v", prospect)
		}
	}

	if len(categories) != 3 {
		t.Fatalf("featured prospects should prefer distinct categories, got %v", categories)
	}
}

func TestBuildFeaturedProspectsIgnoresDraftedPlayers(t *testing.T) {
	players := []models.Player{
		{ID: "drafted", Name: "Drafted", Team: "Space", Drafted: true, Image: "/images/drafted.png"},
		{ID: "available", Name: "Available", Team: "Ocean", Image: "/images/available.png"},
	}

	prospects := buildFeaturedProspects(players, "ROOM|standard")
	if len(prospects) != 1 {
		t.Fatalf("featured prospects length = %d, want 1", len(prospects))
	}
	if prospects[0].Name != "Available" {
		t.Fatalf("featured prospect = %q, want available player", prospects[0].Name)
	}
}

func requestWithUser(request *http.Request, user *auth.User) *http.Request {
	return request.WithContext(context.WithValue(request.Context(), "user", user)) //nolint:staticcheck
}
