package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupComprehensiveMockSaltServer creates a mock server that handles all Salt API operations
func setupComprehensiveMockSaltServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Handle login
		if r.URL.Path == "/login" && r.Method == "POST" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"return": []map[string]interface{}{
					{
						"token":  "test-token-comprehensive",
						"start":  1234567890.0,
						"expire": 9999999999.0,
						"user":   "saltapi",
						"eauth":  "pam",
						"perms":  []string{".*", "@wheel", "@runner"},
					},
				},
			})
			return
		}

		// Handle command requests
		if r.URL.Path == "/" || r.URL.Path == "/run" {
			var reqBody map[string]interface{}
			json.NewDecoder(r.Body).Decode(&reqBody)

			fun, _ := reqBody["fun"].(string)
			client, _ := reqBody["client"].(string)

			// Wheel commands (key management)
			if client == "wheel" {
				handleWheelCommands(w, fun, reqBody)
				return
			}

			// Local commands
			handleLocalCommands(w, fun, reqBody)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

func handleWheelCommands(w http.ResponseWriter, fun string, reqBody map[string]interface{}) {
	switch fun {
	case "key.list_all":
		json.NewEncoder(w).Encode(map[string]interface{}{
			"return": []map[string]interface{}{
				{
					"data": map[string]interface{}{
						"minions":          []string{"minion-1", "minion-2", "minion-3"},
						"minions_pre":      []string{"pending-minion-1"},
						"minions_rejected": []string{},
						"minions_denied":   []string{},
					},
				},
			},
		})
	case "key.accept":
		json.NewEncoder(w).Encode(map[string]interface{}{
			"return": []map[string]interface{}{
				{
					"data": map[string]interface{}{
						"minions": []string{"pending-minion-1"},
					},
				},
			},
		})
	default:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"return": []map[string]interface{}{{}},
		})
	}
}

