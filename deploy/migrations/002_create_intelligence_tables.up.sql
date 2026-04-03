-- 情报系统统一存储表
-- 用于存储各种类型的情报数据：价格、API文档、社区、新闻等

-- 统一情报表
CREATE TABLE IF NOT EXISTS intelligence_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- 核心字段
    intel_type TEXT NOT NULL,           -- 'price', 'api_doc', 'community', 'news'
    source TEXT NOT NULL,               -- 'openai', 'anthropic', 'google', 'hackernews'
    source_id TEXT,                     -- 源系统ID（如 HN post id）
    
    -- 内容字段
    title TEXT,                         -- 标题
    content TEXT,                       -- 内容/摘要
    url TEXT,                           -- 原始链接
    
    -- 元数据（JSONB扩展）
    metadata JSONB DEFAULT '{}',        -- 类型特定的扩展数据
    
    -- 时间字段
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,           -- 原始发布时间
    
    -- 处理状态
    status TEXT DEFAULT 'new',          -- 'new', 'processed', 'alerted', 'ignored'
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_intel_type ON intelligence_items(intel_type);
CREATE INDEX IF NOT EXISTS idx_intel_source ON intelligence_items(source);
CREATE INDEX IF NOT EXISTS idx_intel_captured ON intelligence_items(captured_at DESC);
CREATE INDEX IF NOT EXISTS idx_intel_status ON intelligence_items(status);
CREATE INDEX IF NOT EXISTS idx_intel_published ON intelligence_items(published_at DESC);

-- 唯一约束：同一来源同一ID同一时间不重复
CREATE UNIQUE INDEX IF NOT EXISTS idx_intel_unique 
ON intelligence_items(source, source_id, captured_at) 
WHERE source_id IS NOT NULL;

-- 元数据GIN索引（用于JSONB查询）
CREATE INDEX IF NOT EXISTS idx_intel_metadata ON intelligence_items USING GIN (metadata);

-- 采集器运行日志表
CREATE TABLE IF NOT EXISTS collector_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    collector_name TEXT NOT NULL,
    intel_type TEXT NOT NULL,
    source TEXT NOT NULL,
    
    -- 运行状态
    status TEXT NOT NULL,               -- 'success', 'failed', 'partial'
    items_count INT DEFAULT 0,
    error_message TEXT,
    
    -- 性能指标
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    duration_ms INT,                    -- 执行耗时（毫秒）
    
    -- 策略信息
    strategy_used TEXT,                 -- 使用的策略：'web', 'api', 'static'
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_collector_runs_name ON collector_runs(collector_name);
CREATE INDEX IF NOT EXISTS idx_collector_runs_status ON collector_runs(status);
CREATE INDEX IF NOT EXISTS idx_collector_runs_started ON collector_runs(started_at DESC);

-- 告警规则表
CREATE TABLE IF NOT EXISTS alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    
    -- 触发条件
    intel_type TEXT,                    -- 为空表示所有类型
    condition TEXT NOT NULL,            -- 条件表达式
    severity TEXT NOT NULL,             -- 'critical', 'high', 'medium', 'low'
    
    -- 状态
    enabled BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 告警记录表
CREATE TABLE IF NOT EXISTS alert_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID REFERENCES alert_rules(id),
    rule_name TEXT NOT NULL,
    
    -- 触发内容
    intel_item_id UUID REFERENCES intelligence_items(id),
    intel_type TEXT NOT NULL,
    source TEXT NOT NULL,
    title TEXT,
    content TEXT,
    
    -- 告警状态
    severity TEXT NOT NULL,
    status TEXT DEFAULT 'pending',      -- 'pending', 'sent', 'acknowledged', 'resolved'
    
    -- 通知信息
    notified_at TIMESTAMPTZ,
    notification_channels TEXT[],       -- ['email', 'slack']
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alert_history_status ON alert_history(status);
CREATE INDEX IF NOT EXISTS idx_alert_history_created ON alert_history(created_at DESC);

-- 插入默认告警规则
INSERT INTO alert_rules (name, description, intel_type, condition, severity) VALUES
('price_change_significant', '价格变动超过10%', 'price', 'price.change_percent > 10', 'high'),
('api_deprecated', 'API废弃通知', 'api_doc', 'apidoc.change_type == ''deprecated''', 'critical'),
('api_breaking_change', 'API破坏性变更', 'api_doc', 'apidoc.change_type == ''breaking_change''', 'critical'),
('community_hot', '社区热门讨论', 'community', 'community.points > 500', 'medium')
ON CONFLICT (name) DO NOTHING;

-- 创建更新时间触发器
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_intelligence_items_updated_at ON intelligence_items;
CREATE TRIGGER update_intelligence_items_updated_at
    BEFORE UPDATE ON intelligence_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_alert_rules_updated_at ON alert_rules;
CREATE TRIGGER update_alert_rules_updated_at
    BEFORE UPDATE ON alert_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
