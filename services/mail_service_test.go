package services

import (
	"backend/models"
	"os"
	"testing"
)

func TestGetSMTPConfig(t *testing.T) {
	os.Setenv("SMTP_HOST", "smtp.test.com")
	os.Setenv("SMTP_PORT", "587")
	os.Setenv("SMTP_USER", "tester@test.com")
	os.Setenv("SMTP_PASSWORD", "secret")
	os.Setenv("SMTP_FROM", "noreply@test.com")
	os.Setenv("SUPER_ADMIN_EMAIL", "admin@test.com")

	cfg := GetSMTPConfig()
	if cfg.Host != "smtp.test.com" {
		t.Errorf("Expected smtp.test.com, got %s", cfg.Host)
	}
	if cfg.Port != "587" {
		t.Errorf("Expected 587, got %s", cfg.Port)
	}
	if cfg.Username != "tester@test.com" {
		t.Errorf("Expected tester@test.com, got %s", cfg.Username)
	}
	if cfg.From != "noreply@test.com" {
		t.Errorf("Expected noreply@test.com, got %s", cfg.From)
	}

	superAdmin := GetSuperAdminEmail()
	if superAdmin != "admin@test.com" {
		t.Errorf("Expected admin@test.com, got %s", superAdmin)
	}
}

func TestSendNewUserAlertGracefulWhenEmpty(t *testing.T) {
	os.Setenv("SMTP_HOST", "")
	os.Setenv("SMTP_USER", "")
	os.Setenv("SMTP_PASSWORD", "")

	user := models.User{
		Name:  "Test User",
		Email: "newuser@example.com",
		Role:  models.RoleCustomer,
	}

	// Should not panic or crash even if SMTP credentials are missing
	SendNewUserAlert("Registration", user, "127.0.0.1", "Go-Test-Agent")
}
