package service

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"loghub/config"

	"go.uber.org/zap"
)

// CleanupService handles automatic deletion of expired log files
type CleanupService struct {
	baseDir  string
	logger   *zap.Logger
	stopChan chan struct{}
}

// NewCleanupService creates a new CleanupService instance
func NewCleanupService(baseDir string, logger *zap.Logger) *CleanupService {
	return &CleanupService{
		baseDir:  baseDir,
		logger:   logger,
		stopChan: make(chan struct{}),
	}
}

// Start begins the periodic cleanup routine
func (cs *CleanupService) Start() {
	cfg := config.Get()
	interval := time.Duration(cfg.Log.CleanInterval) * time.Minute

	cs.logger.Info("cleanup service started",
		zap.Int("max_retain_days", cfg.Log.MaxRetainDays),
		zap.Duration("interval", interval),
	)

	// Run an initial cleanup
	cs.cleanup()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cs.cleanup()
		case <-cs.stopChan:
			cs.logger.Info("cleanup service stopped")
			return
		}
	}
}

// Stop gracefully stops the cleanup service
func (cs *CleanupService) Stop() {
	close(cs.stopChan)
}

func (cs *CleanupService) cleanup() {
	cfg := config.Get()
	maxRetainDays := cfg.Log.MaxRetainDays
	cutoffDate := time.Now().AddDate(0, 0, -maxRetainDays)

	cs.logger.Info("running log cleanup",
		zap.String("cutoff_date", cutoffDate.Format("2006-01-02")),
		zap.Int("max_retain_days", maxRetainDays),
	)

	totalDeleted := 0
	var totalFreed int64

	for _, app := range cfg.Apps {
		appDir := filepath.Join(cs.baseDir, app.ID)
		entries, err := os.ReadDir(appDir)
		if err != nil {
			if !os.IsNotExist(err) {
				cs.logger.Warn("failed to read app directory",
					zap.String("app_id", app.ID),
					zap.Error(err),
				)
			}
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
				continue
			}

			dateStr := strings.TrimSuffix(entry.Name(), ".log")
			fileDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				cs.logger.Warn("skipping file with invalid date format",
					zap.String("file", entry.Name()),
					zap.Error(err),
				)
				continue
			}

			if fileDate.Before(cutoffDate) {
				filePath := filepath.Join(appDir, entry.Name())
				info, _ := entry.Info()
				var fileSize int64
				if info != nil {
					fileSize = info.Size()
				}

				if err := os.Remove(filePath); err != nil {
					cs.logger.Error("failed to delete expired log file",
						zap.String("file", filePath),
						zap.Error(err),
					)
					continue
				}

				totalDeleted++
				totalFreed += fileSize
				cs.logger.Info("deleted expired log file",
					zap.String("app_id", app.ID),
					zap.String("file", entry.Name()),
					zap.Int64("size", fileSize),
				)
			}
		}
	}

	cs.logger.Info("cleanup completed",
		zap.Int("files_deleted", totalDeleted),
		zap.String("space_freed", formatSize(totalFreed)),
	)
}
