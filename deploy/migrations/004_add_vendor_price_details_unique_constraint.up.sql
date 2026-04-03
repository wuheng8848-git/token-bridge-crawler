-- 为 vendor_price_details 的 ON CONFLICT (vendor, model_code, snapshot_date) 补齐唯一约束
-- 背景：storage.SavePriceDetails 使用该冲突键做 upsert，缺失唯一索引会触发 SQLSTATE 42P10。

CREATE UNIQUE INDEX IF NOT EXISTS uq_vendor_price_details_vendor_model_date
    ON vendor_price_details(vendor, model_code, snapshot_date);
