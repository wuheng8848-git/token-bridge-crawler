-- 创建噪声规则表
CREATE TABLE IF NOT EXISTS noise_rules (
    id SERIAL PRIMARY KEY,
    rule_type VARCHAR(20) NOT NULL CHECK (rule_type IN ('keyword', 'length', 'regex', 'user_type')),
    rule_name VARCHAR(100) NOT NULL UNIQUE,
    rule_value TEXT NOT NULL,
    weight INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    priority INTEGER NOT NULL DEFAULT 50,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_noise_rules_active ON noise_rules(is_active);
CREATE INDEX IF NOT EXISTS idx_noise_rules_priority ON noise_rules(priority DESC);
CREATE INDEX IF NOT EXISTS idx_noise_rules_type ON noise_rules(rule_type);

-- Table for noise filtering rules, positive weight = noise, negative weight = signal
