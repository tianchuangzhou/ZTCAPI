package model

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// PaymentEvent is the durable idempotency ledger for provider webhooks.
type PaymentEvent struct {
	Id          int        `json:"id" gorm:"primaryKey;autoIncrement"`
	Provider    string     `json:"provider" gorm:"type:varchar(32);uniqueIndex:uk_payment_event"`
	EventId     string     `json:"event_id" gorm:"type:varchar(255);uniqueIndex:uk_payment_event"`
	TradeNo     string     `json:"trade_no" gorm:"type:varchar(255);index"`
	Status      string     `json:"status" gorm:"type:varchar(32);index"`
	Attempts    int        `json:"attempts"`
	LastError   string     `json:"last_error" gorm:"type:text"`
	CreatedAt   time.Time  `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

const (
	PaymentEventProcessing = "processing"
	PaymentEventProcessed  = "processed"
	PaymentEventFailed     = "failed"
)

// BeginPaymentEvent atomically claims an event. Processed events are ignored;
// failed/processing events may be retried after a provider timeout.
func BeginPaymentEvent(provider, eventID, tradeNo string) (bool, error) {
	if provider == "" || eventID == "" {
		return true, nil // providers without event IDs use the order-level guard
	}
	var event PaymentEvent
	err := DB.Where("provider = ? AND event_id = ?", provider, eventID).First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		event = PaymentEvent{Provider: provider, EventId: eventID, TradeNo: tradeNo, Status: PaymentEventProcessing, Attempts: 1}
		if err := DB.Create(&event).Error; err != nil {
			// A concurrent webhook may have inserted the unique key.
			if DB.Where("provider = ? AND event_id = ?", provider, eventID).First(&event).Error == nil {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if event.Status == PaymentEventProcessed || event.Status == PaymentEventProcessing {
		return false, nil
	}
	DB.Model(&event).Updates(map[string]interface{}{
		"status":     PaymentEventProcessing,
		"attempts":   gorm.Expr("attempts + ?", 1),
		"last_error": "",
	})
	return true, nil
}

func FinishPaymentEvent(provider, eventID, status, message string) {
	if provider == "" || eventID == "" {
		return
	}
	updates := map[string]interface{}{"status": status, "last_error": message}
	if status == PaymentEventProcessed {
		now := time.Now()
		updates["processed_at"] = &now
	}
	DB.Model(&PaymentEvent{}).Where("provider = ? AND event_id = ?", provider, eventID).Updates(updates)
}

type PaymentRefund struct {
	Id          int        `json:"id" gorm:"primaryKey;autoIncrement"`
	TradeNo     string     `json:"trade_no" gorm:"uniqueIndex;type:varchar(255)"`
	UserId      int        `json:"user_id" gorm:"index"`
	Quota       int        `json:"quota"`
	Money       float64    `json:"money"`
	Provider    string     `json:"provider" gorm:"type:varchar(32)"`
	Status      string     `json:"status" gorm:"type:varchar(32);index"`
	Reason      string     `json:"reason" gorm:"type:text"`
	RequestedBy int        `json:"requested_by"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

const (
	RefundRequested = "requested"
	RefundCompleted = "completed"
)

func RequestRefund(tradeNo, reason, provider string, requestedBy int) (*PaymentRefund, error) {
	if tradeNo == "" {
		return nil, errors.New("订单号不能为空")
	}
	var refund PaymentRefund
	err := DB.Transaction(func(tx *gorm.DB) error {
		var topUp TopUp
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("trade_no = ?", tradeNo).First(&topUp).Error; err != nil {
			return errors.New("充值订单不存在")
		}
		if topUp.Status != common.TopUpStatusSuccess {
			return errors.New("只有成功订单可以申请退款")
		}
		if err := tx.Where("trade_no = ?", tradeNo).First(&refund).Error; err == nil {
			return errors.New("该订单已经存在退款记录")
		}
		quota := int(float64(topUp.Amount) * common.QuotaPerUnit)
		if topUp.PaymentMethod == "stripe" {
			quota = int(topUp.Money * common.QuotaPerUnit)
		}
		if provider == "" {
			provider = topUp.PaymentMethod
		}
		refund = PaymentRefund{TradeNo: tradeNo, UserId: topUp.UserId, Quota: quota, Money: topUp.Money, Provider: provider, Status: RefundRequested, Reason: reason, RequestedBy: requestedBy}
		return tx.Create(&refund).Error
	})
	return &refund, err
}

// CompleteRefund reverses quota exactly once. The external provider refund
// must be confirmed by an operator before calling this endpoint.
func CompleteRefund(tradeNo string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var refund PaymentRefund
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("trade_no = ?", tradeNo).First(&refund).Error; err != nil {
			return errors.New("退款记录不存在")
		}
		if refund.Status == RefundCompleted {
			return nil
		}
		if refund.Status != RefundRequested {
			return errors.New("退款状态不可完成")
		}
		result := tx.Model(&User{}).Where("id = ? AND quota >= ?", refund.UserId, refund.Quota).Update("quota", gorm.Expr("quota - ?", refund.Quota))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("用户余额不足，无法完成退款额度冲销")
		}
		now := time.Now()
		if err := tx.Model(&refund).Updates(map[string]interface{}{"status": RefundCompleted, "completed_at": &now}).Error; err != nil {
			return err
		}
		return nil
	})
}
