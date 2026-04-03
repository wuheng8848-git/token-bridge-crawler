-- 回滚：移除 vendor_price_details 的 upsert 唯一索引

DROP INDEX IF EXISTS uq_vendor_price_details_vendor_model_date;
