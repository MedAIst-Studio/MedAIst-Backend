package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ---------- Plan Types ----------

type PlanType string

const (
	PlanFree        PlanType = "free"
	PlanProMonthly  PlanType = "pro_monthly"
	PlanProSemester PlanType = "pro_semester"
	PlanProYearly   PlanType = "pro_yearly"
	PlanUltra       PlanType = "ultra"
)

// ---------- RevenueCat Product IDs ----------

const (
	ProductProMonthly    = "scope_pro_monthly"
	ProductProSemester   = "scope_pro_semester"
	ProductProAnnual     = "scope_pro_annual"
	ProductUltraMonthly  = "scope_ultra_monthly"
	ProductUltraSemester = "scope_ultra_semester"
	ProductUltraAnnual   = "scope_ultra_annual"

	EntitlementPro   = "pro"
	EntitlementUltra = "ultra"
)

// ---------- Limit Period ----------

type LimitPeriod string

const (
	LimitDaily   LimitPeriod = "daily"
	LimitMonthly LimitPeriod = "monthly"
)

// ---------- Action Types ----------

type ActionType string

const (
	ActionMessage    ActionType = "message"
	ActionOCR        ActionType = "ocr"
	ActionPDFChat    ActionType = "pdf_chat"
	ActionFileUpload ActionType = "file_upload"
	ActionQuiz       ActionType = "quiz"
)

// ---------- Usage Limits Configuration ----------

// PlanLimits defines the usage limits for a specific plan.
// A limit of -1 means unlimited.
type PlanLimits struct {
	Period   LimitPeriod
	Messages int // Normal soru/cevap (ucuz)
	OCR      int // Görüntü tarama (pahalı)
	PDFChat  int // PDF sohbet (pahalı)
	MaxFiles int // Toplam dosya sayısı limiti
	Quizzes  int
}

// UsageLimitsConfig holds limits for all plans. Easy to modify in one place.
type UsageLimitsConfig struct {
	Plans map[PlanType]PlanLimits
}

const UnlimitedUsage = -1

// DefaultUsageLimitsConfig returns the default usage limits configuration.
// Modify this function to change limits across the entire application.
func DefaultUsageLimitsConfig() UsageLimitsConfig {
	return UsageLimitsConfig{
		Plans: map[PlanType]PlanLimits{
			PlanFree: {
				Period:   LimitDaily,
				Messages: 10,
				OCR:      3,
				PDFChat:  5,
				MaxFiles: 3,
				Quizzes:  0,
			},
			PlanProMonthly: {
				Period:   LimitMonthly,
				Messages: 500,
				OCR:      100,
				PDFChat:  200,
				MaxFiles: 20,
				Quizzes:  200,
			},
			PlanProSemester: {
				Period:   LimitMonthly,
				Messages: 1000,
				OCR:      250,
				PDFChat:  500,
				MaxFiles: 50,
				Quizzes:  500,
			},
			PlanProYearly: {
				Period:   LimitMonthly,
				Messages: 1500,
				OCR:      400,
				PDFChat:  750,
				MaxFiles: 100,
				Quizzes:  1000,
			},
			PlanUltra: {
				Period:   LimitMonthly,
				Messages: UnlimitedUsage,
				OCR:      UnlimitedUsage,
				PDFChat:  UnlimitedUsage,
				MaxFiles: UnlimitedUsage,
				Quizzes:  UnlimitedUsage,
			},
		},
	}
}

// GetLimit returns the limit for a given action type in the plan.
func (pl PlanLimits) GetLimit(action ActionType) int {
	switch action {
	case ActionMessage:
		return pl.Messages
	case ActionOCR:
		return pl.OCR
	case ActionPDFChat:
		return pl.PDFChat
	case ActionFileUpload:
		return pl.MaxFiles
	case ActionQuiz:
		return pl.Quizzes
	default:
		return 0
	}
}

// ---------- Subscription Model ----------

type Subscription struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"userId"`
	PlanType      PlanType   `json:"planType"`
	RevenueCatID  string     `json:"revenueCatId,omitempty"`
	ProductID     string     `json:"productId,omitempty"`
	EntitlementID string     `json:"entitlementId,omitempty"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
	IsActive      bool       `json:"isActive"`
	WillRenew     bool       `json:"willRenew"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// ---------- Usage Record Model ----------

type UsageRecord struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"userId"`
	ActionType ActionType `json:"actionType"`
	UsageDate  time.Time  `json:"usageDate"`
	UsageCount int        `json:"usageCount"`
}

// ---------- Webhook Event ----------

type RevenueCatWebhookEvent struct {
	Event map[string]interface{} `json:"event"`
}

// ---------- DTOs ----------

type UsageStatusResponse struct {
	Plan   PlanType               `json:"plan"`
	Period string                 `json:"period"`
	Usage  map[string]UsageDetail `json:"usage"`
}

type UsageDetail struct {
	Used      int  `json:"used"`
	Limit     int  `json:"limit"`
	Unlimited bool `json:"unlimited"`
}

type GetUsageStatusRequest struct{}

// ---------- Repository Interface ----------

type SubscriptionRepository interface {
	GetSubscription(ctx context.Context, userID uuid.UUID) (*Subscription, error)
	UpsertSubscription(ctx context.Context, sub *Subscription) error
	EnsureFreeSubscription(ctx context.Context, userID uuid.UUID) error
	GetUsageCount(ctx context.Context, userID uuid.UUID, actionType ActionType, since time.Time) (int, error)
	IncrementUsage(ctx context.Context, userID uuid.UUID, actionType ActionType) error
	GetUserPDFCount(ctx context.Context, userID uuid.UUID) (int, error)
}

// ---------- Service Interface ----------

type SubscriptionService interface {
	CheckUsageLimit(ctx context.Context, userID uuid.UUID, actionType ActionType) (bool, error)
	IncrementUsage(ctx context.Context, userID uuid.UUID, actionType ActionType) error
	HandleWebhookEvent(ctx context.Context, event *RevenueCatWebhookEvent) error
	GetUsageStatus(ctx context.Context, userID uuid.UUID) (*UsageStatusResponse, error)
}