func handleLocalCommands(w http.ResponseWriter, fun string, reqBody map[string]interface{}) {
	response := map[string]interface{}{}

	switch fun {
	// Basic commands
	case "test.ping":
		response["return"] = []map[string]bool{
			{"minion-1": true, "minion-2": true, "minion-3": true},
		}

	case "cmd.run":
		args, _ := reqBody["arg"].([]interface{})
		cmd := ""
		if len(args) > 0 {
			cmd, _ = args[0].(string)
		}
		response["return"] = []map[string]string{
			{
				"minion-1": "output from minion-1: " + cmd,
				"minion-2": "output from minion-2: " + cmd,
			},
		}

	case "grains.items":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]interface{}{
					"os": "Ubuntu", "osrelease": "22.04", "kernel": "5.15.0",
					"cpuarch": "x86_64", "num_cpus": 4, "mem_total": 8192,
				},
				"minion-2": map[string]interface{}{
					"os": "Ubuntu", "osrelease": "22.04", "kernel": "5.15.0",
					"cpuarch": "x86_64", "num_cpus": 2, "mem_total": 4096,
				},
			},
		}

	case "grains.get":
		response["return"] = []map[string]interface{}{
			{"minion-1": "test_value", "minion-2": "test_value"},
		}

	case "grains.setval":
		response["return"] = []map[string]interface{}{
			{"minion-1": map[string]string{"test_key": "test_value"}},
		}

	case "grains.delval":
		response["return"] = []map[string]interface{}{
			{"minion-1": nil, "minion-2": nil},
		}

	// State commands
	case "state.apply":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]interface{}{
					"pkg_|-nginx_|-nginx_|-installed": map[string]interface{}{
						"result": true, "comment": "Package installed",
						"changes": map[string]string{"new": "1.18.0"},
					},
				},
			},
		}

	case "state.highstate":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]interface{}{
					"result": true, "comment": "Highstate applied",
				},
			},
		}

	// Package commands
	case "pkg.install":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]string{"nginx": "installed"},
				"minion-2": map[string]string{"nginx": "installed"},
			},
		}

	case "pkg.remove":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]string{"nginx": "removed"},
			},
		}

	case "pkg.upgrade":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]string{"upgraded": "true"},
			},
		}

	case "pkg.list_pkgs":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]string{
					"bash": "5.1", "nginx": "1.18", "curl": "7.81",
				},
			},
		}

	// Service commands
	case "service.start":
		response["return"] = []map[string]bool{
			{"minion-1": true, "minion-2": true},
		}

	case "service.stop":
		response["return"] = []map[string]bool{
			{"minion-1": true, "minion-2": true},
		}

	case "service.restart":
		response["return"] = []map[string]bool{
			{"minion-1": true, "minion-2": true},
		}

	case "service.status":
		response["return"] = []map[string]bool{
			{"minion-1": true, "minion-2": false},
		}

	case "service.enable":
		response["return"] = []map[string]bool{
			{"minion-1": true},
		}

	case "service.disable":
		response["return"] = []map[string]bool{
			{"minion-1": true},
		}

	case "service.get_all":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": []string{"nginx", "ssh", "cron", "docker"},
			},
		}

	// File commands
	case "file.file_exists":
		response["return"] = []map[string]bool{
			{"minion-1": true, "minion-2": true},
		}

	case "file.write":
		response["return"] = []map[string]string{
			{"minion-1": "File written", "minion-2": "File written"},
		}

	case "file.remove":
		response["return"] = []map[string]bool{
			{"minion-1": true, "minion-2": true},
		}

	case "cp.get_file_str":
		response["return"] = []map[string]string{
			{"minion-1": "file content here", "minion-2": "file content here"},
		}

	case "file.set_mode":
		response["return"] = []map[string]string{
			{"minion-1": "0755", "minion-2": "0755"},
		}

	// System commands
	case "status.uptime":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]interface{}{
					"days": 5, "hours": 12, "minutes": 30, "seconds": 45,
				},
			},
		}

	case "disk.usage":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]interface{}{
					"/": map[string]string{
						"total": "100G", "used": "45G", "available": "55G",
					},
				},
			},
		}

	case "status.meminfo":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]interface{}{
					"MemTotal": 8192000, "MemFree": 4096000, "MemAvailable": 5000000,
				},
			},
		}

	case "status.cpuinfo":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]interface{}{
					"0": map[string]interface{}{
						"model name": "Intel Core i7", "cpu MHz": "3600.00",
					},
				},
			},
		}

	case "network.interfaces":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]interface{}{
					"eth0": map[string]interface{}{
						"inet": []map[string]string{{"address": "10.0.0.1"}},
					},
				},
			},
		}

	case "status.loadavg":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]float64{"1-min": 0.5, "5-min": 0.3, "15-min": 0.2},
			},
		}

	// User commands
	case "user.add":
		response["return"] = []map[string]bool{
			{"minion-1": true, "minion-2": true},
		}

	case "user.delete":
		response["return"] = []map[string]bool{
			{"minion-1": true, "minion-2": true},
		}

	case "user.list_users":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": []string{"root", "ubuntu", "testuser"},
			},
		}

	case "user.info":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]interface{}{
					"name": "testuser", "uid": 1001, "gid": 1001, "home": "/home/testuser",
				},
			},
		}

	// Network commands
	case "network.ping":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]interface{}{
					"result": true, "host": "8.8.8.8", "latency": "10ms",
				},
			},
		}

	case "network.traceroute":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": []map[string]interface{}{
					{"hop": 1, "host": "gateway", "latency": "1ms"},
					{"hop": 2, "host": "8.8.8.8", "latency": "10ms"},
				},
			},
		}

	case "network.netstat":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": "tcp  0  0  0.0.0.0:22  LISTEN",
			},
		}

	case "network.routes":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": []map[string]string{
					{"destination": "0.0.0.0", "gateway": "10.0.0.1", "interface": "eth0"},
				},
			},
		}

	case "network.arp":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]string{"10.0.0.1": "aa:bb:cc:dd:ee:ff"},
			},
		}

	case "network.active_tcp":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": []map[string]string{
					{"local": "0.0.0.0:22", "remote": "10.0.0.100:54321", "state": "ESTABLISHED"},
				},
			},
		}

	// Docker commands
	case "docker.ps":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": []map[string]string{
					{"id": "abc123", "name": "nginx", "status": "running"},
				},
			},
		}

	case "docker.start":
		response["return"] = []map[string]bool{
			{"minion-1": true},
		}

	case "docker.stop":
		response["return"] = []map[string]bool{
			{"minion-1": true},
		}

	case "docker.restart":
		response["return"] = []map[string]bool{
			{"minion-1": true},
		}

	// Job commands
	case "saltutil.find_job":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]interface{}{
					"jid": "20231201120000", "fun": "test.ping",
				},
			},
		}

	case "saltutil.kill_job":
		response["return"] = []map[string]bool{
			{"minion-1": true},
		}

	case "saltutil.sync_all":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string][]string{
					"modules": {"mod1", "mod2"}, "states": {"state1"},
				},
			},
		}

	// Process commands
	case "ps.pgrep":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": []int{1, 100, 200, 300},
			},
		}

	case "ps.top":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": "top output here",
			},
		}

	case "ps.kill_pid":
		response["return"] = []map[string]bool{
			{"minion-1": true},
		}

	// Cron commands
	case "cron.list_tab":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]interface{}{
					"crons": []map[string]string{
						{"minute": "0", "hour": "2", "cmd": "/usr/bin/backup.sh"},
					},
				},
			},
		}

	case "cron.set_job":
		response["return"] = []map[string]string{
			{"minion-1": "new"},
		}

	case "cron.rm_job":
		response["return"] = []map[string]string{
			{"minion-1": "absent"},
		}

	// Archive commands
	case "archive.tar":
		response["return"] = []map[string]interface{}{
			{"minion-1": []string{"/tmp/archive.tar.gz"}},
		}

	case "archive.zip":
		response["return"] = []map[string]interface{}{
			{"minion-1": []string{"/tmp/archive.zip"}},
		}

	case "archive.unzip":
		response["return"] = []map[string]interface{}{
			{"minion-1": []string{"file1.txt", "file2.txt"}},
		}

	// Monitor commands
	case "status.all_status":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]interface{}{
					"loadavg": map[string]float64{"1-min": 0.5},
					"cpuload": 25.5,
					"meminfo": map[string]int{"MemTotal": 8192},
				},
			},
		}

	case "disk.iostat":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]interface{}{
					"sda": map[string]float64{"read_bytes": 1000, "write_bytes": 500},
				},
			},
		}

	case "status.netstats":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]int{
					"eth0": 12345678,
				},
			},
		}

	// SSH commands
	case "ssh.key_gen":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]string{
					"public":  "ssh-rsa AAAA...",
					"private": "generated",
				},
			},
		}

	case "ssh.auth_keys":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": []string{"ssh-rsa AAAA... user@host"},
			},
		}

	case "ssh.set_auth_key":
		response["return"] = []map[string]string{
			{"minion-1": "new"},
		}

	// Git commands
	case "git.clone":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]bool{"cloned": true},
			},
		}

	case "git.pull":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]string{"result": "Already up to date."},
			},
		}

	// Pillar commands
	case "pillar.get":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": "pillar_value",
			},
		}

	case "pillar.items":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]string{"key1": "value1", "key2": "value2"},
			},
		}

	// Mount commands
	case "mount.active":
		response["return"] = []map[string]interface{}{
			{
				"minion-1": map[string]interface{}{
					"/": map[string]string{"device": "/dev/sda1", "fstype": "ext4"},
				},
			},
		}

	case "mount.mount":
		response["return"] = []map[string]bool{
			{"minion-1": true},
		}

	case "mount.umount":
		response["return"] = []map[string]bool{
			{"minion-1": true},
		}

	default:
		response["return"] = []map[string]interface{}{
			{"minion-1": "unknown command"},
		}
	}

	json.NewEncoder(w).Encode(response)
}

