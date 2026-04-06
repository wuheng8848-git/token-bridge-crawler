// Package main 修复规则乱码
package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	// 数据库连接
	dbURL := "postgres://tbv2:tbv2_password@localhost:15432/token_bridge_crawler?sslmode=disable"

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("数据库连接测试失败: %v", err)
	}

	fmt.Println("=== 开始修复规则乱码 ===")

	// 清空现有规则
	_, err = db.Exec("DELETE FROM noise_rules")
	if err != nil {
		log.Fatalf("清空规则失败: %v", err)
	}
	fmt.Println("✓ 已清空现有规则")

	// 插入正确的规则（英文命名）
	rules := []struct {
		ruleType  string
		ruleName  string
		ruleValue string
		weight    int
		priority  int
	}{
		// Noise rules (positive weight)
		{"keyword", "spam_marketing", `{"keywords":["check out my","my new tool","try our","visit my","subscribe"],"mode":"any"}`, 10, 100},
		{"keyword", "hiring_recruitment", `{"keywords":["hiring","job posting","we are looking for","join our team"],"mode":"any"}`, 8, 95},
		{"length", "content_too_short", `{"min":20,"max":0}`, 5, 50},

		// Signal rules (negative weight)
		{"keyword", "cost_pain", `{"keywords":["expensive","cost too much","pricey","too expensive","billing issue","my bill"],"mode":"any"}`, -10, 90},
		{"keyword", "intent_migration", `{"keywords":["alternative to","switching to","looking for alternative","migrate from","move away from"],"mode":"any"}`, -15, 95},
		{"keyword", "feature_request", `{"keywords":["wish there was","need a","would be great if","feature request","missing feature"],"mode":"any"}`, -8, 85},
		{"keyword", "competitor_mention", `{"keywords":["claude","anthropic","gemini","llama","mistral","deepseek"],"mode":"any"}`, -12, 80},
		{"keyword", "rate_limit_issue", `{"keywords":["rate limit","rate limited","quota exceeded","usage limit","throttle"],"mode":"any"}`, -10, 88},
		{"keyword", "performance_issue", `{"keywords":["slow response","latency","timeout","connection error","api error"],"mode":"any"}`, -8, 85},
		{"keyword", "quality_issue", `{"keywords":["hallucination","wrong answer","bad response","poor quality","inaccurate"],"mode":"any"}`, -8, 82},
	}

	for _, rule := range rules {
		_, err := db.Exec(
			"INSERT INTO noise_rules (rule_type, rule_name, rule_value, weight, priority) VALUES ($1, $2, $3, $4, $5)",
			rule.ruleType, rule.ruleName, rule.ruleValue, rule.weight, rule.priority,
		)
		if err != nil {
			log.Printf("插入规则 '%s' 失败: %v", rule.ruleName, err)
		} else {
			fmt.Printf("✓ 插入规则: %s (权重: %d, 优先级: %d)\n", rule.ruleName, rule.weight, rule.priority)
		}
	}

	// 验证
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM noise_rules").Scan(&count)
	if err != nil {
		log.Fatalf("查询规则数量失败: %v", err)
	}

	fmt.Printf("\n=== 修复完成! 总共 %d 条规则 ===\n", count)
}
