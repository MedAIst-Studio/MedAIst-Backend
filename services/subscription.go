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
	logger domain.Logger
}

func NewSubscriptionService(
	repo domain.SubscriptionRepository,
	limits domain.UsageLimitsConfig,
	logger domain.Logger,
) *SubscriptionService {
	return &SubscriptionService{
		repo:   repo,
		limits: limits,
		logger: logger,
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
		fileCount, err := s.repo.GetUserPDFCount(ctx, userID)
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
	eventType, _ := eventData["type"].(string)

	appUserID, ok := eventData["app_user_id"].(string)
	if !ok || appUserID == "" {
		s.logger.Error("Webhook event missing app_user_id", "type", eventType)
		return fmt.Errorf("webhook event missing app_user_id")
	}

	userID, err := uuid.Parse(appUserID)
	if err != nil {
		s.logger.Error("Invalid app_user_id", "app_user_id", appUserID, "err", err)
		return fmt.Errorf("invalid app_user_id: %w", err)
	}

	var now time.Time
	if tsVal, ok := eventData["event_timestamp_ms"].(float64); ok {
		now = time.UnixMilli(int64(tsVal))
	} else {
		now = time.Now()
	}

	switch eventType {
	case "INITIAL_PURCHASE", "RENEWAL", "PRODUCT_CHANGE", "UNCANCELLATION":
		productID, _ := eventData["product_id"].(string)
		entitlementID := s.detectEntitlement(productID)
		planType := s.determinePlanType(productID)

		var expiresAt *time.Time
		if expVal, ok := eventData["expiration_at_ms"]; ok {
			if expFloat, ok := expVal.(float64); ok {
				t := time.UnixMilli(int64(expFloat))
				expiresAt = &t
			} else {
				s.logger.Error("expiration_at_ms is not float64", "value", expVal)
			}
		}

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

		if err := s.repo.UpsertSubscription(ctx, sub); err != nil {
			s.logger.Error("Failed to upsert subscription", "userID", userID, "err", err)
			return err
		}

		return nil

	case "CANCELLATION":
		sub, err := s.repo.GetSubscription(ctx, userID)
		if err != nil {
			s.logger.Error("Failed to get subscription for cancellation", "userID", userID, "err", err)
			return err
		}
		if sub == nil {
			s.logger.Info("No subscription found for cancellation, skipping", "userID", userID)
			return nil
		}

		sub.WillRenew = false
		sub.UpdatedAt = now

		if err := s.repo.UpsertSubscription(ctx, sub); err != nil {
			s.logger.Error("Failed to update subscription for cancellation", "userID", userID, "err", err)
			return err
		}

		return nil

	case "EXPIRATION":
		sub, err := s.repo.GetSubscription(ctx, userID)
		if err != nil {
			s.logger.Error("Failed to get subscription for expiration", "userID", userID, "err", err)
			return err
		}
		if sub == nil {
			s.logger.Info("No subscription found for expiration, skipping", "userID", userID)
			return nil
		}

		sub.PlanType = domain.PlanFree
		sub.IsActive = true
		sub.WillRenew = true
		sub.UpdatedAt = now

		if err := s.repo.UpsertSubscription(ctx, sub); err != nil {
			s.logger.Error("Failed to update subscription to free after expiration", "userID", userID, "err", err)
			return err
		}

		return nil

	default:
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

		if action == domain.ActionFileUpload {
			count, err = s.repo.GetUserPDFCount(ctx, userID)
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

	periodLabel := "daily"
	if planLimits.Period == domain.LimitMonthly {
		periodLabel = "monthly"
	}

	if sub.PlanType == domain.PlanProSemester || sub.ProductID == domain.ProductProSemester || sub.ProductID == domain.ProductUltraSemester {
		periodLabel = "semester"
	} else if sub.PlanType == domain.PlanProYearly || sub.ProductID == domain.ProductProAnnual || sub.ProductID == domain.ProductUltraAnnual {
		periodLabel = "yearly"
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
