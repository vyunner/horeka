CREATE TABLE IF NOT EXISTS sms_codes (
                                         id BIGSERIAL PRIMARY KEY,
                                         phone TEXT NOT NULL,
                                         code TEXT NOT NULL,
                                         created_at TIMESTAMPTZ NOT NULL DEFAULT now()
    );

CREATE INDEX IF NOT EXISTS idx_sms_codes_phone_created ON sms_codes(phone, created_at DESC);
