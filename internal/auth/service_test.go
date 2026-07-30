package auth

import (
	"context"
	"testing"
)

func TestSignupAndLogin(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	service := NewService()

	user, err := service.Signup(context.Background(), SignupRequest{Email: "admin@example.com", Password: "StrongPass123!", Role: "admin"})
	if err != nil {
		t.Fatalf("Signup() error = %v", err)
	}
	if user.Role != "admin" {
		t.Fatalf("Signup() role = %q, want admin", user.Role)
	}

	token, err := service.Login(context.Background(), LoginRequest{Email: "admin@example.com", Password: "StrongPass123!"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if token == "" {
		t.Fatal("Login() returned an empty token")
	}
}

func TestSuperadminBootstrapFromEnv(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("SUPERADMIN_EMAIL", "superadmin@erp.local")
	t.Setenv("SUPERADMIN_PASSWORD", "SuperAdmin123!")

	service := NewService()

	user, err := service.Signup(context.Background(), SignupRequest{Email: "superadmin@erp.local", Password: "SuperAdmin123!"})
	if err != nil {
		t.Fatalf("Signup() error = %v", err)
	}
	if user.Role != "superadmin" {
		t.Fatalf("Signup() role = %q, want superadmin", user.Role)
	}

	token, err := service.Login(context.Background(), LoginRequest{Email: "superadmin@erp.local", Password: "SuperAdmin123!"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if token == "" {
		t.Fatal("Login() returned an empty token")
	}
}
