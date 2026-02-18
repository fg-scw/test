package cli

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/platform9/vjailbreak/pkg/vpwned/upgrade"
	"github.com/spf13/cobra"
)

var (
	targetVersion string
	autoCleanup   bool
	jobMode       string
	prevVersion   string
)

var upgradeJobCmd = &cobra.Command{
	Use:   "upgrade-job",
	Short: "Run upgrade/rollback as a standalone job",
	Long:  "Execute vjailbreak upgrade or rollback as a Kubernetes Job. This command is meant to be called by the vpwned server.",
	Run: func(cmd *cobra.Command, args []string) {
		runUpgradeJob()
	},
}

func init() {
	rootCmd.AddCommand(upgradeJobCmd)

	upgradeJobCmd.Flags().StringVar(&targetVersion, "target-version", "", "Target version to upgrade to")
	upgradeJobCmd.Flags().BoolVar(&autoCleanup, "auto-cleanup", true, "Automatically cleanup resources if pre-checks fail")
	upgradeJobCmd.Flags().StringVar(&jobMode, "mode", "upgrade", "Job mode: 'upgrade' or 'rollback'")
	upgradeJobCmd.Flags().StringVar(&prevVersion, "previous-version", "", "Previous version for rollback (required for rollback mode)")
}

func runUpgradeJob() {
	if targetVersion == "" {
		targetVersion = os.Getenv("UPGRADE_TARGET_VERSION")
	}
	if prevVersion == "" {
		prevVersion = os.Getenv("UPGRADE_PREVIOUS_VERSION")
	}
	if os.Getenv("UPGRADE_AUTO_CLEANUP") == "true" {
		autoCleanup = true
	} else if os.Getenv("UPGRADE_AUTO_CLEANUP") == "false" {
		autoCleanup = false
	}
	if os.Getenv("UPGRADE_MODE") != "" {
		jobMode = os.Getenv("UPGRADE_MODE")
	}

	if targetVersion == "" {
		log.Fatal("target-version is required (via flag or UPGRADE_TARGET_VERSION env)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	executor, err := upgrade.NewUpgradeExecutor()
	if err != nil {
		log.Fatalf("Failed to create upgrade executor: %v", err)
	}

	switch jobMode {
	case "upgrade":
		log.Printf("Starting upgrade job to version %s (autoCleanup=%v)", targetVersion, autoCleanup)
		if err := executor.Execute(ctx, targetVersion, autoCleanup); err != nil {
			log.Fatalf("Upgrade failed: %v", err)
		}
		log.Printf("Upgrade job completed successfully")

	case "rollback":
		if prevVersion == "" {
			log.Fatal("previous-version is required for rollback mode")
		}
		backupID := os.Getenv("BACKUP_ID")
		log.Printf("Starting rollback job from %s to %s (backupID=%s)", targetVersion, prevVersion, backupID)
		if err := executor.ExecuteRollback(ctx, prevVersion, targetVersion, backupID); err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}
		log.Printf("Rollback job completed successfully")

	default:
		log.Fatalf("Unknown mode: %s (expected 'upgrade' or 'rollback')", jobMode)
	}
}
