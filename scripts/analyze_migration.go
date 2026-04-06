//go:build ignore
// +build ignore

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

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

	ctx := context.Background()

	// 连接数据库
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// 查询包含迁移关键词的情报
	keywords := []string{"alternative", "switching", "migrate", "move from", "transition from"}

	for _, keyword := range keywords {
		fmt.Printf("\n========== 关键词: %s ==========\n", keyword)

		rows, err := db.QueryContext(ctx, `
			SELECT id, source, title, content
			FROM intelligence_items
			WHERE content ILIKE $1 OR title ILIKE $1
			LIMIT 5
		`, "%"+keyword+"%")
		if err != nil {
			log.Printf("Failed to query: %v", err)
			continue
		}
		defer rows.Close()

		found := 0
		for rows.Next() {
			var id, source, title, content string
			if err := rows.Scan(&id, &source, &title, &content); err != nil {
				continue
			}

			// 使用规则引擎评估
			text := strings.ToLower(title + " " + content)

			// 检查是否匹配迁移规则
			migrationKeywords := []string{"alternative to", "switching to", "looking for alternative", "migrate from"}
			matched := false
			for _, mk := range migrationKeywords {
				if strings.Contains(text, mk) {
					matched = true
					break
				}
			}

			found++
			fmt.Printf("\n--- 情报 %d ---\n", found)
			fmt.Printf("ID: %s\n", id[:8])
			fmt.Printf("Source: %s\n", source)
			fmt.Printf("Title: %s\n", title)
			fmt.Printf("Content: %.200s...\n", content)
			fmt.Printf("匹配迁移规则: %v\n", matched)

			// 更新数据库
			if matched {
				_, err = db.ExecContext(ctx, `
					UPDATE intelligence_items
					SET signal_type = 'migration', quality_score = 80, updated_at = NOW()
					WHERE id = $1
				`, id)
				if err != nil {
					log.Printf("Failed to update: %v", err)
				} else {
					fmt.Println("✓ 已标记为迁移意愿信号")
				}
			}
		}

		if found == 0 {
			fmt.Println("未找到匹配的情报")
		}
	}

	// 统计
	fmt.Println("\n========== 迁移意愿信号统计 ==========")
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM intelligence_items WHERE signal_type = 'migration'").Scan(&count)
	if err != nil {
		log.Printf("Failed to count: %v", err)
	} else {
		fmt.Printf("总共标记了 %d 条迁移意愿信号\n", count)
	}
}
