package upgrade

import "time"

// UpgradeProgress represents the progress of an upgrade or rollback operation.
type UpgradeProgress struct {
	CurrentStep      string            `json:"currentStep"`
	TotalSteps       int               `json:"totalSteps"`
	CompletedSteps   int               `json:"completedSteps"`
	Status           string            `json:"status"`
	Error            string            `json:"error,omitempty"`
	StartTime        time.Time         `json:"startTime"`
	EndTime          *time.Time        `json:"endTime,omitempty"`
	PreviousVersion  string            `json:"previousVersion"`
	TargetVersion    string            `json:"targetVersion"`
	BackupID         string            `json:"backupId,omitempty"`
	OriginalReplicas map[string]int32  `json:"originalReplicas,omitempty"`
	PhaseTimings     map[string]string `json:"phaseTimings,omitempty"`
	JobID            string            `json:"jobId,omitempty"`
	PodName          string            `json:"podName,omitempty"`
	Result           string            `json:"result,omitempty"`
}

const (
	StatusPending            = "pending"
	StatusInProgress         = "in_progress"
	StatusDeploying          = "deploying"
	StatusVerifyingStability = "verifying_stability"
	StatusCompleted          = "completed"
	StatusFailed             = "failed"
	StatusRollingBack        = "rolling_back"
	StatusRolledBack         = "rolled_back"
	StatusRollbackFailed     = "rollback_failed"
	StatusUnknown            = "unknown"
)

const TotalUpgradeSteps = 11

const TotalRollbackSteps = 5
