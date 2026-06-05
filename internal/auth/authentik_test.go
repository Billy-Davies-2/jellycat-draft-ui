package auth

import "testing"

func TestIsAdminDefaultsToAdminsGroup(t *testing.T) {
	t.Setenv("AUTH_ADMIN_CLAIM", "")
	t.Setenv("AUTH_ADMIN_VALUE", "")

	if !IsAdmin(&User{Groups: []string{"users", "admins"}}) {
		t.Fatal("expected user in admins group to be an admin")
	}
	if IsAdmin(&User{Groups: []string{"users"}}) {
		t.Fatal("expected user outside admins group to be denied")
	}
}

func TestIsAdminCanUseEmailClaim(t *testing.T) {
	t.Setenv("AUTH_ADMIN_CLAIM", "email")
	t.Setenv("AUTH_ADMIN_VALUE", "billy.davies.10@icloud.com")

	if !IsAdmin(&User{Email: "Billy.Davies.10@icloud.com", Groups: []string{"homelab-admins"}}) {
		t.Fatal("expected configured email to be an admin")
	}
	if IsAdmin(&User{Email: "akadmin@example.com", Groups: []string{"authentik Admins"}}) {
		t.Fatal("expected other users to be denied")
	}
}
