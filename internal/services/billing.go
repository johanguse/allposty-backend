package services

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/allposty/allposty-backend/internal/config"
	"github.com/allposty/allposty-backend/internal/models"
	"github.com/allposty/allposty-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/billingportal/session"
	stripecheckout "github.com/stripe/stripe-go/v78/checkout/session"
	"github.com/stripe/stripe-go/v78/customer"
	"github.com/stripe/stripe-go/v78/webhook"
)

var ErrSubscriptionNotFound = errors.New("subscription not found")

// PlanLimits defines server-enforced constraints per tier.
// -1 means unlimited.
var PlanLimits = map[string]PlanLimit{
	"free":   {Workspaces: 1, SocialAccounts: 3, PostsPerMonth: 30, AIEnabled: false},
	"pro":    {Workspaces: 3, SocialAccounts: 15, PostsPerMonth: -1, AIEnabled: true},
	"agency": {Workspaces: -1, SocialAccounts: -1, PostsPerMonth: -1, AIEnabled: true},
}

type PlanLimit struct {
	Workspaces     int
	SocialAccounts int
	PostsPerMonth  int
	AIEnabled      bool
}

type BillingService struct {
	orgs  *repository.OrgRepository
	users *repository.UserRepository
	subs  *repository.SubscriptionRepository
	cfg   *config.Config
}

func NewBillingService(
	orgs *repository.OrgRepository,
	users *repository.UserRepository,
	subs *repository.SubscriptionRepository,
	cfg *config.Config,
) *BillingService {
	stripe.Key = cfg.Stripe.SecretKey
	return &BillingService{orgs: orgs, users: users, subs: subs, cfg: cfg}
}

// CreateCheckoutSession returns a Stripe Checkout URL for upgrading.
func (s *BillingService) CreateCheckoutSession(userID, orgID uuid.UUID, tier, successURL, cancelURL string) (string, error) {
	org, err := s.orgs.FindOrgByID(orgID)
	if err != nil {
		return "", ErrOrgNotFound
	}
	if org.OwnerID != userID {
		return "", ErrForbidden
	}

	priceID := s.priceIDForTier(tier)
	if priceID == "" {
		return "", fmt.Errorf("unknown tier: %s", tier)
	}

	customerID, err := s.getOrCreateCustomer(userID, orgID, org.Name)
	if err != nil {
		return "", err
	}

	sess, err := stripecheckout.New(&stripe.CheckoutSessionParams{
		Customer:   stripe.String(customerID),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{Price: stripe.String(priceID), Quantity: stripe.Int64(1)},
		},
		Metadata: map[string]string{
			"org_id": orgID.String(),
			"tier":   tier,
		},
	})
	if err != nil {
		return "", fmt.Errorf("stripe: checkout: %w", err)
	}
	return sess.URL, nil
}

// CreatePortalSession returns a Stripe Billing Portal URL.
func (s *BillingService) CreatePortalSession(userID, orgID uuid.UUID, returnURL string) (string, error) {
	org, err := s.orgs.FindOrgByID(orgID)
	if err != nil {
		return "", ErrOrgNotFound
	}
	if org.OwnerID != userID {
		return "", ErrForbidden
	}

	sub, err := s.subs.FindByOrg(orgID)
	if err != nil || sub.StripeCustomerID == "" {
		return "", ErrSubscriptionNotFound
	}

	sess, err := session.New(&stripe.BillingPortalSessionParams{
		Customer:  stripe.String(sub.StripeCustomerID),
		ReturnURL: stripe.String(returnURL),
	})
	if err != nil {
		return "", fmt.Errorf("stripe: portal: %w", err)
	}
	return sess.URL, nil
}

// HandleWebhook processes signed Stripe webhook events.
func (s *BillingService) HandleWebhook(payload []byte, signature string) error {
	event, err := webhook.ConstructEvent(payload, signature, s.cfg.Stripe.WebhookSecret)
	if err != nil {
		return fmt.Errorf("webhook signature failed: %w", err)
	}

	switch event.Type {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			return err
		}
		return s.onCheckoutCompleted(&sess)

	case "customer.subscription.updated":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return err
		}
		return s.onSubscriptionUpdated(&sub)

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return err
		}
		return s.onSubscriptionDeleted(&sub)
	}
	return nil
}

func (s *BillingService) GetPlanLimit(tier string) PlanLimit {
	if limit, ok := PlanLimits[tier]; ok {
		return limit
	}
	return PlanLimits["free"]
}

// --- private ---

func (s *BillingService) getOrCreateCustomer(userID, orgID uuid.UUID, orgName string) (string, error) {
	sub, _ := s.subs.FindByOrg(orgID)
	if sub != nil && sub.StripeCustomerID != "" {
		return sub.StripeCustomerID, nil
	}

	user, err := s.users.FindByID(userID)
	if err != nil {
		return "", err
	}

	cust, err := customer.New(&stripe.CustomerParams{
		Email: stripe.String(user.Email),
		Name:  stripe.String(orgName),
		Metadata: map[string]string{
			"org_id":  orgID.String(),
			"user_id": userID.String(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("stripe: create customer: %w", err)
	}
	return cust.ID, nil
}

func (s *BillingService) onCheckoutCompleted(sess *stripe.CheckoutSession) error {
	orgIDStr := sess.Metadata["org_id"]
	tier := sess.Metadata["tier"]
	if orgIDStr == "" || tier == "" {
		return nil
	}

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		return err
	}

	newSub := &models.Subscription{
		OrganizationID:   orgID,
		StripeCustomerID: sess.Customer.ID,
		StripeSubID:      sess.Subscription.ID,
		Tier:             tier,
		Status:           "active",
	}

	existing, _ := s.subs.FindByOrg(orgID)
	if existing != nil {
		existing.StripeSubID = newSub.StripeSubID
		existing.StripeCustomerID = newSub.StripeCustomerID
		existing.Tier = newSub.Tier
		existing.Status = newSub.Status
		return s.subs.Update(existing)
	}
	return s.subs.Create(newSub)
}

func (s *BillingService) onSubscriptionUpdated(stripeSub *stripe.Subscription) error {
	sub, err := s.subs.FindByStripeSubID(stripeSub.ID)
	if err != nil {
		return nil
	}
	sub.Status = string(stripeSub.Status)
	sub.CurrentPeriodEnd = stripeSub.CurrentPeriodEnd
	return s.subs.Update(sub)
}

func (s *BillingService) onSubscriptionDeleted(stripeSub *stripe.Subscription) error {
	sub, err := s.subs.FindByStripeSubID(stripeSub.ID)
	if err != nil {
		return nil
	}
	sub.Status = "canceled"
	sub.Tier = "free"
	return s.subs.Update(sub)
}

func (s *BillingService) priceIDForTier(tier string) string {
	switch tier {
	case "pro":
		return s.cfg.Stripe.PricePro
	case "agency":
		return s.cfg.Stripe.PriceAgency
	}
	return ""
}