// =============================================================================
// Functional Tests for Salt CLI Commands
// =============================================================================

// TestSaltCLI_BasicCommands_Functional tests basic salt commands execution
func TestSaltCLI_BasicCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	// Setup global variables
	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"
	saltOutputJSON = false

	t.Run("Ping_Execution", func(t *testing.T) {
		// Capture output
		var buf bytes.Buffer
		pingCmd.SetOut(&buf)
		pingCmd.SetErr(&buf)

		err := runSaltPing(pingCmd, []string{})
		assert.NoError(t, err, "Ping command should execute without error")
	})

	t.Run("Minions_Execution", func(t *testing.T) {
		var buf bytes.Buffer
		minionsCmd.SetOut(&buf)
		minionsCmd.SetErr(&buf)

		err := runSaltMinions(minionsCmd, []string{})
		assert.NoError(t, err, "Minions command should execute without error")
	})

	t.Run("Cmd_Execution", func(t *testing.T) {
		var buf bytes.Buffer
		cmdCmd.SetOut(&buf)
		cmdCmd.SetErr(&buf)

		err := runSaltCmd(cmdCmd, []string{"uptime"})
		assert.NoError(t, err, "Cmd command should execute without error")
	})

	t.Run("Grains_Execution", func(t *testing.T) {
		var buf bytes.Buffer
		grainsCmd.SetOut(&buf)
		grainsCmd.SetErr(&buf)

		err := runSaltGrains(grainsCmd, []string{})
		assert.NoError(t, err, "Grains command should execute without error")
	})
}

