package services

import (
	"context"
	"fmt"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type SubscriptionService struct {
	repo   domain.SubscriptionRepository
	limits domain.UsageLimitsConfig
}

func NewSubscriptionService(
	repo domain.SubscriptionRepository,
	limits domain.UsageLimitsConfig,
) *SubscriptionService {
	return &SubscriptionService{
		repo:   repo,
		limits: limits,
	}
}

// CheckUsageLimit verifies if the user still has remaining usage for the action.
// Returns true if the user can proceed.
func (s *SubscriptionService) CheckUsageLimit(ctx context.Context, userID uuid.UUID, actionType domain.ActionType) (bool, error) {
	if err := s.repo.EnsureFreeSubscription(ctx, userID); err != nil {
		return false, err
	}

	sub, err := s.repo.GetSubscription(ctx, userID)
	if err != nil {
		return false, err
	}
	if sub == nil {
		return false, domain.ErrSubscriptionNotFound
	}

	planLimits, exists := s.limits.Plans[sub.PlanType]
	if !exists {
		return false, nil
	}

	limit := planLimits.GetLimit(actionType)
	if limit == domain.UnlimitedUsage {
		return true, nil
	}

	// File uploads: check total file count, not periodic usage
	if actionType == domain.ActionFileUpload {
		fileCount, err := s.repo.GetUserFileCount(ctx, userID)
		if err != nil {
			return false, err
		}
		return fileCount < limit, nil
	}

	since := s.getStartOfPeriod(planLimits.Period)
	currentUsage, err := s.repo.GetUsageCount(ctx, userID, actionType, since)
	if err != nil {
		return false, err
	}

	return currentUsage < limit, nil
}

// IncrementUsage increments the usage counter for the action.
func (s *SubscriptionService) IncrementUsage(ctx context.Context, userID uuid.UUID, actionType domain.ActionType) error {
	return s.repo.IncrementUsage(ctx, userID, actionType)
}

// HandleWebhookEvent processes RevenueCat webhook events.
func (s *SubscriptionService) HandleWebhookEvent(ctx context.Context, event *domain.RevenueCatWebhookEvent) error {
	eventData := event.Event
	appUserID, _ := eventData["app_user_id"].(string)
	if appUserID == "" {
		return fmt.Errorf("webhook event missing app_user_id")
	}

	userID, err := uuid.Parse(appUserID)
	if err != nil {
		return fmt.Errorf("invalid app_user_id: %w", err)
	}

	now := time.Now()

	switch event.Type {
	case "INITIAL_PURCHASE", "RENEWAL", "PRODUCT_CHANGE", "UNCANCELLATION":
		productID, _ := eventData["product_id"].(string)
		entitlementID := s.detectEntitlement(productID)

		var expiresAt *time.Time
		if expStr, ok := eventData["expiration_at_ms"].(float64); ok {
			t := time.UnixMilli(int64(expStr))
			expiresAt = &t
		}

		planType := s.determinePlanType(productID)

		sub := &domain.Subscription{
			UserID:        userID,
			PlanType:      planType,
			RevenueCatID:  appUserID,
			ProductID:     productID,
			EntitlementID: entitlementID,
			ExpiresAt:     expiresAt,
			IsActive:      true,
			WillRenew:     true,
		}

		return s.repo.UpsertSubscription(ctx, sub)

	case "CANCELLATION":
		// İptal edildi ama expires_at'e kadar aktif kalır
		sub, err := s.repo.GetSubscription(ctx, userID)
		if err != nil {
			return err
		}
		if sub == nil {
			return nil
		}

		sub.WillRenew = false
		sub.UpdatedAt = now

		return s.repo.UpsertSubscription(ctx, sub)

	case "EXPIRATION":
		// Süre doldu — free plana düşür
		sub, err := s.repo.GetSubscription(ctx, userID)
		if err != nil {
			return err
		}
		if sub == nil {
			return nil
		}

		sub.PlanType = domain.PlanFree
		sub.IsActive = true
		sub.WillRenew = true
		sub.UpdatedAt = now

		return s.repo.UpsertSubscription(ctx, sub)

	default:
		// "BILLING_ISSUE", "SUBSCRIBER_ALIAS", etc. — no action needed
		return nil
	}
}

// GetUsageStatus returns the current usage status for a user.
func (s *SubscriptionService) GetUsageStatus(ctx context.Context, userID uuid.UUID) (*domain.UsageStatusResponse, error) {
	if err := s.repo.EnsureFreeSubscription(ctx, userID); err != nil {
		return nil, err
	}

	sub, err := s.repo.GetSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, domain.ErrSubscriptionNotFound
	}

	planLimits, exists := s.limits.Plans[sub.PlanType]
	if !exists {
		planLimits = s.limits.Plans[domain.PlanFree]
	}

	since := s.getStartOfPeriod(planLimits.Period)

	actions := []domain.ActionType{domain.ActionMessage, domain.ActionOCR, domain.ActionPDFChat, domain.ActionFileUpload, domain.ActionQuiz}
	usage := make(map[string]domain.UsageDetail)

	for _, action := range actions {
		var count int
		var err error

		// File uploads: show total file count
		if action == domain.ActionFileUpload {
			count, err = s.repo.GetUserFileCount(ctx, userID)
		} else {
			count, err = s.repo.GetUsageCount(ctx, userID, action, since)
		}
		if err != nil {
			return nil, err
		}

		limit := planLimits.GetLimit(action)
		usage[string(action)] = domain.UsageDetail{
			Used:      count,
			Limit:     limit,
			Unlimited: limit == domain.UnlimitedUsage,
		}
	}

	periodLabel := "günlük"
	if planLimits.Period == domain.LimitMonthly {
		periodLabel = "aylık"
	}

	return &domain.UsageStatusResponse{
		Plan:   sub.PlanType,
		Period: periodLabel,
		Usage:  usage,
	}, nil
}

// determinePlanType maps RevenueCat product IDs to internal plan types.
func (s *SubscriptionService) determinePlanType(productID string) domain.PlanType {
	switch productID {
	case domain.ProductProMonthly:
		return domain.PlanProMonthly
	case domain.ProductProSemester:
		return domain.PlanProSemester
	case domain.ProductProAnnual:
		return domain.PlanProYearly
	case domain.ProductUltraMonthly, domain.ProductUltraSemester, domain.ProductUltraAnnual:
		return domain.PlanUltra
	default:
		return domain.PlanFree
	}
}

// detectEntitlement returns the entitlement ID based on the product ID.
func (s *SubscriptionService) detectEntitlement(productID string) string {
	switch productID {
	case domain.ProductUltraMonthly, domain.ProductUltraSemester, domain.ProductUltraAnnual:
		return domain.EntitlementUltra
	default:
		return domain.EntitlementPro
	}
}

// getStartOfPeriod returns the start time for the limit period (today or start of month).
func (s *SubscriptionService) getStartOfPeriod(period domain.LimitPeriod) time.Time {
	now := time.Now()
	switch period {
	case domain.LimitMonthly:
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	default: // daily
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}
}
