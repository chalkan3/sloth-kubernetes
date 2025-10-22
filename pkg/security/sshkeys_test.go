package security

import (
	"strings"
	"testing"
)

func TestGenerateLocalKeyPair(t *testing.T) {
	privateKey, publicKey, err := GenerateLocalKeyPair()

	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	// Validate private key
	if privateKey == "" {
		t.Error("private key is empty")
	}

	if !strings.Contains(privateKey, "BEGIN RSA PRIVATE KEY") {
		t.Error("private key does not contain RSA PRIVATE KEY header")
	}

	if !strings.Contains(privateKey, "END RSA PRIVATE KEY") {
		t.Error("private key does not contain RSA PRIVATE KEY footer")
	}

	// Validate public key
	if publicKey == "" {
		t.Error("public key is empty")
	}

	if !strings.HasPrefix(publicKey, "ssh-rsa ") {
		t.Errorf("public key does not start with 'ssh-rsa ', got: %s", publicKey[:20])
	}

	// Validate key length (4096 bits should produce a long key)
	if len(privateKey) < 1000 {
		t.Errorf("private key seems too short: %d bytes", len(privateKey))
	}

	if len(publicKey) < 200 {
		t.Errorf("public key seems too short: %d bytes", len(publicKey))
	}
}

func TestGenerateLocalKeyPair_Uniqueness(t *testing.T) {
	// Generate two key pairs
	priv1, pub1, err1 := GenerateLocalKeyPair()
	if err1 != nil {
		t.Fatalf("failed to generate first key pair: %v", err1)
	}

	priv2, pub2, err2 := GenerateLocalKeyPair()
	if err2 != nil {
		t.Fatalf("failed to generate second key pair: %v", err2)
	}

	// Ensure they are different
	if priv1 == priv2 {
		t.Error("generated private keys are identical (should be unique)")
	}

	if pub1 == pub2 {
		t.Error("generated public keys are identical (should be unique)")
	}
}

func TestGenerateLocalKeyPair_Format(t *testing.T) {
	privateKey, publicKey, err := GenerateLocalKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	// Check private key PEM format
	lines := strings.Split(privateKey, "\n")
	if len(lines) < 3 {
		t.Error("private key has too few lines")
	}

	if lines[0] != "-----BEGIN RSA PRIVATE KEY-----" {
		t.Errorf("private key has incorrect header: %s", lines[0])
	}

	// Find the footer
	hasFooter := false
	for _, line := range lines {
		if line == "-----END RSA PRIVATE KEY-----" {
			hasFooter = true
			break
		}
	}

	if !hasFooter {
		t.Error("private key missing footer")
	}

	// Check public key format (should be single line)
	pubLines := strings.Split(strings.TrimSpace(publicKey), "\n")
	if len(pubLines) != 1 {
		t.Errorf("public key should be single line, got %d lines", len(pubLines))
	}

	// Public key should have 2 or 3 space-separated parts
	parts := strings.Fields(publicKey)
	if len(parts) < 2 {
		t.Errorf("public key should have at least 2 parts, got %d", len(parts))
	}

	if parts[0] != "ssh-rsa" {
		t.Errorf("public key type should be 'ssh-rsa', got '%s'", parts[0])
	}
}

func TestGenerateLocalKeyPair_MultipleGenerations(t *testing.T) {
	// Generate multiple key pairs to ensure consistency
	const iterations = 5

	for i := 0; i < iterations; i++ {
		privateKey, publicKey, err := GenerateLocalKeyPair()

		if err != nil {
			t.Fatalf("iteration %d: failed to generate key pair: %v", i, err)
		}

		if privateKey == "" || publicKey == "" {
			t.Fatalf("iteration %d: generated empty key", i)
		}

		// Basic format validation
		if !strings.Contains(privateKey, "BEGIN RSA PRIVATE KEY") {
			t.Errorf("iteration %d: invalid private key format", i)
		}

		if !strings.HasPrefix(publicKey, "ssh-rsa ") {
			t.Errorf("iteration %d: invalid public key format", i)
		}
	}
}

func TestGenerateLocalKeyPair_KeySize(t *testing.T) {
	privateKey, _, err := GenerateLocalKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	// 4096-bit RSA key should produce a private key of at least 3000 bytes
	// (PEM encoding + header/footer)
	if len(privateKey) < 3000 {
		t.Errorf("private key is too small for 4096-bit RSA: %d bytes", len(privateKey))
	}

	// Should not be unreasonably large either
	if len(privateKey) > 10000 {
		t.Errorf("private key is suspiciously large: %d bytes", len(privateKey))
	}
}

func TestGenerateLocalKeyPair_Base64Encoding(t *testing.T) {
	_, publicKey, err := GenerateLocalKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	// SSH public keys should have base64-encoded key data
	parts := strings.Fields(publicKey)
	if len(parts) < 2 {
		t.Fatal("public key should have at least 2 parts")
	}

	keyData := parts[1]

	// Base64 characters are A-Z, a-z, 0-9, +, /, =
	for _, ch := range keyData {
		if !isBase64Char(ch) {
			t.Errorf("public key contains invalid base64 character: %c", ch)
		}
	}

	// Should be reasonably long for 4096-bit key
	if len(keyData) < 500 {
		t.Errorf("public key data seems too short: %d characters", len(keyData))
	}
}

// Helper function to check if character is valid base64
func isBase64Char(ch rune) bool {
	return (ch >= 'A' && ch <= 'Z') ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '+' || ch == '/' || ch == '='
}

func TestGenerateLocalKeyPair_NoWhitespaceInKeyData(t *testing.T) {
	_, publicKey, err := GenerateLocalKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	parts := strings.Fields(publicKey)
	if len(parts) < 2 {
		t.Fatal("public key should have at least 2 parts")
	}

	// The key data (second part) should not contain whitespace
	keyData := parts[1]
	if strings.ContainsAny(keyData, " \t\n\r") {
		t.Error("public key data contains whitespace")
	}
}
