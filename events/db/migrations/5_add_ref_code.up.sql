-- Add column to payment table to track original amount before discount
ALTER TABLE payment ADD COLUMN original_amount INT;
ALTER TABLE payment ADD COLUMN discount_amount INT DEFAULT 0;

-- ALTER TABLE payment DROP COLUMN IF EXISTS discount_amount;
-- ALTER TABLE payment DROP COLUMN IF EXISTS original_amount;

CREATE TABLE referral_code (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(20) UNIQUE NOT NULL,
    discount_percentage INT NOT NULL CHECK (discount_percentage >= 0 AND discount_percentage <= 100),
    max_uses INT NOT NULL DEFAULT -1,
    current_uses INT NOT NULL DEFAULT 0,
    valid_from TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    valid_until TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX idx_referral_code_code ON referral_code(code);

CREATE TABLE referral_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    referral_code_id UUID NOT NULL REFERENCES referral_code(id),
    payment_id UUID NOT NULL REFERENCES payment(id),
    discount_amount INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

ALTER TABLE event ADD COLUMN disabled BOOLEAN DEFAULT false;
