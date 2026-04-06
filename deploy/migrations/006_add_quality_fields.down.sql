-- 回滚：删除质量评分相关字段
ALTER TABLE intelligence_items
DROP COLUMN IF EXISTS quality_score,
DROP COLUMN IF EXISTS is_noise,
DROP COLUMN IF EXISTS filter_reason,
DROP COLUMN IF EXISTS customer_tier,
DROP COLUMN IF EXISTS signal_type,
DROP COLUMN IF EXISTS pain_score;