// TestSaltCLI_StateCommands_Functional tests state commands execution
func TestSaltCLI_StateCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("StateApply_Execution", func(t *testing.T) {
		var buf bytes.Buffer
		saltStateApplyCmd.SetOut(&buf)
		saltStateApplyCmd.SetErr(&buf)

		err := runSaltStateApply(saltStateApplyCmd, []string{"nginx"})
		assert.NoError(t, err, "State apply command should execute without error")
	})

	t.Run("Highstate_Execution", func(t *testing.T) {
		var buf bytes.Buffer
		saltStateHighstateCmd.SetOut(&buf)
		saltStateHighstateCmd.SetErr(&buf)

		err := runSaltHighstate(saltStateHighstateCmd, []string{})
		assert.NoError(t, err, "Highstate command should execute without error")
	})
}

// TestSaltCLI_PackageCommands_Functional tests package commands execution
func TestSaltCLI_PackageCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("PkgInstall_Execution", func(t *testing.T) {
		err := runPkgInstall(pkgInstallCmd, []string{"nginx"})
		assert.NoError(t, err, "Package install should execute without error")
	})

	t.Run("PkgRemove_Execution", func(t *testing.T) {
		err := runPkgRemove(pkgRemoveCmd, []string{"nginx"})
		assert.NoError(t, err, "Package remove should execute without error")
	})

	t.Run("PkgUpgrade_Execution", func(t *testing.T) {
		err := runPkgUpgrade(pkgUpgradeCmd, []string{})
		assert.NoError(t, err, "Package upgrade should execute without error")
	})

	t.Run("PkgList_Execution", func(t *testing.T) {
		err := runPkgList(pkgListCmd, []string{})
		assert.NoError(t, err, "Package list should execute without error")
	})
}

// TestSaltCLI_ServiceCommands_Functional tests service commands execution
func TestSaltCLI_ServiceCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("ServiceStart_Execution", func(t *testing.T) {
		err := runServiceStart(serviceStartCmd, []string{"nginx"})
		assert.NoError(t, err, "Service start should execute without error")
	})

	t.Run("ServiceStop_Execution", func(t *testing.T) {
		err := runServiceStop(serviceStopCmd, []string{"nginx"})
		assert.NoError(t, err, "Service stop should execute without error")
	})

	t.Run("ServiceRestart_Execution", func(t *testing.T) {
		err := runServiceRestart(serviceRestartCmd, []string{"nginx"})
		assert.NoError(t, err, "Service restart should execute without error")
	})

	t.Run("ServiceStatus_Execution", func(t *testing.T) {
		err := runServiceStatus(serviceStatusCmd, []string{"nginx"})
		assert.NoError(t, err, "Service status should execute without error")
	})

	t.Run("ServiceEnable_Execution", func(t *testing.T) {
		err := runServiceEnable(serviceEnableCmd, []string{"nginx"})
		assert.NoError(t, err, "Service enable should execute without error")
	})

	t.Run("ServiceDisable_Execution", func(t *testing.T) {
		err := runServiceDisable(serviceDisableCmd, []string{"nginx"})
		assert.NoError(t, err, "Service disable should execute without error")
	})

	t.Run("ServiceList_Execution", func(t *testing.T) {
		err := runServiceList(serviceListCmd, []string{})
		assert.NoError(t, err, "Service list should execute without error")
	})
}

