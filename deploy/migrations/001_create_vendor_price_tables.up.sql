-- 厂商价格历史表迁移
-- Token Bridge Crawler 历史数据存储

-- 抓取快照主表
CREATE TABLE IF NOT EXISTS vendor_price_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor TEXT NOT NULL,                    -- 'google', 'openai', 'anthropic'
    snapshot_date DATE NOT NULL,             -- 抓取日期（2026-03-24）
    snapshot_at TIMESTAMPTZ NOT NULL,        -- 精确时间戳
    total_models INT NOT NULL DEFAULT 0,     -- 本次抓取模型数
    new_models INT NOT NULL DEFAULT 0,       -- 新增模型数
    updated_models INT NOT NULL DEFAULT 0,   -- 价格变动模型数
    removed_models INT NOT NULL DEFAULT 0,   -- 下架模型数
    raw_data_hash TEXT,                      -- 原始数据哈希（防重复存储）
    status TEXT NOT NULL DEFAULT 'success',  -- 'success' | 'partial' | 'failed'
    error_log TEXT,                          -- 失败时的错误信息
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 价格明细历史表（按模型按天）
CREATE TABLE IF NOT EXISTS vendor_price_details (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id UUID REFERENCES vendor_price_snapshots(id) ON DELETE CASCADE,
    vendor TEXT NOT NULL,
    model_code TEXT NOT NULL,
    snapshot_date DATE NOT NULL,
    
    -- 价格数据（保留原始精度）
    input_usd_per_million NUMERIC(12, 6),
    output_usd_per_million NUMERIC(12, 6),
    currency TEXT DEFAULT 'USD',
    
    -- 扩展字段（能力指标，JSONB 灵活扩展）
    capabilities JSONB,  -- {ttft_avg_ms, tps_throughput, context_window...}
    
    -- 变更标记
    change_type TEXT,  -- 'new' | 'updated' | 'unchanged' | 'removed'
    prev_price JSONB,  -- 变更前的价格 {input, output}
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_snapshots_vendor_date 
    ON vendor_price_snapshots(vendor, snapshot_date DESC);

CREATE INDEX IF NOT EXISTS idx_snapshots_status 
    ON vendor_price_snapshots(status);

CREATE INDEX IF NOT EXISTS idx_details_lookup 
    ON vendor_price_details(vendor, model_code, snapshot_date DESC);

CREATE INDEX IF NOT EXISTS idx_details_snapshot 
    ON vendor_price_details(snapshot_id);

CREATE INDEX IF NOT EXISTS idx_details_change_type 
    ON vendor_price_details(change_type) 
    WHERE change_type IN ('new', 'updated');

-- 注释
COMMENT ON TABLE vendor_price_snapshots IS '厂商价格抓取快照，每日一条记录';
COMMENT ON TABLE vendor_price_details IS '厂商价格明细历史，按模型按天记录';

COMMENT ON COLUMN vendor_price_snapshots.vendor IS '厂商标识: google, openai, anthropic';
COMMENT ON COLUMN vendor_price_snapshots.status IS '抓取状态: success, partial, failed';

COMMENT ON COLUMN vendor_price_details.change_type IS '变更类型: new(新增), updated(价格变动), unchanged(无变化), removed(下架)';
COMMENT ON COLUMN vendor_price_details.prev_price IS '变更前的价格，JSON格式 {input: x, output: y}';
