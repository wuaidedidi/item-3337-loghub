package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"loghub/config"
	"loghub/model"

	"go.uber.org/zap"
)

// LogStore manages log file storage for multiple applications
type LogStore struct {
	baseDir string
	mu      sync.RWMutex
	logger  *zap.Logger
}

// NewLogStore creates a new LogStore instance
func NewLogStore(baseDir string, logger *zap.Logger) (*LogStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log base directory: %w", err)
	}

	cfg := config.Get()
	for _, app := range cfg.Apps {
		appDir := filepath.Join(baseDir, app.ID)
		if err := os.MkdirAll(appDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create app log directory for %s: %w", app.ID, err)
		}
	}

	return &LogStore{
		baseDir: baseDir,
		logger:  logger,
	}, nil
}

// WriteLog writes a log entry to the appropriate file
func (ls *LogStore) WriteLog(entry *model.LogEntry) error {
	if entry.AppID == "" {
		return fmt.Errorf("app_id is required")
	}

	cfg := config.Get()
	if !cfg.IsAppAllowed(entry.AppID) {
		return fmt.Errorf("app_id %s is not allowed", entry.AppID)
	}

	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().Format(time.RFC3339)
	}
	if entry.Level == "" {
		entry.Level = model.LogLevelInfo
	}

	// Parse date from entry timestamp; fall back to today if parsing fails
	date := time.Now().Format("2006-01-02")
	if t, err := time.Parse(time.RFC3339, entry.Timestamp); err == nil {
		date = t.Format("2006-01-02")
	}
	appDir := filepath.Join(ls.baseDir, entry.AppID)
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("failed to create app directory: %w", err)
	}

	logFile := filepath.Join(appDir, date+".log")

	ls.mu.Lock()
	defer ls.mu.Unlock()

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	logLine := fmt.Sprintf("[%s] [%s] %s", entry.Timestamp, entry.Level, entry.Message)
	if entry.Source != "" {
		logLine = fmt.Sprintf("[%s] [%s] [%s] %s", entry.Timestamp, entry.Level, entry.Source, entry.Message)
	}
	if len(entry.Extra) > 0 {
		extraJSON, _ := json.Marshal(entry.Extra)
		logLine += " | " + string(extraJSON)
	}
	logLine += "\n"

	if _, err := f.WriteString(logLine); err != nil {
		return fmt.Errorf("failed to write log: %w", err)
	}

	return nil
}

// GetAppList returns a list of all applications with their stats
func (ls *LogStore) GetAppList() ([]model.AppStats, error) {
	cfg := config.Get()
	var stats []model.AppStats

	for _, app := range cfg.Apps {
		appDir := filepath.Join(ls.baseDir, app.ID)
		stat := model.AppStats{
			AppID:       app.ID,
			AppName:     app.Name,
			Description: app.Description,
			DateRange:   []string{},
		}

		entries, err := os.ReadDir(appDir)
		if err != nil {
			if os.IsNotExist(err) {
				stats = append(stats, stat)
				continue
			}
			return nil, fmt.Errorf("failed to read app directory %s: %w", app.ID, err)
		}

		var dates []string
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			stat.TotalFiles++
			stat.TotalSize += info.Size()
			date := strings.TrimSuffix(entry.Name(), ".log")
			dates = append(dates, date)
		}

		sort.Strings(dates)
		stat.DateRange = dates
		stats = append(stats, stat)
	}

	return stats, nil
}

// GetLogFiles returns log files for a specific application
func (ls *LogStore) GetLogFiles(appID string) ([]model.LogFile, error) {
	cfg := config.Get()
	if !cfg.IsAppAllowed(appID) {
		return nil, fmt.Errorf("app_id %s is not allowed", appID)
	}

	appDir := filepath.Join(ls.baseDir, appID)
	entries, err := os.ReadDir(appDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.LogFile{}, nil
		}
		return nil, fmt.Errorf("failed to read app directory: %w", err)
	}

	var files []model.LogFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		date := strings.TrimSuffix(entry.Name(), ".log")
		files = append(files, model.LogFile{
			AppID:    appID,
			Date:     date,
			FileName: entry.Name(),
			Size:     info.Size(),
			ModTime:  info.ModTime(),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Date > files[j].Date
	})

	return files, nil
}

// QueryLogs queries log content from a specific file with filtering and pagination
func (ls *LogStore) QueryLogs(req *model.LogQueryRequest) (*model.LogQueryResponse, error) {
	cfg := config.Get()
	if !cfg.IsAppAllowed(req.AppID) {
		return nil, fmt.Errorf("app_id %s is not allowed", req.AppID)
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 100
	}
	if req.PageSize > 500 {
		req.PageSize = 500
	}

	logFile := filepath.Join(ls.baseDir, req.AppID, req.Date+".log")
	f, err := os.Open(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &model.LogQueryResponse{
				Lines:    []model.LogLine{},
				Total:    0,
				Page:     req.Page,
				PageSize: req.PageSize,
			}, nil
		}
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	var allLines []model.LogLine
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if req.Keyword != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(req.Keyword)) {
			continue
		}

		if req.Level != "" && req.Level != "ALL" {
			levelTag := "[" + strings.ToUpper(req.Level) + "]"
			if !strings.Contains(line, levelTag) {
				continue
			}
		}

		logLine := model.LogLine{
			LineNumber: lineNum,
			Content:    line,
		}

		logLine.Level = extractLevel(line)
		logLine.Timestamp = extractTimestamp(line)

		allLines = append(allLines, logLine)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading log file: %w", err)
	}

	total := len(allLines)
	totalPages := (total + req.PageSize - 1) / req.PageSize
	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	return &model.LogQueryResponse{
		Lines:      allLines[start:end],
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetDashboardStats returns dashboard summary statistics
func (ls *LogStore) GetDashboardStats(onlineApps int) (*model.DashboardStats, error) {
	cfg := config.Get()
	stats := &model.DashboardStats{
		TotalApps:  len(cfg.Apps),
		OnlineApps: onlineApps,
	}

	today := time.Now().Format("2006-01-02")

	for _, app := range cfg.Apps {
		appDir := filepath.Join(ls.baseDir, app.ID)
		entries, err := os.ReadDir(appDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			stats.TotalLogFiles++
			stats.TotalLogSize += info.Size()

			if strings.TrimSuffix(entry.Name(), ".log") == today {
				stats.TodayLogCount++
			}
		}
	}

	stats.TotalLogSizeStr = formatSize(stats.TotalLogSize)
	return stats, nil
}

func extractLevel(line string) string {
	levels := []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}
	for _, level := range levels {
		if strings.Contains(line, "["+level+"]") {
			return level
		}
	}
	return "INFO"
}

func extractTimestamp(line string) string {
	if len(line) > 1 && line[0] == '[' {
		end := strings.Index(line, "]")
		if end > 0 {
			return line[1:end]
		}
	}
	return ""
}

func formatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