// TestSaltCLI_FileCommands_Functional tests file commands execution
func TestSaltCLI_FileCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("FileRead_Execution", func(t *testing.T) {
		err := runFileRead(fileReadCmd, []string{"/etc/hostname"})
		assert.NoError(t, err, "File read should execute without error")
	})

	t.Run("FileWrite_Execution", func(t *testing.T) {
		err := runFileWrite(fileWriteCmd, []string{"/tmp/test.txt", "test content"})
		assert.NoError(t, err, "File write should execute without error")
	})

	t.Run("FileExists_Execution", func(t *testing.T) {
		err := runFileExists(fileExistsCmd, []string{"/etc/hostname"})
		assert.NoError(t, err, "File exists should execute without error")
	})

	t.Run("FileRemove_Execution", func(t *testing.T) {
		err := runFileRemove(fileRemoveCmd, []string{"/tmp/test.txt"})
		assert.NoError(t, err, "File remove should execute without error")
	})

	t.Run("FileChmod_Execution", func(t *testing.T) {
		err := runFileChmod(fileChmodCmd, []string{"/tmp/script.sh", "755"})
		assert.NoError(t, err, "File chmod should execute without error")
	})
}

// TestSaltCLI_SystemCommands_Functional tests system commands execution
func TestSaltCLI_SystemCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("SystemUptime_Execution", func(t *testing.T) {
		err := runSystemUptime(systemUptimeCmd, []string{})
		assert.NoError(t, err, "System uptime should execute without error")
	})

	t.Run("SystemDisk_Execution", func(t *testing.T) {
		err := runSystemDisk(systemDiskCmd, []string{})
		assert.NoError(t, err, "System disk should execute without error")
	})

	t.Run("SystemMemory_Execution", func(t *testing.T) {
		err := runSystemMemory(systemMemoryCmd, []string{})
		assert.NoError(t, err, "System memory should execute without error")
	})

	t.Run("SystemCPU_Execution", func(t *testing.T) {
		err := runSystemCPU(systemCPUCmd, []string{})
		assert.NoError(t, err, "System CPU should execute without error")
	})

	t.Run("SystemNetwork_Execution", func(t *testing.T) {
		err := runSystemNetwork(systemNetworkCmd, []string{})
		assert.NoError(t, err, "System network should execute without error")
	})
}

// TestSaltCLI_UserCommands_Functional tests user commands execution
func TestSaltCLI_UserCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("UserAdd_Execution", func(t *testing.T) {
		err := runUserAdd(userAddCmd, []string{"testuser"})
		assert.NoError(t, err, "User add should execute without error")
	})

	t.Run("UserList_Execution", func(t *testing.T) {
		err := runUserList(userListCmd, []string{})
		assert.NoError(t, err, "User list should execute without error")
	})

	t.Run("UserInfo_Execution", func(t *testing.T) {
		err := runUserInfo(userInfoCmd, []string{"testuser"})
		assert.NoError(t, err, "User info should execute without error")
	})

	t.Run("UserDelete_Execution", func(t *testing.T) {
		err := runUserDelete(userDeleteCmd, []string{"testuser"})
		assert.NoError(t, err, "User delete should execute without error")
	})
}

