package service

import (
	"fmt"
	"math/rand"
	"time"

	"loghub/config"
	"loghub/model"

	"go.uber.org/zap"
)

// SeedDemoData generates demo log data for all configured apps
func SeedDemoData(store *LogStore, logger *zap.Logger) {
	cfg := config.Get()

	logger.Info("seeding demo log data for configured applications")

	levels := []model.LogLevel{
		model.LogLevelDebug,
		model.LogLevelInfo,
		model.LogLevelInfo,
		model.LogLevelInfo,
		model.LogLevelWarn,
		model.LogLevelError,
	}

	messages := map[string][]string{
		"app-web-frontend": {
			"页面加载完成, 耗时 %dms",
			"用户点击了导航菜单",
			"API请求成功: GET /api/users",
			"组件渲染异常: Dashboard",
			"WebSocket连接建立成功",
			"静态资源缓存命中率: %d%%",
			"路由切换: /home -> /dashboard",
			"表单验证失败: 邮箱格式不正确",
			"用户登录成功: user_%d",
			"图片懒加载触发: banner_%d.png",
		},
		"app-api-gateway": {
			"请求路由: %s -> upstream:8080",
			"限流策略触发: IP 192.168.1.%d",
			"请求认证成功: Bearer token verified",
			"上游服务响应超时: order-service",
			"新增路由规则: /api/v2/*",
			"健康检查通过: all services healthy",
			"请求日志: POST /api/orders 201 %dms",
			"CORS预检请求处理: origin=localhost:3000",
			"负载均衡: 选择节点 node-%d",
			"SSL证书有效期: 剩余 %d 天",
		},
		"app-user-service": {
			"用户注册成功: user_%d@example.com",
			"密码重置邮件发送: user_%d",
			"Token刷新: 用户 %d",
			"用户资料更新: nickname changed",
			"登录失败: 密码错误, 剩余尝试次数 %d",
			"用户角色变更: user_%d -> admin",
			"会话过期清理: 清理了 %d 个过期会话",
			"用户头像上传: size=%dKB",
			"双因素认证启用: user_%d",
			"批量用户导入: 成功 %d 条",
		},
		"app-order-service": {
			"订单创建成功: ORDER_%d",
			"支付回调处理: order_%d status=PAID",
			"库存检查: product_%d remaining=%d",
			"订单状态变更: PENDING -> PROCESSING",
			"发货通知发送: order_%d",
			"退款申请: order_%d amount=%.2f",
			"订单超时取消: order_%d",
			"促销活动应用: coupon_%d discount=%d%%",
			"订单导出任务完成: 共 %d 条记录",
			"物流信息更新: tracking=%s",
		},
		"app-payment-service": {
			"支付请求: amount=%.2f currency=CNY",
			"支付渠道选择: alipay",
			"交易签名验证通过: txn_%d",
			"退款处理完成: refund_%d",
			"对账任务开始: date=%s",
			"支付成功通知: order_%d",
			"风控检查: 交易金额异常, amount=%.2f",
			"渠道余额查询: balance=%.2f",
			"批量结算完成: 共 %d 笔",
			"支付超时: txn_%d, 已重试 %d 次",
		},
	}

	sources := map[string][]string{
		"app-web-frontend":   {"Router", "APIClient", "WebSocket", "Component", "Store"},
		"app-api-gateway":    {"Proxy", "RateLimiter", "Auth", "HealthCheck", "LoadBalancer"},
		"app-user-service":   {"UserController", "AuthService", "TokenManager", "ProfileService", "SessionManager"},
		"app-order-service":  {"OrderController", "PaymentCallback", "InventoryService", "ShippingService", "ExportService"},
		"app-payment-service": {"PaymentGateway", "RefundService", "ReconcileJob", "RiskControl", "SettlementService"},
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for _, app := range cfg.Apps {
		appMessages, ok := messages[app.ID]
		if !ok {
			continue
		}
		appSources, ok := sources[app.ID]
		if !ok {
			continue
		}

		// Generate logs for the last 5 days
		for dayOffset := 4; dayOffset >= 0; dayOffset-- {
			date := time.Now().AddDate(0, 0, -dayOffset)
			logCount := 20 + r.Intn(30) // 20-50 logs per day

			for i := 0; i < logCount; i++ {
				hour := r.Intn(24)
				minute := r.Intn(60)
				second := r.Intn(60)
				ts := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, second, 0, time.Local)

				level := levels[r.Intn(len(levels))]
				msgTemplate := appMessages[r.Intn(len(appMessages))]
				source := appSources[r.Intn(len(appSources))]

				msg := formatMessage(msgTemplate, r)

				entry := &model.LogEntry{
					AppID:     app.ID,
					Level:     level,
					Message:   msg,
					Timestamp: ts.Format(time.RFC3339),
					Source:    source,
				}

				if err := store.WriteLog(entry); err != nil {
					logger.Warn("failed to seed log entry",
						zap.String("app_id", app.ID),
						zap.Error(err),
					)
				}
			}
		}

		logger.Info("seeded demo data for app",
			zap.String("app_id", app.ID),
			zap.String("app_name", app.Name),
		)
	}

	logger.Info("demo data seeding completed")
}

func formatMessage(template string, r *rand.Rand) string {
	args := make([]interface{}, 0)
	i := 0
	for i < len(template) {
		if template[i] == '%' && i+1 < len(template) {
			if template[i+1] == 'd' {
				args = append(args, r.Intn(9000)+1000)
				i += 2
				continue
			} else if template[i+1] == 's' {
				strs := []string{"GET /api/users", "POST /api/orders", "PUT /api/settings", "SF1234567890", "2024-01-15"}
				args = append(args, strs[r.Intn(len(strs))])
				i += 2
				continue
			} else if template[i+1] == '.' {
				// Handle %.2f style format
				args = append(args, r.Float64()*10000)
				// Skip past the full format specifier
				j := i + 2
				for j < len(template) && template[j] != 'f' && j-i < 8 {
					j++
				}
				if j < len(template) && template[j] == 'f' {
					i = j + 1
				} else {
					i += 2
				}
				continue
			}
		}
		i++
	}

	if len(args) == 0 {
		return template
	}

	return fmt.Sprintf(template, args...)
}
