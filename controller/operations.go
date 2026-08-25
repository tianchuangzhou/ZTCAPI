package controller

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type operationsRankRow struct {
	Name   string `json:"name"`
	Calls  int64  `json:"calls"`
	Tokens int64  `json:"tokens"`
	Quota  int64  `json:"quota"`
}

// GetOperationsReport returns privacy-safe aggregates for the admin console.
// It never returns prompts, API keys, IPs, or individual customer balances.
func GetOperationsReport(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days < 1 {
		days = 1
	}
	if days > 90 {
		days = 90
	}
	start := time.Now().Add(-time.Duration(days) * 24 * time.Hour).Unix()
	base := model.LOG_DB.Model(&model.Log{}).Where("type = ? AND created_at >= ?", model.LogTypeConsume, start)
	var total struct {
		Calls  int64 `json:"calls"`
		Tokens int64 `json:"tokens"`
		Quota  int64 `json:"quota"`
	}
	base.Select("count(*) as calls, coalesce(sum(prompt_tokens), 0) + coalesce(sum(completion_tokens), 0) as tokens, coalesce(sum(quota), 0) as quota").Scan(&total)
	var errorsCount int64
	model.LOG_DB.Model(&model.Log{}).Where("type = ? AND created_at >= ?", model.LogTypeError, start).Count(&errorsCount)
	var requestsCount int64
	model.LOG_DB.Model(&model.Log{}).Where("(type = ? OR type = ?) AND created_at >= ?", model.LogTypeConsume, model.LogTypeError, start).Count(&requestsCount)

	var models []operationsRankRow
	base.Select("model_name as name, count(*) as calls, coalesce(sum(prompt_tokens), 0) + coalesce(sum(completion_tokens), 0) as tokens, coalesce(sum(quota), 0) as quota").Group("model_name").Order("quota desc").Limit(10).Scan(&models)
	var users []operationsRankRow
	base.Select("username as name, count(*) as calls, coalesce(sum(prompt_tokens), 0) + coalesce(sum(completion_tokens), 0) as tokens, coalesce(sum(quota), 0) as quota").Group("username").Order("quota desc").Limit(10).Scan(&users)

	var pendingTopups, successfulTopups, expiredTopups int64
	model.DB.Model(&model.TopUp{}).Where("status = ?", common.TopUpStatusPending).Count(&pendingTopups)
	model.DB.Model(&model.TopUp{}).Where("status = ?", common.TopUpStatusSuccess).Count(&successfulTopups)
	model.DB.Model(&model.TopUp{}).Where("status = ?", common.TopUpStatusExpired).Count(&expiredTopups)
	var refundsRequested, refundsCompleted int64
	model.DB.Model(&model.PaymentRefund{}).Where("status = ?", model.RefundRequested).Count(&refundsRequested)
	model.DB.Model(&model.PaymentRefund{}).Where("status = ?", model.RefundCompleted).Count(&refundsCompleted)
	var eventFailures int64
	model.DB.Model(&model.PaymentEvent{}).Where("status = ?", model.PaymentEventFailed).Count(&eventFailures)

	errorRate := float64(0)
	if requestsCount > 0 {
		errorRate = float64(errorsCount) / float64(requestsCount)
	}
	c.JSON(200, gin.H{"success": true, "data": gin.H{
		"window_days":   days,
		"usage":         gin.H{"calls": total.Calls, "tokens": total.Tokens, "quota": total.Quota, "errors": errorsCount, "error_rate": errorRate},
		"model_ranking": models,
		"user_ranking":  users,
		"payments":      gin.H{"pending": pendingTopups, "successful": successfulTopups, "expired": expiredTopups, "refund_requested": refundsRequested, "refund_completed": refundsCompleted, "webhook_failures": eventFailures},
		"operations":    gin.H{"support_url": strings.TrimSpace(os.Getenv("SUPPORT_URL")), "registration_enabled": common.RegisterEnabled || common.PasswordRegisterEnabled},
	}})
}

type refundRequest struct {
	TradeNo  string `json:"trade_no"`
	Reason   string `json:"reason"`
	Provider string `json:"provider"`
}

func RequestPaymentRefund(c *gin.Context) {
	var req refundRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		common.ApiErrorMsg(c, "缺少订单号")
		return
	}
	refund, err := model.RequestRefund(strings.TrimSpace(req.TradeNo), strings.TrimSpace(req.Reason), strings.TrimSpace(req.Provider), c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, refund)
}

func CompletePaymentRefund(c *gin.Context) {
	var req refundRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		common.ApiErrorMsg(c, "缺少订单号")
		return
	}
	if err := model.CompleteRefund(strings.TrimSpace(req.TradeNo)); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetPaymentReconciliation(c *gin.Context) {
	var events []model.PaymentEvent
	model.DB.Order("id desc").Limit(100).Find(&events)
	var refunds []model.PaymentRefund
	model.DB.Order("id desc").Limit(100).Find(&refunds)
	c.JSON(200, gin.H{"success": true, "data": gin.H{"events": events, "refunds": refunds}})
}
