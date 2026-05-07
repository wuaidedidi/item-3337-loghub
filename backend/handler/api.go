package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"loghub/config"
	"loghub/model"
	"loghub/service"

	"go.uber.org/zap"
)

// APIHandler handles REST API requests
type APIHandler struct {
	logStore *service.LogStore
	hub      *WSHub
	logger   *zap.Logger
	cfg      *config.Config
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(store *service.LogStore, hub *WSHub, logger *zap.Logger) *APIHandler {
	return &APIHandler{
		logStore: store,
		hub:      hub,
		logger:   logger,
		cfg:      config.Get(),
	}
}

// HandleLogin processes login requests
func (h *APIHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, "请求方法不允许", nil)
		return
	}

	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, "请求参数格式不正确", nil)
		return
	}

	if req.Username == "" || req.Password == "" {
		respondJSON(w, http.StatusBadRequest, "用户名和密码不能为空", nil)
		return
	}

	cfg := config.Get()
	if req.Username != cfg.Auth.Username || req.Password != cfg.Auth.Password {
		respondJSON(w, http.StatusUnauthorized, "用户名或密码错误", nil)
		return
	}

	token := generateToken(cfg.Auth.JWTSecret, req.Username)

	respondJSON(w, http.StatusOK, "登录成功", model.LoginResponse{
		Token:    token,
		Username: req.Username,
	})
}

// HandleGetDashboard returns dashboard statistics
func (h *APIHandler) HandleGetDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, "请求方法不允许", nil)
		return
	}

	stats, err := h.logStore.GetDashboardStats(h.hub.GetOnlineAppCount())
	if err != nil {
		h.logger.Error("failed to get dashboard stats", zap.Error(err))
		respondJSON(w, http.StatusInternalServerError, "获取统计数据失败", nil)
		return
	}

	respondJSON(w, http.StatusOK, "success", stats)
}

// HandleGetApps returns the list of configured applications
func (h *APIHandler) HandleGetApps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, "请求方法不允许", nil)
		return
	}

	appStats, err := h.logStore.GetAppList()
	if err != nil {
		h.logger.Error("failed to get app list", zap.Error(err))
		respondJSON(w, http.StatusInternalServerError, "获取应用列表失败", nil)
		return
	}

	onlineApps := h.hub.GetOnlineApps()
	onlineSet := make(map[string]bool)
	for _, id := range onlineApps {
		onlineSet[id] = true
	}

	for i := range appStats {
		appStats[i].Online = onlineSet[appStats[i].AppID]
	}

	respondJSON(w, http.StatusOK, "success", appStats)
}

// HandleGetLogFiles returns log files for a specific application
func (h *APIHandler) HandleGetLogFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, "请求方法不允许", nil)
		return
	}

	appID := r.URL.Query().Get("app_id")
	if appID == "" {
		respondJSON(w, http.StatusBadRequest, "app_id 参数不能为空", nil)
		return
	}

	cfg := config.Get()
	if !cfg.IsAppAllowed(appID) {
		respondJSON(w, http.StatusForbidden, "应用ID不在允许列表中", nil)
		return
	}

	files, err := h.logStore.GetLogFiles(appID)
	if err != nil {
		h.logger.Error("failed to get log files",
			zap.String("app_id", appID),
			zap.Error(err),
		)
		respondJSON(w, http.StatusInternalServerError, "获取日志文件列表失败", nil)
		return
	}

	respondJSON(w, http.StatusOK, "success", files)
}

// HandleQueryLogs queries log content with filtering and pagination
func (h *APIHandler) HandleQueryLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, "请求方法不允许", nil)
		return
	}

	appID := r.URL.Query().Get("app_id")
	date := r.URL.Query().Get("date")
	keyword := r.URL.Query().Get("keyword")
	level := r.URL.Query().Get("level")
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")

	if appID == "" {
		respondJSON(w, http.StatusBadRequest, "app_id 参数不能为空", nil)
		return
	}
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// Validate date format
	if _, err := time.Parse("2006-01-02", date); err != nil {
		respondJSON(w, http.StatusBadRequest, "日期格式不正确，应为 YYYY-MM-DD", nil)
		return
	}

	cfg := config.Get()
	if !cfg.IsAppAllowed(appID) {
		respondJSON(w, http.StatusForbidden, "应用ID不在允许列表中", nil)
		return
	}

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	req := &model.LogQueryRequest{
		AppID:    appID,
		Date:     date,
		Keyword:  keyword,
		Level:    level,
		Page:     page,
		PageSize: pageSize,
	}

	result, err := h.logStore.QueryLogs(req)
	if err != nil {
		h.logger.Error("failed to query logs",
			zap.String("app_id", appID),
			zap.Error(err),
		)
		respondJSON(w, http.StatusInternalServerError, "查询日志失败", nil)
		return
	}

	respondJSON(w, http.StatusOK, "success", result)
}