// TestSaltCLI_DockerCommands_Functional tests docker commands execution
func TestSaltCLI_DockerCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("DockerPS_Execution", func(t *testing.T) {
		err := runDockerPS(dockerPSCmd, []string{})
		assert.NoError(t, err, "Docker ps should execute without error")
	})

	t.Run("DockerStart_Execution", func(t *testing.T) {
		err := runDockerStart(dockerStartCmd, []string{"nginx"})
		assert.NoError(t, err, "Docker start should execute without error")
	})

	t.Run("DockerStop_Execution", func(t *testing.T) {
		err := runDockerStop(dockerStopCmd, []string{"nginx"})
		assert.NoError(t, err, "Docker stop should execute without error")
	})

	t.Run("DockerRestart_Execution", func(t *testing.T) {
		err := runDockerRestart(dockerRestartCmd, []string{"nginx"})
		assert.NoError(t, err, "Docker restart should execute without error")
	})
}

// TestSaltCLI_JobCommands_Functional tests job commands execution
func TestSaltCLI_JobCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("JobList_Execution", func(t *testing.T) {
		err := runJobList(jobListCmd, []string{})
		assert.NoError(t, err, "Job list should execute without error")
	})

	t.Run("JobKill_Execution", func(t *testing.T) {
		err := runJobKill(jobKillCmd, []string{"20231201120000"})
		assert.NoError(t, err, "Job kill should execute without error")
	})

	t.Run("JobSync_Execution", func(t *testing.T) {
		err := runJobSync(jobSyncCmd, []string{})
		assert.NoError(t, err, "Job sync should execute without error")
	})
}

// TestSaltCLI_KeysCommands_Functional tests keys commands execution
func TestSaltCLI_KeysCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("KeysList_Execution", func(t *testing.T) {
		err := runSaltKeysList(keysListCmd, []string{})
		assert.NoError(t, err, "Keys list should execute without error")
	})

	t.Run("KeysAccept_Execution", func(t *testing.T) {
		err := runSaltKeysAccept(keysAcceptCmd, []string{"pending-minion-1"})
		assert.NoError(t, err, "Keys accept should execute without error")
	})
}

// TestSaltCLI_AdvancedNetworkCommands_Functional tests advanced network commands
func TestSaltCLI_AdvancedNetworkCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("NetworkPing_Execution", func(t *testing.T) {
		err := runNetworkPing(networkPingCmd, []string{"8.8.8.8"})
		assert.NoError(t, err, "Network ping should execute without error")
	})

	t.Run("NetworkTraceroute_Execution", func(t *testing.T) {
		err := runNetworkTraceroute(networkTracerouteCmd, []string{"8.8.8.8"})
		assert.NoError(t, err, "Network traceroute should execute without error")
	})

	t.Run("NetworkNetstat_Execution", func(t *testing.T) {
		err := runNetworkNetstat(networkNetstatCmd, []string{})
		assert.NoError(t, err, "Network netstat should execute without error")
	})

	t.Run("NetworkRoutes_Execution", func(t *testing.T) {
		err := runNetworkRoutes(networkRoutesCmd, []string{})
		assert.NoError(t, err, "Network routes should execute without error")
	})

	t.Run("NetworkARP_Execution", func(t *testing.T) {
		err := runNetworkARP(networkARPCmd, []string{})
		assert.NoError(t, err, "Network ARP should execute without error")
	})

	t.Run("NetworkConnections_Execution", func(t *testing.T) {
		err := runNetworkConnections(networkConnectionsCmd, []string{})
		assert.NoError(t, err, "Network connections should execute without error")
	})
}

// TestSaltCLI_ProcessCommands_Functional tests process commands execution
func TestSaltCLI_ProcessCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("ProcessList_Execution", func(t *testing.T) {
		err := runProcessList(processListCmd, []string{})
		assert.NoError(t, err, "Process list should execute without error")
	})

	t.Run("ProcessTop_Execution", func(t *testing.T) {
		err := runProcessTop(processTopCmd, []string{})
		assert.NoError(t, err, "Process top should execute without error")
	})

	t.Run("ProcessKill_Execution", func(t *testing.T) {
		err := runProcessKill(processKillCmd, []string{"12345", "SIGTERM"})
		assert.NoError(t, err, "Process kill should execute without error")
	})
}

