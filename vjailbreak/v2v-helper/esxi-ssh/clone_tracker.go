// Copyright © 2024 The vjailbreak authors

package esxissh

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/platform9/vjailbreak/v2v-helper/pkg/utils"
)

// ProgressLogger is an interface for logging clone progress messages.
// This allows the migrate package to receive progress updates without circular imports.
type ProgressLogger interface {
	LogMessage(msg string)
}

// CloneStatus represents the current state of a clone operation
type CloneStatus struct {
	PID         int
	IsRunning   bool
	PercentDone float64
	ElapsedTime time.Duration
	Error       string
}

// CloneTracker monitors a vmkfstools clone operation in real-time
type CloneTracker struct {
	client            *Client
	task              *VmkfstoolsTask
	startTime         time.Time
	pollInterval      time.Duration
	startupTimeout    time.Duration
	stallTimeout      time.Duration
	lastLoggedPercent int
	lastProgress      float64
	lastProgressTime  time.Time
	logger            ProgressLogger
	diskIndex         int
}

// NewCloneTracker creates a new clone operation tracker.
// logger can be nil if no progress events are needed.
func NewCloneTracker(client *Client, task *VmkfstoolsTask, diskIndex int, logger ProgressLogger) *CloneTracker {
	now := time.Now()
	return &CloneTracker{
		client:            client,
		task:              task,
		startTime:         now,
		pollInterval:      10 * time.Second,
		startupTimeout:    5 * time.Minute,
		stallTimeout:      5 * time.Minute,
		lastLoggedPercent: -1,
		lastProgress:      -1,
		lastProgressTime:  now,
		logger:            logger,
		diskIndex:         diskIndex,
	}
}

// SetPollInterval sets how often to check clone status
func (ct *CloneTracker) SetPollInterval(interval time.Duration) {
	ct.pollInterval = interval
}

// SetStartupTimeout sets how long to wait for the process to actually start executing
func (ct *CloneTracker) SetStartupTimeout(timeout time.Duration) {
	ct.startupTimeout = timeout
}

// GetStatus returns the current status of the clone operation
func (ct *CloneTracker) GetStatus(ctx context.Context) (*CloneStatus, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	status := &CloneStatus{
		PID:         ct.task.Pid,
		ElapsedTime: time.Since(ct.startTime),
	}

	// Read log file content
	logContent := ct.readLogFile()

	// Parse state from log content
	status.PercentDone = ct.parseProgress(logContent)
	status.Error = ct.parseError(logContent)
	status.IsRunning = ct.determineIfRunning(logContent, status.PercentDone, status.Error)

	// Log progress at 5% increments
	ct.logProgressIfNeeded(status.PercentDone)

	return status, nil
}

// readLogFile reads the vmkfstools log file content
func (ct *CloneTracker) readLogFile() string {
	if ct.task.LogFile == "" {
		return ""
	}
	logCmd := fmt.Sprintf("cat %s 2>/dev/null", ct.task.LogFile)
	content, _ := ct.client.ExecuteCommand(logCmd)
	return strings.TrimSpace(content)
}

// parseProgress extracts the highest percentage from log content
// vmkfstools uses \r (carriage return) to overwrite progress lines in place
func (ct *CloneTracker) parseProgress(logContent string) float64 {
	if logContent == "" {
		return 0
	}

	var maxPct float64 = 0

	// Replace \r with \n to handle vmkfstools progress output
	// vmkfstools writes: "Clone: 0% done.\rClone: 1% done.\r..."
	normalized := strings.ReplaceAll(logContent, "\r", "\n")

	for _, line := range strings.Split(normalized, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var pct float64
		// Try with period at end (most common format)
		if _, err := fmt.Sscanf(line, "Clone: %f%% done.", &pct); err == nil {
			if pct > maxPct {
				maxPct = pct
			}
			continue
		}
		// Try without period
		if _, err := fmt.Sscanf(line, "Clone: %f%% done", &pct); err == nil {
			if pct > maxPct {
				maxPct = pct
			}
		}
	}

	return maxPct
}

// parseError checks log content for error messages
func (ct *CloneTracker) parseError(logContent string) string {
	if logContent == "" {
		return ""
	}
	// Check for failure indicators
	lowerLog := strings.ToLower(logContent)
	if strings.Contains(lowerLog, "failed") || strings.Contains(lowerLog, "error") {
		return fmt.Sprintf("vmkfstools failed: %s", logContent)
	}
	return ""
}

// determineIfRunning determines if the clone is still running based on all available signals
func (ct *CloneTracker) determineIfRunning(logContent string, percentDone float64, errorMsg string) bool {
	// If there's an error, not running
	if errorMsg != "" {
		return false
	}

	// If 100% done, not running (completed)
	if percentDone >= 100 {
		return false
	}

	// Track progress changes to detect stalled clones
	now := time.Now()
	if percentDone > ct.lastProgress {
		ct.lastProgress = percentDone
		ct.lastProgressTime = now
	}

	// Check if process is visible
	processVisible, _ := ct.client.CheckCloneStatus(ct.task.Pid)
	if processVisible {
		return true
	}

	// Process not visible - check if clone is stalled
	elapsed := time.Since(ct.startTime)
	timeSinceProgress := now.Sub(ct.lastProgressTime)

	// If we have progress but it's stalled (no change for stallTimeout), clone likely failed
	if percentDone > 0 && percentDone < 100 && timeSinceProgress > ct.stallTimeout {
		utils.PrintLog(fmt.Sprintf("WARNING: Clone appears stalled: no progress change for %v (stuck at %.0f%%)", timeSinceProgress.Round(time.Second), percentDone))
		return false
	}

	// If log shows progress (but not 100%) and not stalled, clone is running even if process not visible
	if percentDone > 0 {
		return true
	}

	// If log is empty and within startup timeout, assume still starting
	if logContent == "" && elapsed < ct.startupTimeout {
		return true
	}

	// Process not visible, no progress, startup timeout exceeded
	return false
}

// logProgressIfNeeded logs progress at 5% increments
func (ct *CloneTracker) logProgressIfNeeded(percentDone float64) {
	currentBucket := (int(percentDone) / 5) * 5
	if currentBucket > ct.lastLoggedPercent {
		msg := fmt.Sprintf("Copying disk %d, Completed: %d%%", ct.diskIndex, currentBucket)
		utils.PrintLog(msg)
		if ct.logger != nil {
			ct.logger.LogMessage(msg)
		}
		ct.lastLoggedPercent = currentBucket
	}
}

// WaitForCompletion blocks until the clone completes, fails, or context is cancelled
func (ct *CloneTracker) WaitForCompletion(ctx context.Context) error {
	msg := fmt.Sprintf("Starting clone monitor for disk %d (PID %d)", ct.diskIndex, ct.task.Pid)
	utils.PrintLog(msg)
	if ct.logger != nil {
		ct.logger.LogMessage(msg)
	}

	ticker := time.NewTicker(ct.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			utils.PrintLog("Clone monitoring cancelled")
			return ctx.Err()

		case <-ticker.C:
			status, err := ct.GetStatus(ctx)
			if err != nil {
				return err
			}

			if !status.IsRunning {
				if status.Error != "" {
					return fmt.Errorf("clone failed: %s", status.Error)
				}
				msg := fmt.Sprintf("Disk %d clone completed successfully in %v", ct.diskIndex, status.ElapsedTime.Round(time.Second))
				utils.PrintLog(msg)
				return nil
			}
		}
	}
}