// HandleGetConfig returns the current configuration (safe fields only)
func (h *APIHandler) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, "请求方法不允许", nil)
		return
	}

	cfg := config.Get()
	safeConfig := map[string]interface{}{
		"apps":            cfg.Apps,
		"max_retain_days": cfg.Log.MaxRetainDays,
		"wss_port":        cfg.Server.WSSPort,
	}

	respondJSON(w, http.StatusOK, "success", safeConfig)
}

func respondJSON(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	code := statusCode
	if statusCode == http.StatusOK {
		code = 200
	}

	json.NewEncoder(w).Encode(model.APIResponse{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// Simple token generation using HMAC-like approach (no external JWT library needed)
func generateToken(secret, username string) string {
	payload := map[string]interface{}{
		"username": username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	payloadBytes, _ := json.Marshal(payload)

	// Simple base64-like encoding for the token
	encoded := encodeBase64URL(payloadBytes)
	sig := computeHMAC(encoded, secret)

	return encoded + "." + sig
}

// ValidateToken validates a token and returns the username
func ValidateToken(tokenStr, secret string) (string, bool) {
	parts := strings.SplitN(tokenStr, ".", 2)
	if len(parts) != 2 {
		return "", false
	}

	expectedSig := computeHMAC(parts[0], secret)
	if parts[1] != expectedSig {
		return "", false
	}

	payloadBytes, err := decodeBase64URL(parts[0])
	if err != nil {
		return "", false
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return "", false
	}

	exp, ok := payload["exp"].(float64)
	if !ok || time.Now().Unix() > int64(exp) {
		return "", false
	}

	username, ok := payload["username"].(string)
	if !ok {
		return "", false
	}

	return username, true
}

func encodeBase64URL(data []byte) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	result := make([]byte, 0, (len(data)*4+2)/3)

	for i := 0; i < len(data); i += 3 {
		var b uint32
		remaining := len(data) - i

		switch {
		case remaining >= 3:
			b = uint32(data[i])<<16 | uint32(data[i+1])<<8 | uint32(data[i+2])
			result = append(result, charset[b>>18&0x3F], charset[b>>12&0x3F], charset[b>>6&0x3F], charset[b&0x3F])
		case remaining == 2:
			b = uint32(data[i])<<16 | uint32(data[i+1])<<8
			result = append(result, charset[b>>18&0x3F], charset[b>>12&0x3F], charset[b>>6&0x3F])
		case remaining == 1:
			b = uint32(data[i]) << 16
			result = append(result, charset[b>>18&0x3F], charset[b>>12&0x3F])
		}
	}

	return string(result)
}

func decodeBase64URL(s string) ([]byte, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	lookup := make(map[byte]byte)
	for i := 0; i < len(charset); i++ {
		lookup[charset[i]] = byte(i)
	}

	// Save original length BEFORE padding to determine trim amount
	originalLen := len(s)

	// Add padding
	for len(s)%4 != 0 {
		s += "A"
	}

	result := make([]byte, 0, len(s)*3/4)
	for i := 0; i < len(s); i += 4 {
		var b uint32
		for j := 0; j < 4; j++ {
			v, ok := lookup[s[i+j]]
			if !ok {
				v = 0
			}
			b = b<<6 | uint32(v)
		}
		result = append(result, byte(b>>16), byte(b>>8), byte(b))
	}

	// Trim extra bytes based on how many padding chars were added
	switch originalLen % 4 {
	case 2:
		result = result[:len(result)-2]
	case 3:
		result = result[:len(result)-1]
	}

	return result, nil
}

func computeHMAC(message, key string) string {
	// Simple HMAC-like computation
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	keyBytes := []byte(key)
	msgBytes := []byte(message)

	// XOR-based mixing
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = keyBytes[i%len(keyBytes)]
	}

	for i, b := range msgBytes {
		hash[i%32] ^= b
		hash[(i+1)%32] = hash[(i+1)%32]<<1 | hash[(i+1)%32]>>7
		hash[(i+7)%32] ^= hash[i%32]
	}

	// Additional rounds of mixing
	for round := 0; round < 16; round++ {
		for i := 0; i < 32; i++ {
			hash[i] ^= hash[(i+13)%32]
			hash[(i+5)%32] = hash[(i+5)%32]<<3 | hash[(i+5)%32]>>5
		}
	}

	result := make([]byte, 43)
	for i := range result {
		result[i] = charset[hash[i%32]%64]
	}

	return string(result)
}
