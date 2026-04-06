-- 情报表增加质量评分相关字段
ALTER TABLE intelligence_items
ADD COLUMN IF NOT EXISTS quality_score NUMERIC(5,2) DEFAULT 0,
ADD COLUMN IF NOT EXISTS is_noise BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS filter_reason TEXT,
ADD COLUMN IF NOT EXISTS customer_tier VARCHAR(5),
ADD COLUMN IF NOT EXISTS signal_type TEXT,
ADD COLUMN IF NOT EXISTS pain_score NUMERIC(5,2) DEFAULT 0;

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_intel_quality_score ON intelligence_items(quality_score);
CREATE INDEX IF NOT EXISTS idx_intel_is_noise ON intelligence_items(is_noise);
CREATE INDEX IF NOT EXISTS idx_intel_customer_tier ON intelligence_items(customer_tier);
CREATE INDEX IF NOT EXISTS idx_intel_signal_type ON intelligence_items(signal_type);
