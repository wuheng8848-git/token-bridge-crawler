-- 创建噪声规则配置表
CREATE TABLE IF NOT EXISTS noise_rules (
    id SERIAL PRIMARY KEY,
    rule_type VARCHAR(50) NOT NULL,     -- 'keyword', 'length', 'regex', 'user_type'
    rule_name VARCHAR(100) NOT NULL,    -- 规则名称
    rule_value TEXT NOT NULL,           -- 规则值（JSON 或字符串）
    weight INT NOT NULL DEFAULT 1,      -- 权重（正数=噪声，负数=信号）
    is_active BOOLEAN DEFAULT TRUE,     -- 是否启用
    priority INT DEFAULT 0,             -- 执行优先级（越大越先执行）
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_noise_rules_type ON noise_rules(rule_type);
CREATE INDEX IF NOT EXISTS idx_noise_rules_active ON noise_rules(is_active);

-- 初始规则数据
INSERT INTO noise_rules (rule_type, rule_name, rule_value, weight, priority) VALUES
-- 噪声规则（正权重）
('keyword', '营销推广', '{"keywords":["check out my","my new tool","try our","visit my","subscribe"],"mode":"any"}', 10, 100),
('keyword', '招聘内容', '{"keywords":["hiring","job posting","we are looking for","join our team"],"mode":"any"}', 8, 95),
('length', '内容过短', '{"min":20,"max":0}', 5, 50),

-- 信号规则（负权重）
('keyword', '价格痛点', '{"keywords":["expensive","cost too much","pricey","too expensive","billing issue","my bill"],"mode":"any"}', -10, 90),
('keyword', '迁移意愿', '{"keywords":["alternative to","switching to","looking for alternative","migrate from","move away from"],"mode":"any"}', -15, 95),
('keyword', '功能需求', '{"keywords":["wish there was","need a","would be great if","feature request","missing feature"],"mode":"any"}', -8, 85),
('keyword', '竞品动态', '{"keywords":["claude","anthropic","gemini","llama","mistral","deepseek"],"mode":"any"}', -12, 80),
('keyword', '成本压力', '{"keywords":["rate limit","rate limited","quota exceeded","usage limit","throttle"],"mode":"any"}', -10, 88),
('keyword', '性能问题', '{"keywords":["slow response","latency","timeout","connection error","api error"],"mode":"any"}', -8, 85),
('keyword', '质量问题', '{"keywords":["hallucination","wrong answer","bad response","poor quality","inaccurate"],"mode":"any"}', -8, 82);
