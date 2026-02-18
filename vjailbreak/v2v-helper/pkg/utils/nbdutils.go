package utils

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/platform9/vjailbreak/v2v-helper/pkg/constants"
)

func ParseFraction(text string) (int, int, error) {
	var numerator, denominator int
	_, err := fmt.Sscanf(text, "%d/%d", &numerator, &denominator)
	if err != nil {
		return 0, 0, err
	}
	return numerator, denominator, nil
}

// migrationLogInfo tracks log files by migration name to ensure one file per migration attempt
type migrationLogInfo struct {
	filePath string
	file     *os.File
	refCount int
}

// We use a map to track migration log files
var migrationLogs = make(map[string]*migrationLogInfo)

// commandLogs maps commands to their migration name for cleanup
var commandLogs = make(map[*exec.Cmd]string)
var logMutex sync.Mutex

// CloseLogFile decreases reference count for a migration log file and closes it if no longer needed
func CloseLogFile(cmd *exec.Cmd) {
	logMutex.Lock()
	defer logMutex.Unlock()

	// Get the migration name associated with this command
	migrationName, exists := commandLogs[cmd]
	if !exists {
		return // No log file associated with this command
	}

	// Remove command tracking
	delete(commandLogs, cmd)

	// Update reference count for the migration log
	if logInfo, ok := migrationLogs[migrationName]; ok {
		logInfo.refCount--
		if logInfo.refCount <= 0 {
			// Last command using this log file, close it
			PrintLog(fmt.Sprintf("Closing migration log file: %s", logInfo.filePath))
			logInfo.file.Sync()
			logInfo.file.Close()
			delete(migrationLogs, migrationName)
		}
	}
}

// CleanupMigrationLogs forcibly closes a migration's log file regardless of reference count
// Call this when a migration is complete to clean up resources
func CleanupMigrationLogs(migrationName string) {
	logMutex.Lock()
	defer logMutex.Unlock()

	// If there's an open log file for this migration, close it
	if logInfo, exists := migrationLogs[migrationName]; exists {
		PrintLog(fmt.Sprintf("Force closing migration log file: %s", logInfo.filePath))
		logInfo.file.Sync()
		logInfo.file.Close()
		delete(migrationLogs, migrationName)
	}

	// Clean up any commands associated with this migration
	for cmd, name := range commandLogs {
		if name == migrationName {
			delete(commandLogs, cmd)
		}
	}
}

// RunCommandWithLogFile runs a command with output redirected to a log file
// and ensures the log file is properly closed after execution
// WARNING: This logs the full command including all arguments. Do NOT use this
// for commands containing sensitive data (passwords, tokens, etc.). Use
// RunCommandWithLogFileRedacted instead and pass a redacted command string.
func RunCommandWithLogFile(cmd *exec.Cmd) error {
	return RunCommandWithLogFileRedacted(cmd, cmd.String())
}

// RunCommandWithLogFileRedacted runs a command with output redirected to a log file
// using a redacted command string for logging, and ensures the log file is properly closed
func RunCommandWithLogFileRedacted(cmd *exec.Cmd, cmdString string) error {
	// First ensure output is directed to a log file with redacted command
	AddDebugOutputToFileWithCommand(cmd, cmdString)

	// Run the command
	err := cmd.Run()

	// Close the log file
	CloseLogFile(cmd)

	return err
}

// AddDebugOutputToFile redirects command output to a debug log file
// WARNING: This logs the full command including all arguments. Do NOT use this
// for commands containing sensitive data (passwords, tokens, etc.). Use
// AddDebugOutputToFileWithCommand instead and pass a redacted command string.
func AddDebugOutputToFile(cmd *exec.Cmd) {
	AddDebugOutputToFileWithCommand(cmd, cmd.String())
}

func AddDebugOutputToFileWithCommand(cmd *exec.Cmd, cmdString string) {
	migrationName, err := GetMigrationObjectName()
	if err != nil {
		return
	}

	// Ensure logs directory exists
	if err := os.MkdirAll(constants.LogsDir, 0755); err != nil {
		return
	}

	// Create a directory for this specific migration
	attemptDir := fmt.Sprintf("%s/%s", constants.LogsDir, migrationName)
	if err := os.MkdirAll(attemptDir, 0755); err != nil {
		return
	}

	logMutex.Lock()
	defer logMutex.Unlock()

	// First close any existing log file for this command
	// (do this within lock to avoid race conditions)
	if oldMigrationName, exists := commandLogs[cmd]; exists {
		// Remove command tracking
		delete(commandLogs, cmd)

		// Update reference count for the migration log
		if logInfo, ok := migrationLogs[oldMigrationName]; ok {
			logInfo.refCount--
			if logInfo.refCount <= 0 {
				// Last command using this log file, close it
				PrintLog(fmt.Sprintf("Closing migration log file: %s", logInfo.filePath))
				logInfo.file.Sync()
				logInfo.file.Close()
				delete(migrationLogs, oldMigrationName)
			}
		}
	}

	// Check if we already have an open log file for this migration
	var logFile *os.File
	logInfo, exists := migrationLogs[migrationName]
	if exists {
		// Reuse existing log file
		logFile = logInfo.file
		logInfo.refCount++ // Increment reference count
	} else {
		// Create a new log file with timestamp in name (only done once per migration)
		timestamp := time.Now().Format("2006-01-02-15:04:05")
		logFilePath := fmt.Sprintf("%s/migration.%s.log", attemptDir, timestamp)

		// Create/open the log file
		var err error
		logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return
		}

		// Store new log info
		migrationLogs[migrationName] = &migrationLogInfo{
			filePath: logFilePath,
			file:     logFile,
			refCount: 1,
		}

		// Write header to log file
		headerTime := time.Now().Format("2006-01-02 15:04:05")
		logFile.WriteString(fmt.Sprintf("==== MIGRATION LOG: %s - Started at %s ====\n\n",
			migrationName, headerTime))
	}

	// Associate this command with the migration name for later tracking
	commandLogs[cmd] = migrationName

	// Write a separator for this command to make logs more readable
	timePrefix := time.Now().Format("2006-01-02 15:04:05")
	logFile.WriteString(fmt.Sprintf("\n\n==== %s: COMMAND: %s ====\n\n", timePrefix, cmdString))

	// Redirect command output to this file
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	// NOTE: The log file will NOT be automatically closed when the command completes.
	// Either call CloseLogFile(cmd) manually after the command completes,
	// or use RunCommandWithLogFile() which handles the cleanup automatically.
}
