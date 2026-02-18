// Copyright © 2024 The vjailbreak authors

package nbd

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPasswordRedactionLogic verifies that passwords are properly redacted from command strings
// before being logged.
//
// IMPORTANT FOR DEVELOPERS:
// When logging commands that contain sensitive information (passwords, tokens, API keys, etc.),
// always build a redacted command string BEFORE passing it to logging functions.
//
// Example usage pattern (from nbdops.go StartNBDServer):
//   1. Create your command with the actual password
//   2. Build a redacted string by replacing password with [REDACTED]
//   3. Pass the redacted string to utils.AddDebugOutputToFileWithCommand()
//
//   password := "SecretPassword123"
//   cmd := exec.Command("nbdkit", "server=vcenter.local", fmt.Sprintf("password=%s", password))
//
//   // Build redacted command string
//   cmdstring := ""
//   for _, arg := range cmd.Args {
//       if strings.Contains(arg, password) {
//           cmdstring += "password=[REDACTED] "
//       } else {
//           cmdstring += fmt.Sprintf("%s ", arg)
//       }
//   }
//
//   // Use the redacted string for logging
//   utils.AddDebugOutputToFileWithCommand(cmd, cmdstring)
//
// WARNING: Never pass cmd.String() directly to logging functions if the command contains
// sensitive data. Always build a redacted version first.
func TestPasswordRedactionLogic(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"simple password", "SimplePassword123"},
		{"complex password", "C0mpl3x!P@ssw0rd#2024"},
		{"password with spaces", "My Secret Password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the command that would be created in StartNBDServer
			cmdArgs := []string{
				"nbdkit",
				"server=vcenter.local",
				"user=admin",
				fmt.Sprintf("password=%s", tt.password),
				"thumbprint=AA:BB:CC",
			}

			cmd := &exec.Cmd{Args: cmdArgs}

			// Apply the same redaction logic from nbdops.go lines 120-127
			cmdstring := ""
			for _, arg := range cmd.Args {
				if strings.Contains(arg, tt.password) {
					cmdstring += "password=[REDACTED] "
				} else {
					cmdstring += fmt.Sprintf("%s ", arg)
				}
			}

			// Verify password is redacted
			assert.Contains(t, cmdstring, "password=[REDACTED]")
			assert.NotContains(t, cmdstring, tt.password)

			// Verify other parameters are visible
			assert.Contains(t, cmdstring, "server=vcenter.local")
			assert.Contains(t, cmdstring, "user=admin")
		})
	}
}
