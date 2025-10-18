package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

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
	s.publicKey = keyPair.PublicKeyOpenssh

	// Export keys
	s.ctx.Export("ssh_private_key", pulumi.ToSecret(s.privateKey))
	s.ctx.Export("ssh_public_key", s.publicKey)

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
			s.ctx.Export("ssh_private_key_path", pulumi.String(keyPath))
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

// ExportSSHAccess exports SSH access information
func (s *SSHKeyManager) ExportSSHAccess(nodes []string) {
	s.ctx.Export("ssh_access_info", pulumi.Map{
		"private_key_path": pulumi.String(fmt.Sprintf("~/.ssh/kubernetes-clusters/%s.pem", s.ctx.Stack())),
		"nodes": pulumi.ToStringArray(nodes),
		"example_command": pulumi.String(fmt.Sprintf("ssh -i ~/.ssh/kubernetes-clusters/%s.pem root@10.8.0.11", s.ctx.Stack())),
	})
}