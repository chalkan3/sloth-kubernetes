package tailscale

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HeadscaleConfig holds configuration for the Headscale API
type HeadscaleConfig struct {
	APIURL    string // e.g., "https://headscale.example.com"
	APIKey    string // Admin API key
	Namespace string // Default namespace for operations
}

// HeadscaleManager manages Headscale coordination server
type HeadscaleManager struct {
	config     HeadscaleConfig
	httpClient *http.Client
}

// NewHeadscaleManager creates a new Headscale manager
func NewHeadscaleManager(config HeadscaleConfig) *HeadscaleManager {
	return &HeadscaleManager{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AuthKeyOptions contains options for creating pre-auth keys
type AuthKeyOptions struct {
	Reusable   bool          // Can the key be used multiple times
	Ephemeral  bool          // Are nodes ephemeral (removed when offline)
	Expiration time.Duration // Key expiration duration
	Tags       []string      // ACL tags to apply
}

// HeadscaleNode represents a node in Headscale
type HeadscaleNode struct {
	ID             string    `json:"id"`
	MachineKey     string    `json:"machineKey"`
	NodeKey        string    `json:"nodeKey"`
	Name           string    `json:"name"`
	GivenName      string    `json:"givenName"`
	IPAddresses    []string  `json:"ipAddresses"`
	Namespace      string    `json:"namespace"`
	LastSeen       time.Time `json:"lastSeen"`
	LastSuccessfulUpdate time.Time `json:"lastSuccessfulUpdate"`
	Expiry         time.Time `json:"expiry"`
	Online         bool      `json:"online"`
	RegisterMethod string    `json:"registerMethod"`
	ForcedTags     []string  `json:"forcedTags"`
	ValidTags      []string  `json:"validTags"`
	InvalidTags    []string  `json:"invalidTags"`
}

// HeadscalePreAuthKey represents a pre-auth key
type HeadscalePreAuthKey struct {
	ID         string    `json:"id"`
	Key        string    `json:"key"`
	Reusable   bool      `json:"reusable"`
	Ephemeral  bool      `json:"ephemeral"`
	Used       bool      `json:"used"`
	Expiration time.Time `json:"expiration"`
	CreatedAt  time.Time `json:"createdAt"`
	ACLTags    []string  `json:"aclTags"`
}

// HeadscaleNamespace represents a namespace/user in Headscale
type HeadscaleNamespace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// API request/response structures

type createPreAuthKeyRequest struct {
	User       string   `json:"user"`
	Reusable   bool     `json:"reusable"`
	Ephemeral  bool     `json:"ephemeral"`
	Expiration string   `json:"expiration"`
	AclTags    []string `json:"aclTags,omitempty"`
}

type createPreAuthKeyResponse struct {
	PreAuthKey HeadscalePreAuthKey `json:"preAuthKey"`
}

type listNodesResponse struct {
	Nodes []HeadscaleNode `json:"nodes"`
}

type getNodeResponse struct {
	Node HeadscaleNode `json:"node"`
}

type listPreAuthKeysResponse struct {
	PreAuthKeys []HeadscalePreAuthKey `json:"preAuthKeys"`
}

type createNamespaceRequest struct {
	Name string `json:"name"`
}

type createNamespaceResponse struct {
	User HeadscaleNamespace `json:"user"`
}

type listNamespacesResponse struct {
	Users []HeadscaleNamespace `json:"users"`
}

// CreateAuthKey creates a new pre-auth key
// Note: Headscale v0.26+ requires numeric user ID instead of name
func (h *HeadscaleManager) CreateAuthKey(ctx context.Context, opts AuthKeyOptions) (string, error) {
	expiration := time.Now().Add(opts.Expiration)
	if opts.Expiration == 0 {
		expiration = time.Now().Add(24 * time.Hour) // Default 24h
	}

	// Get user ID from namespace name (required for Headscale v0.26+)
	userID, err := h.getUserIDByName(ctx, h.config.Namespace)
	if err != nil {
		return "", fmt.Errorf("failed to resolve user ID for namespace '%s': %w", h.config.Namespace, err)
	}

	req := createPreAuthKeyRequest{
		User:       userID,
		Reusable:   opts.Reusable,
		Ephemeral:  opts.Ephemeral,
		Expiration: expiration.Format(time.RFC3339),
		AclTags:    opts.Tags,
	}

	var resp createPreAuthKeyResponse
	if err := h.apiCall(ctx, "POST", "/api/v1/preauthkey", req, &resp); err != nil {
		return "", fmt.Errorf("failed to create auth key: %w", err)
	}

	return resp.PreAuthKey.Key, nil
}

// getUserIDByName resolves a namespace/user name to its numeric ID
func (h *HeadscaleManager) getUserIDByName(ctx context.Context, name string) (string, error) {
	users, err := h.ListNamespaces(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list users: %w", err)
	}

	for _, user := range users {
		if user.Name == name {
			return user.ID, nil
		}
	}

	return "", fmt.Errorf("user/namespace '%s' not found", name)
}

// ListAuthKeys lists all pre-auth keys for the namespace
func (h *HeadscaleManager) ListAuthKeys(ctx context.Context) ([]HeadscalePreAuthKey, error) {
	var resp listPreAuthKeysResponse
	endpoint := fmt.Sprintf("/api/v1/preauthkey?user=%s", h.config.Namespace)
	if err := h.apiCall(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, fmt.Errorf("failed to list auth keys: %w", err)
	}

	return resp.PreAuthKeys, nil
}

// ExpireAuthKey expires (invalidates) a pre-auth key
func (h *HeadscaleManager) ExpireAuthKey(ctx context.Context, keyID string) error {
	endpoint := fmt.Sprintf("/api/v1/preauthkey/expire")
	req := map[string]string{
		"user": h.config.Namespace,
		"key":  keyID,
	}

	if err := h.apiCall(ctx, "POST", endpoint, req, nil); err != nil {
		return fmt.Errorf("failed to expire auth key: %w", err)
	}

	return nil
}

// ListNodes lists all nodes in the namespace
func (h *HeadscaleManager) ListNodes(ctx context.Context) ([]HeadscaleNode, error) {
	var resp listNodesResponse
	endpoint := fmt.Sprintf("/api/v1/node?user=%s", h.config.Namespace)
	if err := h.apiCall(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	return resp.Nodes, nil
}

// GetNode gets details about a specific node
func (h *HeadscaleManager) GetNode(ctx context.Context, nodeID string) (*HeadscaleNode, error) {
	var resp getNodeResponse
	endpoint := fmt.Sprintf("/api/v1/node/%s", nodeID)
	if err := h.apiCall(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	return &resp.Node, nil
}

// DeleteNode removes a node from the tailnet
func (h *HeadscaleManager) DeleteNode(ctx context.Context, nodeID string) error {
	endpoint := fmt.Sprintf("/api/v1/node/%s", nodeID)
	if err := h.apiCall(ctx, "DELETE", endpoint, nil, nil); err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	return nil
}

// RenameNode renames a node in the tailnet
func (h *HeadscaleManager) RenameNode(ctx context.Context, nodeID, newName string) error {
	endpoint := fmt.Sprintf("/api/v1/node/%s/rename/%s", nodeID, newName)
	if err := h.apiCall(ctx, "POST", endpoint, nil, nil); err != nil {
		return fmt.Errorf("failed to rename node: %w", err)
	}

	return nil
}

// SetNodeTags sets tags on a node
func (h *HeadscaleManager) SetNodeTags(ctx context.Context, nodeID string, tags []string) error {
	endpoint := fmt.Sprintf("/api/v1/node/%s/tags", nodeID)
	req := map[string][]string{
		"tags": tags,
	}
	if err := h.apiCall(ctx, "POST", endpoint, req, nil); err != nil {
		return fmt.Errorf("failed to set node tags: %w", err)
	}

	return nil
}

// ExpireNode expires a node (forces re-authentication)
func (h *HeadscaleManager) ExpireNode(ctx context.Context, nodeID string) error {
	endpoint := fmt.Sprintf("/api/v1/node/%s/expire", nodeID)
	if err := h.apiCall(ctx, "POST", endpoint, nil, nil); err != nil {
		return fmt.Errorf("failed to expire node: %w", err)
	}

	return nil
}

// CreateNamespace creates a new namespace
func (h *HeadscaleManager) CreateNamespace(ctx context.Context, name string) (*HeadscaleNamespace, error) {
	req := createNamespaceRequest{Name: name}
	var resp createNamespaceResponse
	if err := h.apiCall(ctx, "POST", "/api/v1/user", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}

	return &resp.User, nil
}

// ListNamespaces lists all namespaces
func (h *HeadscaleManager) ListNamespaces(ctx context.Context) ([]HeadscaleNamespace, error) {
	var resp listNamespacesResponse
	if err := h.apiCall(ctx, "GET", "/api/v1/user", nil, &resp); err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	return resp.Users, nil
}

// DeleteNamespace deletes a namespace (must be empty)
func (h *HeadscaleManager) DeleteNamespace(ctx context.Context, name string) error {
	endpoint := fmt.Sprintf("/api/v1/user/%s", name)
	if err := h.apiCall(ctx, "DELETE", endpoint, nil, nil); err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	return nil
}

// GetHealthStatus checks if Headscale is healthy
func (h *HeadscaleManager) GetHealthStatus(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", h.config.APIURL+"/health", nil)
	if err != nil {
		return false, err
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// apiCall makes an API call to Headscale
func (h *HeadscaleManager) apiCall(ctx context.Context, method, endpoint string, reqBody interface{}, respBody interface{}) error {
	var bodyReader io.Reader
	if reqBody != nil {
		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	url := h.config.APIURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+h.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	if respBody != nil && len(body) > 0 {
		if err := json.Unmarshal(body, respBody); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// GetConfig returns the current configuration
func (h *HeadscaleManager) GetConfig() HeadscaleConfig {
	return h.config
}

// SetNamespace changes the default namespace for operations
func (h *HeadscaleManager) SetNamespace(namespace string) {
	h.config.Namespace = namespace
}
