//go:build ignore
// +build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Rule 规则定义
type Rule struct {
	ID        int64     `json:"id"`
	RuleType  string    `json:"rule_type"`
	RuleName  string    `json:"rule_name"`
	RuleValue string    `json:"rule_value"`
	Weight    int       `json:"weight"`
	IsActive  bool      `json:"is_active"`
	Priority  int       `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func main() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Println("[Config] .env file not found, using environment variables")
	}

	// 获取数据库连接字符串
	databaseURL := os.Getenv("CRAWLER_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://tbuser:tbpassword@localhost:5432/tokenbridge?sslmode=disable"
	}

	// 连接数据库
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database")

	// 定义默认规则
	rules := []Rule{
		// 噪声规则（正权重）
		{
			RuleType:  "keyword",
			RuleName:  "spam_marketing",
			RuleValue: `{"keywords":["check out my","my new tool","try our","visit my","subscribe"],"mode":"any"}`,
			Weight:    10,
			IsActive:  true,
			Priority:  100,
		},
		{
			RuleType:  "length",
			RuleName:  "content_too_short",
			RuleValue: `{"min":20,"max":0}`,
			Weight:    5,
			IsActive:  true,
			Priority:  50,
		},
		{
			RuleType:  "keyword",
			RuleName:  "hiring_recruitment",
			RuleValue: `{"keywords":["we are hiring","join our team","career opportunity","apply now","job opening","position available","looking for"],"mode":"any"}`,
			Weight:    8,
			IsActive:  true,
			Priority:  90,
		},
		{
			RuleType:  "keyword",
			RuleName:  "tech_chitchat",
			RuleValue: `{"keywords":["python vs go","vim vs emacs","best laptop","favorite editor","tabs vs spaces","docker vs vm","mac vs windows"],"mode":"any"}`,
			Weight:    6,
			IsActive:  true,
			Priority:  70,
		},
		// 信号规则（负权重）
		{
			RuleType:  "keyword",
			RuleName:  "pain_price",
			RuleValue: `{"keywords":["expensive","cost too much","pricey","too expensive","billing issue"],"mode":"any"}`,
			Weight:    -8,
			IsActive:  true,
			Priority:  90,
		},
		{
			RuleType:  "keyword",
			RuleName:  "intent_migration",
			RuleValue: `{"keywords":["alternative to","switching to","looking for alternative","migrate from"],"mode":"any"}`,
			Weight:    -10,
			IsActive:  true,
			Priority:  95,
		},
		{
			RuleType:  "keyword",
			RuleName:  "feature_request",
			RuleValue: `{"keywords":["wish there was","need a","would be great if","feature request"],"mode":"any"}`,
			Weight:    -7,
			IsActive:  true,
			Priority:  85,
		},
		{
			RuleType:  "keyword",
			RuleName:  "competitor_news",
			RuleValue: `{"keywords":["claude pricing","anthropic api cost","gemini price comparison"],"mode":"any"}`,
			Weight:    -6,
			IsActive:  true,
			Priority:  80,
		},
		{
			RuleType:  "keyword",
			RuleName:  "cost_pressure",
			RuleValue: `{"keywords":["my bill","billing is","invoice","payment issue","rate limit"],"mode":"any"}`,
			Weight:    -8,
			IsActive:  true,
			Priority:  88,
		},
	}

	// 插入规则
	now := time.Now()
	inserted := 0
	for _, rule := range rules {
		rule.CreatedAt = now
		rule.UpdatedAt = now

		// 检查规则是否已存在
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM noise_rules WHERE rule_name = $1)", rule.RuleName).Scan(&exists)
		if err != nil {
			log.Printf("Failed to check if rule exists: %v", err)
			continue
		}

		if exists {
			log.Printf("Rule '%s' already exists, skipping", rule.RuleName)
			continue
		}

		// 插入规则
		_, err = db.Exec(
			`INSERT INTO noise_rules (rule_type, rule_name, rule_value, weight, is_active, priority, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			rule.RuleType, rule.RuleName, rule.RuleValue, rule.Weight, rule.IsActive, rule.Priority, rule.CreatedAt, rule.UpdatedAt,
		)
		if err != nil {
			log.Printf("Failed to insert rule '%s': %v", rule.RuleName, err)
			continue
		}

		inserted++
		log.Printf("Inserted rule: %s (weight: %d)", rule.RuleName, rule.Weight)
	}

	log.Printf("Successfully inserted %d/%d rules", inserted, len(rules))

	// 验证插入的规则
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM noise_rules").Scan(&count)
	if err != nil {
		log.Fatalf("Failed to count rules: %v", err)
	}
	log.Printf("Total rules in database: %d", count)

	// 列出所有规则
	rows, err := db.Query("SELECT id, rule_name, rule_type, weight, priority FROM noise_rules ORDER BY id")
	if err != nil {
		log.Fatalf("Failed to query rules: %v", err)
	}
	defer rows.Close()

	fmt.Println("\nRules in database:")
	fmt.Println("ID | Name | Type | Weight | Priority")
	fmt.Println("---|------|------|--------|----------")
	for rows.Next() {
		var id, weight, priority int
		var name, ruleType string
		if err := rows.Scan(&id, &name, &ruleType, &weight, &priority); err != nil {
			continue
		}
		fmt.Printf("%d | %s | %s | %d | %d\n", id, name, ruleType, weight, priority)
	}
}
