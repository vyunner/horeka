CREATE TABLE sms_codes (
                           id BIGSERIAL PRIMARY KEY,
                           phone TEXT NOT NULL,
                           code TEXT NOT NULL,
                           created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                           used_at TIMESTAMPTZ
);

CREATE INDEX idx_sms_codes_phone_active
    ON sms_codes(phone, code)
    WHERE used_at IS NULL;

CREATE INDEX idx_sms_codes_used
    ON sms_codes(used_at)
    WHERE used_at IS NOT NULL;
