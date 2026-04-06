-- 修复 noise_rules 表中的乱码规则名称
-- 先清空现有数据
DELETE FROM noise_rules;

-- 重新插入正确的规则（UTF-8 编码）
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

-- 验证插入结果
SELECT id, rule_name, weight, priority FROM noise_rules ORDER BY priority DESC;