// TestSaltCLI_CronCommands_Functional tests cron commands execution
func TestSaltCLI_CronCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("CronList_Execution", func(t *testing.T) {
		err := runCronList(cronListCmd, []string{"root"})
		assert.NoError(t, err, "Cron list should execute without error")
	})

	t.Run("CronAdd_Execution", func(t *testing.T) {
		err := runCronAdd(cronAddCmd, []string{"root", "0", "2", "*", "*", "*", "/usr/bin/backup.sh"})
		assert.NoError(t, err, "Cron add should execute without error")
	})

	t.Run("CronRemove_Execution", func(t *testing.T) {
		err := runCronRemove(cronRemoveCmd, []string{"root", "/usr/bin/backup.sh"})
		assert.NoError(t, err, "Cron remove should execute without error")
	})
}

// TestSaltCLI_MonitorCommands_Functional tests monitor commands execution
func TestSaltCLI_MonitorCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("MonitorLoad_Execution", func(t *testing.T) {
		err := runMonitorLoad(monitorLoadCmd, []string{})
		assert.NoError(t, err, "Monitor load should execute without error")
	})

	t.Run("MonitorIO_Execution", func(t *testing.T) {
		err := runMonitorIO(monitorIOCmd, []string{})
		assert.NoError(t, err, "Monitor IO should execute without error")
	})

	t.Run("MonitorNetIO_Execution", func(t *testing.T) {
		err := runMonitorNetIO(monitorNetIOCmd, []string{})
		assert.NoError(t, err, "Monitor NetIO should execute without error")
	})

	t.Run("MonitorInfo_Execution", func(t *testing.T) {
		err := runMonitorInfo(monitorInfoCmd, []string{})
		assert.NoError(t, err, "Monitor info should execute without error")
	})
}

// TestSaltCLI_ArchiveCommands_Functional tests archive commands execution
func TestSaltCLI_ArchiveCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("ArchiveTar_Execution", func(t *testing.T) {
		err := runArchiveTar(archiveTarCmd, []string{"/tmp/source", "/tmp/archive.tar.gz"})
		assert.NoError(t, err, "Archive tar should execute without error")
	})

	t.Run("ArchiveUntar_Execution", func(t *testing.T) {
		err := runArchiveUntar(archiveUntarCmd, []string{"/tmp/archive.tar.gz", "/tmp/dest"})
		assert.NoError(t, err, "Archive untar should execute without error")
	})

	t.Run("ArchiveZip_Execution", func(t *testing.T) {
		err := runArchiveZip(archiveZipCmd, []string{"/tmp/source", "/tmp/archive.zip"})
		assert.NoError(t, err, "Archive zip should execute without error")
	})

	t.Run("ArchiveUnzip_Execution", func(t *testing.T) {
		err := runArchiveUnzip(archiveUnzipCmd, []string{"/tmp/archive.zip", "/tmp/dest"})
		assert.NoError(t, err, "Archive unzip should execute without error")
	})
}

// TestSaltCLI_SSHCommands_Functional tests SSH commands execution
func TestSaltCLI_SSHCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("SSHKeyGen_Execution", func(t *testing.T) {
		err := runSSHKeyGen(sshKeyGenCmd, []string{"ubuntu", "ed25519"})
		assert.NoError(t, err, "SSH keygen should execute without error")
	})

	t.Run("SSHAuthKeys_Execution", func(t *testing.T) {
		err := runSSHAuthKeys(sshAuthKeysCmd, []string{"ubuntu"})
		assert.NoError(t, err, "SSH authkeys should execute without error")
	})

	t.Run("SSHSetKey_Execution", func(t *testing.T) {
		err := runSSHSetKey(sshSetKeyCmd, []string{"ubuntu", "ssh-rsa AAAA..."})
		assert.NoError(t, err, "SSH setkey should execute without error")
	})
}

// TestSaltCLI_GitCommands_Functional tests git commands execution
func TestSaltCLI_GitCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("GitClone_Execution", func(t *testing.T) {
		err := runGitClone(gitCloneCmd, []string{"https://github.com/test/repo.git", "/opt/repo"})
		assert.NoError(t, err, "Git clone should execute without error")
	})

	t.Run("GitPull_Execution", func(t *testing.T) {
		err := runGitPull(gitPullCmd, []string{"/opt/repo"})
		assert.NoError(t, err, "Git pull should execute without error")
	})
}

