package controller

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

type readinessCheck struct {
	Key      string `json:"key"`
	Category string `json:"category"`
	Label    string `json:"label"`
	Status   string `json:"status"`
	Detail   string `json:"detail"`
	Action   string `json:"action,omitempty"`
}

// GetProductionReadiness exposes a read-only production checklist for root users.
// It deliberately returns configuration state, not secrets or provider credentials.
func GetProductionReadiness(c *gin.Context) {
	checks := make([]readinessCheck, 0, 12)
	add := func(key, category, label, status, detail, action string) {
		checks = append(checks, readinessCheck{key, category, label, status, detail, action})
	}

	sessionSecret := strings.TrimSpace(os.Getenv("SESSION_SECRET"))
	switch {
	case sessionSecret == "" || sessionSecret == "random_string":
		add("session_secret", "security", "会话密钥", "fail", "未配置稳定的 SESSION_SECRET，重启或多实例部署会导致会话失效。", "设置一个至少 32 位的随机 SESSION_SECRET 并重启服务")
	case len(sessionSecret) < 32:
		add("session_secret", "security", "会话密钥", "warn", "SESSION_SECRET 长度低于 32 位。", "更换为至少 32 位的随机字符串")
	default:
		add("session_secret", "security", "会话密钥", "pass", "已配置稳定的会话密钥。", "")
	}

	if strings.TrimSpace(os.Getenv("CRYPTO_SECRET")) == "" {
		add("crypto_secret", "security", "加密密钥", "warn", "未单独配置 CRYPTO_SECRET，当前会复用会话密钥。", "生产环境建议单独设置 CRYPTO_SECRET")
	} else {
		add("crypto_secret", "security", "加密密钥", "pass", "已配置独立的加密密钥。", "")
	}

	var rootUser model.User
	rootErr := model.DB.Where("role = ?", common.RoleRootUser).First(&rootUser).Error
	if rootErr != nil {
		add("root_account", "security", "管理员账号", "fail", "没有找到 root 管理员账号。", "完成初始化并创建管理员账号")
	} else if common.ValidatePasswordAndHash("123456", rootUser.Password) {
		add("root_account", "security", "管理员账号", "fail", "管理员仍在使用默认密码 123456。", "立即修改管理员密码")
	} else {
		add("root_account", "security", "管理员账号", "pass", "管理员密码不是默认值。", "")
	}

	if err := model.PingDB(); err != nil {
		add("database", "stability", "数据库连接", "fail", "数据库连接失败。", "检查 SQL_DSN、SQLite 文件权限和数据库服务")
	} else {
		add("database", "stability", "数据库连接", "pass", "数据库连接正常。", "")
	}

	if common.RedisEnabled {
		add("redis", "stability", "Redis 缓存", "pass", "Redis 已连接，适合多用户和多实例场景。", "")
	} else {
		add("redis", "stability", "Redis 缓存", "warn", "Redis 未连接，单机可运行，但限流和多实例一致性能力受限。", "配置 REDIS_CONN_STRING 并重启服务")
	}

	if common.GlobalApiRateLimitEnable && common.GlobalWebRateLimitEnable && common.CriticalRateLimitEnable {
		add("rate_limits", "security", "请求限流", "pass", "全局、网页和关键操作限流均已开启。", "")
	} else {
		add("rate_limits", "security", "请求限流", "fail", "至少有一类请求限流未开启。", "开启 GLOBAL_API_RATE_LIMIT_ENABLE、GLOBAL_WEB_RATE_LIMIT_ENABLE 和 CRITICAL_RATE_LIMIT_ENABLE")
	}

	if !common.RegisterEnabled && !common.PasswordRegisterEnabled {
		add("registration_policy", "security", "注册策略", "pass", "公开注册已关闭，适合封闭测试和邀请制运营。", "")
	} else {
		add("registration_policy", "security", "注册策略", "warn", "公开注册仍处于开启状态，可能导致批量注册和额度滥用。", "封闭测试阶段关闭 RegisterEnabled 和 PasswordRegisterEnabled")
	}

	var enabledChannels int64
	model.DB.Model(&model.Channel{}).Where("status = ?", common.ChannelStatusEnabled).Count(&enabledChannels)
	if enabledChannels == 0 {
		add("channels", "stability", "启用渠道", "fail", "当前没有启用的模型渠道。", "在渠道管理中添加并测试至少一个渠道")
	} else {
		add("channels", "stability", "启用渠道", "pass", "当前有启用的模型渠道。", "")
	}

	var enabledChannelList []model.Channel
	model.DB.Where("status = ?", common.ChannelStatusEnabled).Find(&enabledChannelList)
	providerPresent := map[string]bool{"GPT": false, "Claude": false, "Grok": false}
	providerCounts := map[string]int{"GPT": 0, "Claude": 0, "Grok": 0}
	for _, channel := range enabledChannelList {
		name := strings.ToLower(channel.Name)
		models := strings.ToLower(channel.Models)
		switch {
		case channel.Type == constant.ChannelTypeAnthropic || strings.Contains(name, "anthropic") || strings.Contains(name, "claude") || strings.Contains(models, "claude"):
			providerPresent["Claude"] = true
			providerCounts["Claude"]++
		case channel.Type == constant.ChannelTypeXai || strings.Contains(name, "xai") || strings.Contains(name, "grok") || strings.Contains(models, "grok"):
			providerPresent["Grok"] = true
			providerCounts["Grok"]++
		case strings.Contains(name, "openai") || strings.Contains(name, "gpt") || strings.Contains(models, "gpt"):
			providerPresent["GPT"] = true
			providerCounts["GPT"]++
		}
	}
	missingProviders := make([]string, 0, len(providerPresent))
	configuredProviders := make([]string, 0, len(providerPresent))
	for _, provider := range []string{"GPT", "Claude", "Grok"} {
		if providerPresent[provider] {
			configuredProviders = append(configuredProviders, provider)
		} else {
			missingProviders = append(missingProviders, provider)
		}
	}
	if len(configuredProviders) == 0 {
		add("primary_providers", "stability", "GPT / Claude / Grok 渠道", "warn", "尚未发现启用的 GPT、Claude 或 Grok 模型渠道。", "至少配置并测试一个真实模型供应商")
	} else if len(missingProviders) > 0 {
		add("primary_providers", "stability", "GPT / Claude / Grok 渠道", "warn", "已配置："+strings.Join(configuredProviders, "、")+"；缺少："+strings.Join(missingProviders, "、")+"。", "补齐缺失供应商并完成真实请求测试")
	} else {
		weak := make([]string, 0, 3)
		for _, provider := range []string{"GPT", "Claude", "Grok"} {
			if providerCounts[provider] < 2 {
				weak = append(weak, provider+"仅"+strconv.Itoa(providerCounts[provider])+"个")
			}
		}
		if len(weak) > 0 {
			add("primary_providers", "stability", "GPT / Claude / Grok 渠道", "warn", "三类供应商均已启用，但备用渠道不足："+strings.Join(weak, "、")+"。", "每个主供应商至少配置两个不同密钥或区域的启用渠道")
		} else {
			add("primary_providers", "stability", "GPT / Claude / Grok 渠道", "pass", "GPT、Claude、Grok 均有主渠道和备用渠道。", "")
		}
	}

	abilities, abilityErr := model.GetAllEnableAbilityWithChannels()
	missingPrices := make([]string, 0)
	seenModels := make(map[string]bool)
	if abilityErr == nil {
		for _, ability := range abilities {
			name := strings.TrimSpace(ability.Model)
			if name == "" || seenModels[name] {
				continue
			}
			seenModels[name] = true
			if _, _, exists := ratio_setting.GetModelRatioOrPrice(name); !exists {
				missingPrices = append(missingPrices, name)
			}
		}
	}
	if abilityErr != nil {
		add("model_prices", "billing", "模型价格", "fail", "无法读取启用模型或价格配置。", "检查模型价格配置和数据库")
	} else if len(missingPrices) > 0 {
		detail := "有启用模型未配置价格，可能被计费策略拦截。"
		if len(missingPrices) <= 3 {
			detail += " 缺失：" + strings.Join(missingPrices, ", ")
		}
		add("model_prices", "billing", "模型价格", "fail", detail, "在模型价格页面补齐模型倍率或价格")
	} else {
		add("model_prices", "billing", "模型价格", "pass", "所有启用模型均有价格或倍率配置。", "")
	}

	if common.LogConsumeEnabled {
		add("usage_logging", "billing", "用量与计费日志", "pass", "消费日志已开启。", "")
	} else {
		add("usage_logging", "billing", "用量与计费日志", "fail", "消费日志已关闭，无法可靠核对用量和费用。", "开启 LogConsumeEnabled")
	}

	paymentConfigured := operation_setting.PayAddress != "" && operation_setting.EpayId != "" && operation_setting.EpayKey != ""
	stripeConfigured := setting.StripeApiSecret != "" && setting.StripeWebhookSecret != "" && setting.StripePriceId != ""
	creemConfigured := setting.CreemApiKey != "" && strings.TrimSpace(setting.CreemProducts) != "" && strings.TrimSpace(setting.CreemProducts) != "[]"
	if paymentConfigured || stripeConfigured || creemConfigured {
		add("payments", "billing", "支付通道", "pass", "至少有一个支付通道已配置。", "上线收费前完成支付回调、幂等和对账测试")
	} else {
		add("payments", "billing", "支付通道", "warn", "当前没有配置支付通道，适合封闭测试，但不能直接开始收费运营。", "正式收费前配置 Stripe、Creem 或兼容易支付，并完成回调测试")
	}

	var errorLogs int64
	lastDay := time.Now().Add(-24 * time.Hour).Unix()
	model.LOG_DB.Model(&model.Log{}).Where("type = ? AND created_at >= ?", model.LogTypeError, lastDay).Count(&errorLogs)
	if errorLogs > 20 {
		add("error_rate", "operations", "错误告警", "warn", "过去 24 小时错误日志超过 20 条。", "检查日志中的渠道、模型和上游错误并设置告警")
	} else {
		add("error_rate", "operations", "错误告警", "pass", "过去 24 小时错误日志处于可接受范围。", "")
	}

	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(system_setting.ServerAddress)), "https://") {
		add("https", "security", "HTTPS", "pass", "服务地址已配置为 HTTPS。", "")
	} else {
		add("https", "security", "HTTPS", "warn", "服务地址不是 HTTPS，公网部署时会暴露登录和 API Token。", "使用反向代理配置 HTTPS，并更新 ServerAddress")
	}

	docsLink := strings.TrimSpace(operation_setting.GetGeneralSetting().DocsLink)
	if docsLink == "" || docsLink == "https://docs.newapi.pro" {
		add("developer_docs", "operations", "开发者文档", "warn", "当前仍使用通用 New API 文档，用户看不到本服务的地址、模型、限流和计费规则。", "发布包含 Base URL、模型清单、鉴权、限流、计费和错误码的产品文档")
	} else {
		add("developer_docs", "operations", "开发者文档", "pass", "已配置产品专属开发者文档入口。", "")
	}

	add("backup", "operations", "数据库备份", "warn", "系统无法自动确认外部备份策略。", "为 SQLite 或 SQL 数据库配置每日备份，并定期演练恢复")

	var userCount int64
	model.DB.Model(&model.User{}).Count(&userCount)
	if userCount < 2 {
		add("multi_user", "security", "多用户与风控", "warn", "当前用户数少于 2，尚未完成多用户额度、限流和客服流程演练。", "管理员手动创建至少 2 个测试用户，验证额度隔离、Token 撤销和限流")
	} else {
		add("multi_user", "security", "多用户与风控", "pass", "已有多个用户，可进行额度隔离和风控演练。", "")
	}
	if supportURL := strings.TrimSpace(os.Getenv("SUPPORT_URL")); supportURL == "" {
		add("support", "operations", "客服入口", "warn", "未配置客服入口，用户遇到支付或模型故障时无法快速升级。", "设置 SUPPORT_URL 指向工单、邮箱或企业客服系统")
	} else {
		add("support", "operations", "客服入口", "pass", "已配置客服入口。", "")
	}

	passCount, warnCount, failCount := 0, 0, 0
	for _, check := range checks {
		switch check.Status {
		case "pass":
			passCount++
		case "warn":
			warnCount++
		case "fail":
			failCount++
		}
	}
	status := "pass"
	if warnCount > 0 {
		status = "warn"
	}
	if failCount > 0 {
		status = "fail"
	}

	categorySummary := map[string]map[string]int{}
	for _, check := range checks {
		if categorySummary[check.Category] == nil {
			categorySummary[check.Category] = map[string]int{"pass": 0, "warn": 0, "fail": 0}
		}
		categorySummary[check.Category][check.Status]++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"status": status,
			"checks": checks,
			"summary": gin.H{
				"total": len(checks), "pass": passCount, "warn": warnCount, "fail": failCount,
			},
			"categories":     categorySummary,
			"generated_at":   time.Now().Unix(),
			"operation_mode": operation_setting.SelfUseModeEnabled,
		},
	})
}
