package model

import "time"

type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
	LogLevelFatal LogLevel = "FATAL"
)

// LogEntry represents a single log message received via WebSocket
type LogEntry struct {
	AppID     string   `json:"app_id"`
	Level     LogLevel `json:"level"`
	Message   string   `json:"message"`
	Timestamp string   `json:"timestamp"`
	Source    string   `json:"source,omitempty"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// LogFile represents a log file on disk
type LogFile struct {
	AppID    string    `json:"app_id"`
	Date     string    `json:"date"`
	FileName string    `json:"file_name"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mod_time"`
}

// LogQueryRequest represents a request to query log files
type LogQueryRequest struct {
	AppID    string `json:"app_id"`
	Date     string `json:"date"`
	Keyword  string `json:"keyword,omitempty"`
	Level    string `json:"level,omitempty"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

// LogQueryResponse represents the response of a log query
type LogQueryResponse struct {
	Lines      []LogLine `json:"lines"`
	Total      int       `json:"total"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalPages int       `json:"total_pages"`
}

// LogLine represents a single line in a log file
type LogLine struct {
	LineNumber int    `json:"line_number"`
	Content    string `json:"content"`
	Level      string `json:"level,omitempty"`
	Timestamp  string `json:"timestamp,omitempty"`
}

// AppStats represents statistics for an application
type AppStats struct {
	AppID       string   `json:"app_id"`
	AppName     string   `json:"app_name"`
	Description string   `json:"description"`
	TotalFiles  int      `json:"total_files"`
	TotalSize   int64    `json:"total_size"`
	DateRange   []string `json:"date_range"`
	Online      bool     `json:"online"`
}

// WSMessage represents a WebSocket message from client
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// WSResponse represents a WebSocket response to client
type WSResponse struct {
	Type    string      `json:"type"`
	Code    int         `json:"code"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// APIResponse is the standard API response wrapper
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

// DashboardStats represents dashboard summary statistics
type DashboardStats struct {
	TotalApps      int    `json:"total_apps"`
	OnlineApps     int    `json:"online_apps"`
	TotalLogFiles  int    `json:"total_log_files"`
	TotalLogSize   int64  `json:"total_log_size"`
	TodayLogCount  int    `json:"today_log_count"`
	TotalLogSizeStr string `json:"total_log_size_str"`
}