// TestSaltCLI_K8sCommands_Functional tests kubernetes commands execution
func TestSaltCLI_K8sCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("K8sGet_Execution", func(t *testing.T) {
		err := runK8sGet(k8sGetCmd, []string{"pods"})
		assert.NoError(t, err, "K8s get should execute without error")
	})

	t.Run("K8sApply_Execution", func(t *testing.T) {
		err := runK8sApply(k8sApplyCmd, []string{"/tmp/manifest.yaml"})
		assert.NoError(t, err, "K8s apply should execute without error")
	})

	t.Run("K8sDelete_Execution", func(t *testing.T) {
		err := runK8sDelete(k8sDeleteCmd, []string{"deployment", "nginx"})
		assert.NoError(t, err, "K8s delete should execute without error")
	})
}

// TestSaltCLI_PillarCommands_Functional tests pillar commands execution
func TestSaltCLI_PillarCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("PillarGet_Execution", func(t *testing.T) {
		err := runPillarGet(pillarGetCmd, []string{"mykey"})
		assert.NoError(t, err, "Pillar get should execute without error")
	})

	t.Run("PillarList_Execution", func(t *testing.T) {
		err := runPillarList(pillarListCmd, []string{})
		assert.NoError(t, err, "Pillar list should execute without error")
	})
}

// TestSaltCLI_MountCommands_Functional tests mount commands execution
func TestSaltCLI_MountCommands_Functional(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	t.Run("MountList_Execution", func(t *testing.T) {
		err := runMountList(mountListCmd, []string{})
		assert.NoError(t, err, "Mount list should execute without error")
	})

	t.Run("MountMount_Execution", func(t *testing.T) {
		err := runMountMount(mountMountCmd, []string{"/dev/sdb1", "/mnt/data", "ext4"})
		assert.NoError(t, err, "Mount mount should execute without error")
	})

	t.Run("MountUmount_Execution", func(t *testing.T) {
		err := runMountUnmount(mountUnmountCmd, []string{"/mnt/data"})
		assert.NoError(t, err, "Mount umount should execute without error")
	})
}

// TestSaltCLI_JSONOutput tests JSON output mode for various commands
func TestSaltCLI_JSONOutput(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"
	saltOutputJSON = true

	t.Run("Ping_JSONOutput", func(t *testing.T) {
		err := runSaltPing(pingCmd, []string{})
		assert.NoError(t, err, "Ping with JSON output should work")
	})

	t.Run("Grains_JSONOutput", func(t *testing.T) {
		err := runSaltGrains(grainsCmd, []string{})
		assert.NoError(t, err, "Grains with JSON output should work")
	})

	t.Run("Cmd_JSONOutput", func(t *testing.T) {
		err := runSaltCmd(cmdCmd, []string{"hostname"})
		assert.NoError(t, err, "Cmd with JSON output should work")
	})

	// Reset
	saltOutputJSON = false
}

// TestSaltCLI_ErrorHandling tests error handling for various commands
func TestSaltCLI_ErrorHandling(t *testing.T) {
	// Test with no URL
	originalURL := saltAPIURL
	saltAPIURL = ""

	t.Run("MissingURL_Error", func(t *testing.T) {
		_, err := getSaltClient()
		require.Error(t, err, "Should error when API URL is missing")
		assert.Contains(t, err.Error(), "stack name is required")
	})

	// Restore
	saltAPIURL = originalURL
}

// TestSaltCLI_Targeting tests different targeting patterns
func TestSaltCLI_Targeting(t *testing.T) {
	server := setupComprehensiveMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"

	targets := []string{"*", "minion-1", "minion*", "worker*"}

	for _, target := range targets {
		t.Run("Target_"+target, func(t *testing.T) {
			saltTarget = target
			err := runSaltPing(pingCmd, []string{})
			assert.NoError(t, err, "Ping with target %s should work", target)
		})
	}
}
