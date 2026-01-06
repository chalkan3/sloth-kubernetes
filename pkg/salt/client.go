package salt

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents a Salt API client
type Client struct {
	BaseURL    string
	Username   string
	Password   string
	Token      string
	HTTPClient *http.Client
}

// NewClient creates a new Salt API client
func NewClient(baseURL, username, password string) *Client {
	return &Client{
		BaseURL:  baseURL,
		Username: username,
		Password: password,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second, // Increased timeout for Salt commands
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Eauth    string `json:"eauth"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Return []struct {
		Token  string   `json:"token"`
		Expire float64  `json:"expire"`
		Start  float64  `json:"start"`
		User   string   `json:"user"`
		Eauth  string   `json:"eauth"`
		Perms  []string `json:"perms"`
	} `json:"return"`
}

// Login authenticates with Salt API and obtains a token
// Uses PAM authentication which is the default Salt external_auth method
func (c *Client) Login() error {
	loginReq := LoginRequest{
		Username: c.Username,
		Password: c.Password,
		Eauth:    "pam",
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/login", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	if len(loginResp.Return) == 0 {
		return fmt.Errorf("no token returned from login")
	}

	c.Token = loginResp.Return[0].Token
	return nil
}

// CommandRequest represents a Salt command request
type CommandRequest struct {
	Client  string                 `json:"client"`
	Tgt     string                 `json:"tgt"`
	Fun     string                 `json:"fun"`
	Arg     []string               `json:"arg,omitempty"`
	KWarg   map[string]interface{} `json:"kwarg,omitempty"`
	TgtType string                 `json:"tgt_type,omitempty"`
	Timeout int                    `json:"timeout,omitempty"`
}

// CommandResponse represents a Salt command response
type CommandResponse struct {
	Return []map[string]interface{} `json:"return"`
}

// RunCommand executes a command on Salt minions
func (c *Client) RunCommand(target, function string, args []string) (*CommandResponse, error) {
	if c.Token == "" {
		if err := c.Login(); err != nil {
			return nil, fmt.Errorf("login required: %w", err)
		}
	}

	cmdReq := CommandRequest{
		Client: "local",
		Tgt:    target,
		Fun:    function,
		Arg:    args,
	}

	jsonData, err := json.Marshal(cmdReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Auth-Token", c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("command request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Token expired, re-login and retry
		if err := c.Login(); err != nil {
			return nil, fmt.Errorf("re-login failed: %w", err)
		}
		return c.RunCommand(target, function, args)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("command failed with status %d: %s", resp.StatusCode, string(body))
	}

	var cmdResp CommandResponse
	if err := json.NewDecoder(resp.Body).Decode(&cmdResp); err != nil {
		return nil, fmt.Errorf("failed to decode command response: %w", err)
	}

	return &cmdResp, nil
}

// Ping pings all minions to check connectivity
func (c *Client) Ping(target string) (map[string]bool, error) {
	resp, err := c.RunCommand(target, "test.ping", nil)
	if err != nil {
		return nil, err
	}

	results := make(map[string]bool)
	if len(resp.Return) > 0 {
		for minion, result := range resp.Return[0] {
			if result == true {
				results[minion] = true
			} else {
				results[minion] = false
			}
		}
	}

	return results, nil
}

// GetMinions lists all connected minions
func (c *Client) GetMinions() ([]string, error) {
	resp, err := c.RunCommand("*", "test.ping", nil)
	if err != nil {
		return nil, err
	}

	var minions []string
	if len(resp.Return) > 0 {
		for minion := range resp.Return[0] {
			minions = append(minions, minion)
		}
	}

	return minions, nil
}

// RunShellCommand executes a shell command on minions
func (c *Client) RunShellCommand(target, command string) (*CommandResponse, error) {
	return c.RunCommand(target, "cmd.run", []string{command})
}

// GetGrains retrieves grain data from minions
func (c *Client) GetGrains(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "grains.items", nil)
}

// ApplyState applies a Salt state to minions
func (c *Client) ApplyState(target, state string) (*CommandResponse, error) {
	return c.RunCommand(target, "state.apply", []string{state})
}

// HighState applies the full highstate to minions
func (c *Client) HighState(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "state.highstate", nil)
}

// Package Management

// PackageInstall installs packages on minions
func (c *Client) PackageInstall(target string, packages ...string) (*CommandResponse, error) {
	return c.RunCommand(target, "pkg.install", packages)
}

// PackageRemove removes packages from minions
func (c *Client) PackageRemove(target string, packages ...string) (*CommandResponse, error) {
	return c.RunCommand(target, "pkg.remove", packages)
}

// PackageUpgrade upgrades packages on minions
func (c *Client) PackageUpgrade(target string, packages ...string) (*CommandResponse, error) {
	if len(packages) == 0 {
		return c.RunCommand(target, "pkg.upgrade", nil)
	}
	return c.RunCommand(target, "pkg.upgrade", packages)
}

// PackageList lists installed packages
func (c *Client) PackageList(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "pkg.list_pkgs", nil)
}

// PackageAvailable lists available packages
func (c *Client) PackageAvailable(target string, packageName string) (*CommandResponse, error) {
	return c.RunCommand(target, "pkg.available_version", []string{packageName})
}

// Service Management

// ServiceStart starts a service
func (c *Client) ServiceStart(target, service string) (*CommandResponse, error) {
	return c.RunCommand(target, "service.start", []string{service})
}

// ServiceStop stops a service
func (c *Client) ServiceStop(target, service string) (*CommandResponse, error) {
	return c.RunCommand(target, "service.stop", []string{service})
}

// ServiceRestart restarts a service
func (c *Client) ServiceRestart(target, service string) (*CommandResponse, error) {
	return c.RunCommand(target, "service.restart", []string{service})
}

// ServiceStatus checks service status
func (c *Client) ServiceStatus(target, service string) (*CommandResponse, error) {
	return c.RunCommand(target, "service.status", []string{service})
}

// ServiceEnable enables a service
func (c *Client) ServiceEnable(target, service string) (*CommandResponse, error) {
	return c.RunCommand(target, "service.enable", []string{service})
}

// ServiceDisable disables a service
func (c *Client) ServiceDisable(target, service string) (*CommandResponse, error) {
	return c.RunCommand(target, "service.disable", []string{service})
}

// ServiceGetAll lists all services
func (c *Client) ServiceGetAll(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "service.get_all", nil)
}

// File Management

// FileRead reads a file from minions
func (c *Client) FileRead(target, path string) (*CommandResponse, error) {
	return c.RunCommand(target, "cp.get_file_str", []string{path})
}

// FileWrite writes content to a file
func (c *Client) FileWrite(target, path, content string) (*CommandResponse, error) {
	return c.RunCommand(target, "file.write", []string{path, content})
}

// FileRemove removes a file
func (c *Client) FileRemove(target, path string) (*CommandResponse, error) {
	return c.RunCommand(target, "file.remove", []string{path})
}

// FileExists checks if file exists
func (c *Client) FileExists(target, path string) (*CommandResponse, error) {
	return c.RunCommand(target, "file.file_exists", []string{path})
}

// FileCopy copies files between minions
func (c *Client) FileCopy(target, source, dest string) (*CommandResponse, error) {
	return c.RunCommand(target, "file.copy", []string{source, dest})
}

// FileChmod changes file permissions
func (c *Client) FileChmod(target, path, mode string) (*CommandResponse, error) {
	return c.RunCommand(target, "file.set_mode", []string{path, mode})
}

// FileChown changes file ownership
func (c *Client) FileChown(target, path, user, group string) (*CommandResponse, error) {
	return c.RunCommand(target, "file.chown", []string{path, user, group})
}

// User Management

// UserAdd adds a user
func (c *Client) UserAdd(target, username string) (*CommandResponse, error) {
	return c.RunCommand(target, "user.add", []string{username})
}

// UserDelete deletes a user
func (c *Client) UserDelete(target, username string) (*CommandResponse, error) {
	return c.RunCommand(target, "user.delete", []string{username})
}

// UserList lists all users
func (c *Client) UserList(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "user.list_users", nil)
}

// UserInfo gets user information
func (c *Client) UserInfo(target, username string) (*CommandResponse, error) {
	return c.RunCommand(target, "user.info", []string{username})
}

// Group Management

// GroupAdd adds a group
func (c *Client) GroupAdd(target, groupname string) (*CommandResponse, error) {
	return c.RunCommand(target, "group.add", []string{groupname})
}

// GroupDelete deletes a group
func (c *Client) GroupDelete(target, groupname string) (*CommandResponse, error) {
	return c.RunCommand(target, "group.delete", []string{groupname})
}

// System Management

// SystemReboot reboots minions
func (c *Client) SystemReboot(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "system.reboot", nil)
}

// SystemShutdown shuts down minions
func (c *Client) SystemShutdown(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "system.shutdown", nil)
}

// SystemUptime gets system uptime
func (c *Client) SystemUptime(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "status.uptime", nil)
}

// DiskUsage gets disk usage
func (c *Client) DiskUsage(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "disk.usage", nil)
}

// MemoryUsage gets memory usage
func (c *Client) MemoryUsage(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "status.meminfo", nil)
}

// CPUInfo gets CPU information
func (c *Client) CPUInfo(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "status.cpuinfo", nil)
}

// NetworkInterfaces gets network interfaces
func (c *Client) NetworkInterfaces(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "network.interfaces", nil)
}

// Job Management

// JobsList lists recent jobs
func (c *Client) JobsList(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "saltutil.find_job", nil)
}

// JobKill kills a running job
func (c *Client) JobKill(target, jid string) (*CommandResponse, error) {
	return c.RunCommand(target, "saltutil.kill_job", []string{jid})
}

// SyncAll syncs all modules to minions
func (c *Client) SyncAll(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "saltutil.sync_all", nil)
}

// Schedule Management

// ScheduleList lists scheduled jobs
func (c *Client) ScheduleList(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "schedule.list", nil)
}

// Docker Management

// DockerPS lists running containers
func (c *Client) DockerPS(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "docker.ps", nil)
}

// DockerStart starts a container
func (c *Client) DockerStart(target, container string) (*CommandResponse, error) {
	return c.RunCommand(target, "docker.start", []string{container})
}

// DockerStop stops a container
func (c *Client) DockerStop(target, container string) (*CommandResponse, error) {
	return c.RunCommand(target, "docker.stop", []string{container})
}

// DockerRestart restarts a container
func (c *Client) DockerRestart(target, container string) (*CommandResponse, error) {
	return c.RunCommand(target, "docker.restart", []string{container})
}

// Git Operations

// GitClone clones a git repository
func (c *Client) GitClone(target, repo, dest string) (*CommandResponse, error) {
	return c.RunCommand(target, "git.clone", []string{dest, repo})
}

// GitPull pulls latest changes
func (c *Client) GitPull(target, repo string) (*CommandResponse, error) {
	return c.RunCommand(target, "git.pull", []string{repo})
}

// Network Management

// NetworkPing pings a host
func (c *Client) NetworkPing(target, host string, count int) (*CommandResponse, error) {
	return c.RunCommand(target, "network.ping", []string{host, fmt.Sprintf("%d", count)})
}

// NetworkTraceroute traces route to host
func (c *Client) NetworkTraceroute(target, host string) (*CommandResponse, error) {
	return c.RunCommand(target, "network.traceroute", []string{host})
}

// NetworkNetstat shows network statistics
func (c *Client) NetworkNetstat(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "network.netstat", nil)
}

// NetworkActiveConnections shows active connections
func (c *Client) NetworkActiveConnections(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "network.active_tcp", nil)
}

// NetworkDefaultRoute shows default route
func (c *Client) NetworkDefaultRoute(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "network.default_route", nil)
}

// NetworkRoutes shows all routes
func (c *Client) NetworkRoutes(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "network.routes", nil)
}

// NetworkARP shows ARP table
func (c *Client) NetworkARP(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "network.arp", nil)
}

// Process Management

// ProcessList lists all processes
func (c *Client) ProcessList(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "ps.pgrep", []string{".*"})
}

// ProcessTop shows top processes
func (c *Client) ProcessTop(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "ps.top", nil)
}

// ProcessKill kills a process
func (c *Client) ProcessKill(target, pid, signal string) (*CommandResponse, error) {
	return c.RunCommand(target, "ps.kill_pid", []string{pid, signal})
}

// ProcessInfo gets process information
func (c *Client) ProcessInfo(target, pid string) (*CommandResponse, error) {
	return c.RunCommand(target, "ps.proc_info", []string{pid})
}

// Cron Management

// CronList lists cron jobs
func (c *Client) CronList(target, user string) (*CommandResponse, error) {
	return c.RunCommand(target, "cron.list_tab", []string{user})
}

// CronAdd adds a cron job
func (c *Client) CronAdd(target, user, minute, hour, daymonth, month, dayweek, cmd string) (*CommandResponse, error) {
	return c.RunCommand(target, "cron.set_job", []string{user, minute, hour, daymonth, month, dayweek, cmd})
}

// CronRemove removes a cron job
func (c *Client) CronRemove(target, user, cmd string) (*CommandResponse, error) {
	return c.RunCommand(target, "cron.rm_job", []string{user, cmd})
}

// Archive Management

// ArchiveTar creates a tar archive
func (c *Client) ArchiveTar(target, source, dest string) (*CommandResponse, error) {
	return c.RunCommand(target, "archive.tar", []string{"czf", dest, source})
}

// ArchiveUntar extracts a tar archive
func (c *Client) ArchiveUntar(target, source, dest string) (*CommandResponse, error) {
	return c.RunCommand(target, "archive.tar", []string{"xzf", source, "-C", dest})
}

// ArchiveZip creates a zip archive
func (c *Client) ArchiveZip(target, source, dest string) (*CommandResponse, error) {
	return c.RunCommand(target, "archive.zip", []string{dest, source})
}

// ArchiveUnzip extracts a zip archive
func (c *Client) ArchiveUnzip(target, source, dest string) (*CommandResponse, error) {
	return c.RunCommand(target, "archive.unzip", []string{source, dest})
}

// Monitoring and Status

// LoadAverage gets load average
func (c *Client) LoadAverage(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "status.loadavg", nil)
}

// DiskIOStats gets disk I/O statistics
func (c *Client) DiskIOStats(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "disk.iostat", nil)
}

// NetworkIOStats gets network I/O statistics
func (c *Client) NetworkIOStats(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "status.netstats", nil)
}

// SystemTime gets system time
func (c *Client) SystemTime(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "system.get_system_time", nil)
}

// Timezone gets timezone
func (c *Client) Timezone(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "timezone.get_zone", nil)
}

// Hostname gets hostname
func (c *Client) Hostname(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "network.get_hostname", nil)
}

// KernelVersion gets kernel version
func (c *Client) KernelVersion(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "system.get_kernel", nil)
}

// OSVersion gets OS version
func (c *Client) OSVersion(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "grains.get", []string{"osrelease"})
}

// SystemInfo gets comprehensive system information
func (c *Client) SystemInfo(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "status.all_status", nil)
}

// Firewall Management

// FirewallList lists firewall rules
func (c *Client) FirewallList(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "firewalld.list_all", nil)
}

// FirewallAddRule adds a firewall rule
func (c *Client) FirewallAddRule(target, port, protocol string) (*CommandResponse, error) {
	return c.RunCommand(target, "firewalld.add_port", []string{port, protocol})
}

// FirewallRemoveRule removes a firewall rule
func (c *Client) FirewallRemoveRule(target, port, protocol string) (*CommandResponse, error) {
	return c.RunCommand(target, "firewalld.remove_port", []string{port, protocol})
}

// Mount Management

// MountList lists mounted filesystems
func (c *Client) MountList(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "mount.active", nil)
}

// MountFS mounts a filesystem
func (c *Client) MountFS(target, device, mountpoint, fstype string) (*CommandResponse, error) {
	return c.RunCommand(target, "mount.mount", []string{mountpoint, device, fstype})
}

// UnmountFS unmounts a filesystem
func (c *Client) UnmountFS(target, mountpoint string) (*CommandResponse, error) {
	return c.RunCommand(target, "mount.umount", []string{mountpoint})
}

// SSH Key Management

// SSHKeyGen generates SSH key
func (c *Client) SSHKeyGen(target, user, keytype string) (*CommandResponse, error) {
	return c.RunCommand(target, "ssh.key_gen", []string{user, keytype})
}

// SSHAuth lists authorized keys
func (c *Client) SSHAuth(target, user string) (*CommandResponse, error) {
	return c.RunCommand(target, "ssh.auth_keys", []string{user})
}

// SSHSetAuth sets authorized key
func (c *Client) SSHSetAuth(target, user, key string) (*CommandResponse, error) {
	return c.RunCommand(target, "ssh.set_auth_key", []string{user, key})
}

// Environment Variables

// EnvGet gets environment variable
func (c *Client) EnvGet(target, key string) (*CommandResponse, error) {
	return c.RunCommand(target, "environ.get", []string{key})
}

// EnvSet sets environment variable
func (c *Client) EnvSet(target, key, value string) (*CommandResponse, error) {
	return c.RunCommand(target, "environ.setval", []string{key, value})
}

// EnvList lists all environment variables
func (c *Client) EnvList(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "environ.items", nil)
}

// HTTP Requests

// HTTPQuery makes HTTP request
func (c *Client) HTTPQuery(target, url, method string) (*CommandResponse, error) {
	return c.RunCommand(target, "http.query", []string{url, "method=" + method})
}

// Modules and Grain Management

// ModulesList lists available modules
func (c *Client) ModulesList(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "sys.list_modules", nil)
}

// FunctionsList lists available functions
func (c *Client) FunctionsList(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "sys.list_functions", nil)
}

// GrainSet sets a grain value
func (c *Client) GrainSet(target, key, value string) (*CommandResponse, error) {
	return c.RunCommand(target, "grains.setval", []string{key, value})
}

// GrainGet gets a grain value
func (c *Client) GrainGet(target, key string) (*CommandResponse, error) {
	return c.RunCommand(target, "grains.get", []string{key})
}

// GrainDelete deletes a grain
func (c *Client) GrainDelete(target, key string) (*CommandResponse, error) {
	return c.RunCommand(target, "grains.delval", []string{key})
}

// Pillar Management

// PillarGet gets pillar data
func (c *Client) PillarGet(target, key string) (*CommandResponse, error) {
	return c.RunCommand(target, "pillar.get", []string{key})
}

// PillarItems gets all pillar items
func (c *Client) PillarItems(target string) (*CommandResponse, error) {
	return c.RunCommand(target, "pillar.items", nil)
}

// Kubernetes Operations

// KubectlGet runs kubectl get
func (c *Client) KubectlGet(target, resource string) (*CommandResponse, error) {
	return c.RunCommand(target, "cmd.run", []string{"kubectl get " + resource})
}

// KubectlApply applies kubernetes manifest
func (c *Client) KubectlApply(target, manifest string) (*CommandResponse, error) {
	return c.RunCommand(target, "cmd.run", []string{"kubectl apply -f " + manifest})
}

// KubectlDelete deletes kubernetes resource
func (c *Client) KubectlDelete(target, resource, name string) (*CommandResponse, error) {
	return c.RunCommand(target, "cmd.run", []string{fmt.Sprintf("kubectl delete %s %s", resource, name)})
}

// KeyAccept accepts minion keys
func (c *Client) KeyAccept(minionID string) error {
	if c.Token == "" {
		if err := c.Login(); err != nil {
			return fmt.Errorf("login required: %w", err)
		}
	}

	cmdReq := map[string]interface{}{
		"client": "wheel",
		"fun":    "key.accept",
		"match":  minionID,
	}

	jsonData, err := json.Marshal(cmdReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Auth-Token", c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("key accept request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("key accept failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// KeyList lists minion keys
func (c *Client) KeyList() (map[string][]string, error) {
	if c.Token == "" {
		if err := c.Login(); err != nil {
			return nil, fmt.Errorf("login required: %w", err)
		}
	}

	cmdReq := map[string]interface{}{
		"client": "wheel",
		"fun":    "key.list_all",
	}

	jsonData, err := json.Marshal(cmdReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Auth-Token", c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("key list request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("key list failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Return []struct {
			Data map[string]interface{} `json:"data"`
		} `json:"return"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	keys := make(map[string][]string)
	if len(result.Return) > 0 {
		data := result.Return[0].Data
		// The actual keys are nested inside data["return"]
		if returnData, ok := data["return"].(map[string]interface{}); ok {
			data = returnData
		}
		for keyType, keyList := range data {
			if keySlice, ok := keyList.([]interface{}); ok {
				strSlice := make([]string, len(keySlice))
				for i, v := range keySlice {
					strSlice[i] = fmt.Sprintf("%v", v)
				}
				keys[keyType] = strSlice
			}
		}
	}

	return keys, nil
}
