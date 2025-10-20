package common

import (
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// SSHExecutor centralizes SSH command execution across the project
type SSHExecutor struct {
	ctx        *pulumi.Context
	privateKey pulumi.StringOutput
}

// NewSSHExecutor creates a new SSH executor
func NewSSHExecutor(ctx *pulumi.Context, privateKey pulumi.StringOutput) *SSHExecutor {
	return &SSHExecutor{
		ctx:        ctx,
		privateKey: privateKey,
	}
}

// Execute runs a command on a remote host via SSH
func (e *SSHExecutor) Execute(name string, host pulumi.StringOutput, command string, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	return remote.NewCommand(e.ctx, name, &remote.CommandArgs{
		Connection: remote.ConnectionArgs{
			Host:       host,
			User:       pulumi.String("root"),
			PrivateKey: e.privateKey,
		},
		Create: pulumi.String(command),
	}, opts...)
}

// ExecuteWithRetry runs a command with retry logic
func (e *SSHExecutor) ExecuteWithRetry(name string, host pulumi.StringOutput, command string, retries int, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	retryCommand := BuildRetryCommand(command, retries)
	return e.Execute(name, host, retryCommand, opts...)
}

// BuildRetryCommand wraps a command with retry logic
func BuildRetryCommand(command string, retries int) string {
	if retries <= 1 {
		return command
	}

	return "for i in {1.." + string(rune(retries+48)) + "}; do " + command + " && break || sleep 10; done"
}
