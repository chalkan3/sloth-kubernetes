package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chalkan3/sloth-kubernetes/pkg/secrets"
	"github.com/pulumi/pulumi-tls/sdk/v4/go/tls"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"golang.org/x/crypto/ssh"
)

// SSHKeyManager manages SSH key generation and distribution
type SSHKeyManager struct {
	ctx        *pulumi.Context
	privateKey pulumi.StringOutput
	publicKey  pulumi.StringOutput
	keyPair    *tls.PrivateKey
}

// NewSSHKeyManager creates a new SSH key manager
func NewSSHKeyManager(ctx *pulumi.Context) *SSHKeyManager {
	return &SSHKeyManager{
		ctx: ctx,
	}
}

// GenerateKeyPair generates a new SSH key pair using Pulumi TLS provider
func (s *SSHKeyManager) GenerateKeyPair() error {
	// Generate SSH key pair using Pulumi TLS provider
	keyPair, err := tls.NewPrivateKey(s.ctx, fmt.Sprintf("%s-ssh-key", s.ctx.Stack()), &tls.PrivateKeyArgs{
		Algorithm: pulumi.String("RSA"),
		RsaBits:   pulumi.Int(4096),
	})
	if err != nil {
		return fmt.Errorf("failed to generate SSH key pair: %w", err)
	}

	s.keyPair = keyPair
	s.privateKey = keyPair.PrivateKeyPem

	// Clean the public key to ensure it's in the correct OpenSSH format
	// Linode is very strict about the format: "ssh-rsa AAAAB3... [optional-comment]"
	s.publicKey = keyPair.PublicKeyOpenssh.ApplyT(func(key string) string {
		// Remove any newlines, carriage returns, and extra whitespace
		cleaned := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(key, "\n", ""), "\r", ""))

		// Ensure the key has exactly 2 or 3 parts (type, key-data, optional-comment)
		parts := strings.Fields(cleaned)
		if len(parts) >= 2 {
			// Return just type + key-data (no comment) to avoid any issues
			return parts[0] + " " + parts[1]
		}
		return cleaned
	}).(pulumi.StringOutput)

	// Export keys (all encrypted with passphrase)
	secretExporter := secrets.NewSecretExporter(s.ctx)
	secretExporter.Export("ssh_private_key", s.privateKey)
	secretExporter.Export("ssh_public_key", s.publicKey)

	// Save private key to local file for SSH access
	s.savePrivateKey()

	return nil
}

// GetPublicKey returns the public key in OpenSSH format
func (s *SSHKeyManager) GetPublicKey() pulumi.StringOutput {
	if s.publicKey == pulumi.String("").ToStringOutput() {
		// Generate key if not already done
		s.GenerateKeyPair()
	}
	return s.publicKey
}

// GetPrivateKey returns the private key in PEM format
func (s *SSHKeyManager) GetPrivateKey() pulumi.StringOutput {
	if s.privateKey == pulumi.String("").ToStringOutput() {
		// Generate key if not already done
		s.GenerateKeyPair()
	}
	return s.privateKey
}

// GetPublicKeyString returns the public key as a string for use in resources
func (s *SSHKeyManager) GetPublicKeyString() pulumi.StringInput {
	return s.GetPublicKey()
}

// GetPrivateKeyString returns the private key as a string for use in resources
func (s *SSHKeyManager) GetPrivateKeyString() pulumi.StringInput {
	return s.GetPrivateKey()
}

// savePrivateKey saves the private key to a local file
func (s *SSHKeyManager) savePrivateKey() {
	s.privateKey.ApplyT(func(key string) string {
		// Create .ssh directory if it doesn't exist
		sshDir := filepath.Join(os.Getenv("HOME"), ".ssh", "kubernetes-clusters")
		os.MkdirAll(sshDir, 0700)

		// Save private key
		keyPath := filepath.Join(sshDir, fmt.Sprintf("%s.pem", s.ctx.Stack()))
		err := os.WriteFile(keyPath, []byte(key), 0600)
		if err != nil {
			s.ctx.Log.Warn("Failed to save private key to file", nil)
		} else {
			s.ctx.Log.Info("SSH private key saved", nil)
			secretExporter := secrets.NewSecretExporter(s.ctx)
			secretExporter.ExportString("ssh_private_key_path", keyPath)
		}
		return key
	})
}

// GenerateLocalKeyPair generates a key pair locally (fallback method)
func GenerateLocalKeyPair() (privateKey string, publicKey string, err error) {
	// Generate RSA key pair
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Generate private key in PEM format
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)

	// Generate public key in OpenSSH format
	pub, err := ssh.NewPublicKey(&key.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate SSH public key: %w", err)
	}
	publicKeyBytes := ssh.MarshalAuthorizedKey(pub)

	return string(privateKeyBytes), string(publicKeyBytes), nil
}

// ExportSSHAccess exports SSH access information (encrypted)
func (s *SSHKeyManager) ExportSSHAccess(nodes []string) {
	secretExporter := secrets.NewSecretExporter(s.ctx)
	secretExporter.ExportMap("ssh_access_info", pulumi.Map{
		"private_key_path": pulumi.String(fmt.Sprintf("~/.ssh/kubernetes-clusters/%s.pem", s.ctx.Stack())),
		"nodes":            pulumi.ToStringArray(nodes),
		"example_command":  pulumi.String(fmt.Sprintf("ssh -i ~/.ssh/kubernetes-clusters/%s.pem root@10.8.0.11", s.ctx.Stack())),
	})
}
