package plugin_magefile

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/magefile/mage/sh"
)

// Cmd represents a command to be executed
type Cmd struct {
	env            map[string]string
	namespace      string
	target         string
	workingDir     string
	beforeCommands []string // Commands to run before the main command
}

// NewCmd creates a new command with the given namespace and target
func NewCmd(namespace, target string, env map[string]string) *Cmd {
	if env == nil {
		env = make(map[string]string)
	}
	return &Cmd{
		env:       env,
		namespace: namespace,
		target:    target,
	}
}

// WorkingDir sets the working directory for the command
func (c *Cmd) WorkingDir(dir string) *Cmd {
	c.workingDir = dir
	return c
}

// RunBefore adds a command to be executed before the main command
func (c *Cmd) RunBefore(command string) *Cmd {
	c.beforeCommands = append(c.beforeCommands, command)
	return c
}

// Run executes the command with the given arguments
func (c *Cmd) Run(cmd string, args ...string) error {
	stdout := NewLogWriter(c.namespace, c.target, slog.LevelDebug)
	stderr := NewLogWriter(c.namespace, c.target, slog.LevelError)

	// Build shell command that includes before commands and cd
	var commands []string
	if c.workingDir != "" {
		commands = append([]string{fmt.Sprintf("cd %s", c.workingDir)}, commands...)
	}

	// Add any before commands
	commands = append(commands, c.beforeCommands...)

	// Add the main command
	commands = append(commands, fmt.Sprintf("%s %s", cmd, strings.Join(args, " ")))

	// Join all commands with &&
	shellCmd := strings.Join(commands, " && ")

	// Use user's default shell or fallback to sh
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}

	Logger.Debug("Running command",
		"namespace", c.namespace,
		"target", c.target,
		"command", shellCmd)

	ok, err := sh.Exec(c.env, stdout, stderr, shell, "-c", shellCmd)

	if err != nil || !ok {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

// CheckCommand checks if a command exists on the system
func CheckCommand(name string) bool {
	// Use user's default shell or fallback to sh
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}

	cmd := NewCmd("", "", nil)
	// Use command -v which is POSIX compliant
	return cmd.Run("command", "-v", name) == nil
}
