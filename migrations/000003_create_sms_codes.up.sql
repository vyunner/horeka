CREATE TABLE IF NOT EXISTS sms_codes (
                                         id BIGSERIAL PRIMARY KEY,
                                         phone TEXT NOT NULL,
                                         code TEXT NOT NULL,
                                         created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    used_at TIMESTAMPTZ
    );

-- индекс для быстрого поиска активного кода
CREATE INDEX IF NOT EXISTS idx_sms_codes_phone_active
    ON sms_codes(phone, code)
    WHERE used_at IS NULL;

-- индекс для чистки использованных
CREATE INDEX IF NOT EXISTS idx_sms_codes_used
    ON sms_codes(used_at)
    WHERE used_at IS NOT NULL;
