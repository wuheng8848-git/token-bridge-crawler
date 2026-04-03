-- 营销信号与动作存储表
-- 用于存储从情报中检测到的客户信号和生成的营销动作

-- 客户信号表
CREATE TABLE IF NOT EXISTS customer_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- 关联情报
    intel_item_id UUID REFERENCES intelligence_items(id) ON DELETE SET NULL,
    
    -- 信号属性
    signal_type TEXT NOT NULL,           -- cost_pressure, config_friction, tool_fragmentation, governance_start, migration_intent, general_interest
    strength INT NOT NULL DEFAULT 1,     -- 1=低, 2=中, 3=高
    content TEXT,                        -- 信号内容摘要
    
    -- 来源信息
    platform TEXT,                       -- hacker_news, reddit, indie_hackers 等
    author TEXT,                         -- 作者标识
    url TEXT,                            -- 原始链接
    
    -- 扩展数据
    metadata JSONB DEFAULT '{}',
    
    -- 状态
    status TEXT DEFAULT 'new',           -- new, processed, ignored
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_signals_type ON customer_signals(signal_type);
CREATE INDEX IF NOT EXISTS idx_signals_intel_item ON customer_signals(intel_item_id);
CREATE INDEX IF NOT EXISTS idx_signals_status ON customer_signals(status);
CREATE INDEX IF NOT EXISTS idx_signals_detected ON customer_signals(detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_signals_strength ON customer_signals(strength DESC);

-- 营销动作表
CREATE TABLE IF NOT EXISTS marketing_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 动作属性
    action_type TEXT NOT NULL,           -- short_response, technical_post, setup_guide, competitor_comparison, follow_up
    channel TEXT NOT NULL,               -- hacker_news, reddit, indie_hackers, product_hunt, linkedin, twitter
    title TEXT,                          -- 动作标题
    content TEXT,                        -- 动作内容/模板
    template_id TEXT,                    -- 模板ID（可选）
    target_audience TEXT,                -- 目标受众

    -- 优先级与关联
    priority INT NOT NULL DEFAULT 3,     -- 1-5, 5最高
    signal_ids JSONB DEFAULT '[]',       -- 关联的信号ID数组（JSONB格式，与代码写入一致）

    -- 新语义字段
    auto_execute BOOLEAN DEFAULT FALSE,  -- 是否自动执行
    customer_stage TEXT,                 -- 客户阶段：awareness, consideration, decision, retention
    qualified_score NUMERIC(5,2),        -- 资格化分数（0-100）

    -- 扩展数据
    metadata JSONB DEFAULT '{}',

    -- 状态与时间
    status TEXT DEFAULT 'pending',       -- pending, scheduled, executed, failed, cancelled, draft
    scheduled_at TIMESTAMPTZ,            -- 计划执行时间
    executed_at TIMESTAMPTZ,             -- 实际执行时间
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_actions_type ON marketing_actions(action_type);
CREATE INDEX IF NOT EXISTS idx_actions_channel ON marketing_actions(channel);
CREATE INDEX IF NOT EXISTS idx_actions_status ON marketing_actions(status);
CREATE INDEX IF NOT EXISTS idx_actions_priority ON marketing_actions(priority DESC);
CREATE INDEX IF NOT EXISTS idx_actions_scheduled ON marketing_actions(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_actions_created ON marketing_actions(created_at DESC);

-- 复合索引：查询待处理动作
CREATE INDEX IF NOT EXISTS idx_actions_pending ON marketing_actions(status, priority DESC, created_at)
    WHERE status = 'pending';

-- 更新时间触发器
DROP TRIGGER IF EXISTS update_marketing_actions_updated_at ON marketing_actions;
CREATE TRIGGER update_marketing_actions_updated_at
    BEFORE UPDATE ON marketing_actions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 注释
COMMENT ON TABLE customer_signals IS '从情报中检测到的客户信号';
COMMENT ON TABLE marketing_actions IS '从信号生成的营销动作';

COMMENT ON COLUMN customer_signals.signal_type IS '信号类型: cost_pressure(P0), config_friction(P0), tool_fragmentation(P0), governance_start(P1), migration_intent(P1), general_interest(P3)';
COMMENT ON COLUMN customer_signals.strength IS '信号强度: 1=低, 2=中, 3=高';
COMMENT ON COLUMN customer_signals.status IS '状态: new(新检测), processed(已处理), ignored(已忽略)';

COMMENT ON COLUMN marketing_actions.action_type IS '动作类型: short_response, technical_post, setup_guide, competitor_comparison, follow_up';
COMMENT ON COLUMN marketing_actions.channel IS '目标渠道: hacker_news, reddit, indie_hackers, product_hunt, linkedin, twitter';
COMMENT ON COLUMN marketing_actions.status IS '状态: pending(待执行), scheduled(已排期), executed(已执行), failed(执行失败), cancelled(已取消)';