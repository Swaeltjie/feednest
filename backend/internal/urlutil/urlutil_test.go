package urlutil

import (
	"testing"
)

func TestIsSafeURL_BlocksLocalhost(t *testing.T) {
	AllowPrivate = false
	tests := []string{
		"http://localhost/evil",
		"http://LOCALHOST:8080/path",
		"http://sub.localhost/path",
	}
	for _, url := range tests {
		if err := IsSafeURL(url); err == nil {
			t.Errorf("expected error for %q, got nil", url)
		}
	}
}

func TestIsSafeURL_BlocksBadSchemes(t *testing.T) {
	AllowPrivate = false
	tests := []string{
		"ftp://example.com/file",
		"javascript:alert(1)",
		"file:///etc/passwd",
	}
	for _, url := range tests {
		if err := IsSafeURL(url); err == nil {
			t.Errorf("expected error for %q, got nil", url)
		}
	}
}

func TestIsSafeURL_AllowPrivateBypass(t *testing.T) {
	AllowPrivate = true
	defer func() { AllowPrivate = false }()

	if err := IsSafeURL("http://localhost:8080"); err != nil {
		t.Errorf("expected nil with AllowPrivate=true, got %v", err)
	}
}

func TestIsSafeURL_EmptyHostname(t *testing.T) {
	AllowPrivate = false
	if err := IsSafeURL("http:///path"); err == nil {
		t.Error("expected error for empty hostname")
	}
}

func TestIsSafeURL_AllowsPublicHTTPS(t *testing.T) {
	AllowPrivate = true
	defer func() { AllowPrivate = false }()

	if err := IsSafeURL("https://example.com"); err != nil {
		t.Errorf("expected nil for public URL, got %v", err)
	}
}
