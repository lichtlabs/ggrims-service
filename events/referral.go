package events

import (
	"context"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/types/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lichtlabs/ggrims-service/events/db"
)

type CreateReferralCodeRequest struct {
	Code            string    `json:"code"`
	DiscountPercent int       `json:"discount_percentage"`
	MaxUses         int       `json:"max_uses"`
	ValidUntil      time.Time `json:"valid_until,omitempty"`
}

type ReferralCodeResponse struct {
	ID              uuid.UUID  `json:"id"`
	Code            string     `json:"code"`
	DiscountPercent int        `json:"discount_percentage"`
	MaxUses         int        `json:"max_uses"`
	CurrentUses     int        `json:"current_uses"`
	ValidFrom       time.Time  `json:"valid_from"`
	ValidUntil      *time.Time `json:"valid_until,omitempty"`
}

//encore:api auth method=POST path=/v1/referral-codes
func CreateReferralCode(ctx context.Context, req *CreateReferralCodeRequest) (*BaseResponse[ReferralCodeResponse], error) {
	eb := errs.B()

	result, err := query.CreateReferralCode(ctx, db.CreateReferralCodeParams{
		Code:               req.Code,
		DiscountPercentage: int32(req.DiscountPercent),
		MaxUses:            int32(req.MaxUses),
		ValidUntil: pgtype.Timestamptz{
			Time:  req.ValidUntil,
			Valid: !req.ValidUntil.IsZero(),
		},
	})
	if err != nil {
		return nil, eb.Code(errs.Internal).Msg("failed to create referral code").Err()
	}

	return &BaseResponse[ReferralCodeResponse]{
		Data: ReferralCodeResponse{
			ID:              result.ID.Bytes,
			Code:            result.Code,
			DiscountPercent: int(result.DiscountPercentage),
			MaxUses:         int(result.MaxUses),
			CurrentUses:     int(result.CurrentUses),
			ValidFrom:       result.ValidFrom.Time,
			ValidUntil:      &result.ValidUntil.Time,
		},
		Message: "Referral code created successfully",
	}, nil
}

func validateReferralCode(ctx context.Context, code string) (*db.ReferralCode, error) {
	if code == "" {
		return nil, nil
	}

	refCode, err := query.GetReferralCodeByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	// Check if code is expired
	if refCode.ValidUntil.Valid && refCode.ValidUntil.Time.Before(time.Now()) {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("referral code has expired").Err()
	}

	// Check if max uses reached from referral_code table
	if refCode.MaxUses != -1 && int(refCode.CurrentUses) >= int(refCode.MaxUses) {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("referral code has reached maximum uses").Err()
	}

	// Check usage count from referral_usage table
	usageCount, err := query.GetReferralUsageCount(ctx, refCode.ID)
	if err != nil {
		return nil, err
	}

	// Validate against actual usage count
	if refCode.MaxUses != -1 && int32(usageCount) >= refCode.MaxUses {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("referral code has reached maximum uses (usage count)").Err()
	}

	return &refCode, nil
}
