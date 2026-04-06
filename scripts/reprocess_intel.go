//go:build ignore
// +build ignore

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"token-bridge-crawler/internal/core"
	"token-bridge-crawler/internal/rules"
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
	log.Println("Connected to database")

	// 初始化规则引擎（从数据库加载规则）
	ruleStorage := rules.NewDBStorage(db)
	ruleEngine, err := rules.NewEngine(ruleStorage)
	if err != nil {
		log.Fatalf("Failed to create rule engine: %v", err)
	}
	log.Println("Rule engine initialized with database rules")

	// 列出加载的规则
	allRules, _ := ruleEngine.ListRules()
	log.Printf("Loaded %d rules from database", len(allRules))

	// 查询所有未处理或已处理的情报
	rows, err := db.QueryContext(ctx, `
		SELECT id, intel_type, source, source_id, title, content, url, metadata,
		       captured_at, published_at, status, quality_score, is_noise,
		       filter_reason, customer_tier, signal_type, pain_score
		FROM intelligence_items
		ORDER BY captured_at DESC
		LIMIT 100
	`)
	if err != nil {
		log.Fatalf("Failed to query intelligence items: %v", err)
	}
	defer rows.Close()

	// 统计
	var total, noiseCount, signalCount int
	var qualitySum float64

	fmt.Println("\n========== 重新处理情报数据 ==========")
	fmt.Println("ID | Source | Type | Quality | IsNoise | SignalType | PainScore")
	fmt.Println("---|--------|------|---------|---------|------------|----------")

	batchSize := 0
	for rows.Next() {
		var item core.IntelItem
		var metadataJSON []byte
		var publishedAt sql.NullTime
		var qualityScore, painScore sql.NullFloat64
		var isNoise sql.NullBool
		var filterReason, customerTier, signalType sql.NullString

		err := rows.Scan(
			&item.ID, &item.IntelType, &item.Source, &item.SourceID,
			&item.Title, &item.Content, &item.URL, &metadataJSON,
			&item.CapturedAt, &publishedAt, &item.Status,
			&qualityScore, &isNoise, &filterReason, &customerTier, &signalType, &painScore,
		)
		if err != nil {
			log.Printf("Failed to scan item: %v", err)
			continue
		}

		// 解析 metadata
		if metadataJSON != nil {
			item.Metadata = make(core.Metadata)
			// 简化处理，不解析 JSON
		}

		if publishedAt.Valid {
			item.PublishedAt = &publishedAt.Time
		}

		// 使用规则引擎重新评估
		result := ruleEngine.Evaluate(item)

		// 更新统计
		total++
		if result.IsNoise {
			noiseCount++
		} else {
			signalCount++
			qualitySum += result.QualityScore
		}

		// 显示结果
		isNoiseStr := "否"
		if result.IsNoise {
			isNoiseStr = "是"
		}

		fmt.Printf("%s | %s | %s | %.1f | %s | %s | %.1f\n",
			item.ID[:8], item.Source, item.IntelType,
			result.QualityScore, isNoiseStr, result.SignalType, 50.0)

		// 更新数据库（可选）
		_, updateErr := db.ExecContext(ctx, `
			UPDATE intelligence_items
			SET quality_score = $1, is_noise = $2, signal_type = $3, updated_at = NOW()
			WHERE id = $4
		`, result.QualityScore, result.IsNoise, result.SignalType, item.ID)
		if updateErr != nil {
			log.Printf("Failed to update item %s: %v", item.ID, updateErr)
		}

		batchSize++
		if batchSize >= 20 {
			// 每处理 20 条暂停一下，避免输出太快
			time.Sleep(100 * time.Millisecond)
			batchSize = 0
		}
	}

	// 输出统计
	fmt.Println("\n========== 处理统计 ==========")
	fmt.Printf("总情报数: %d\n", total)
	fmt.Printf("噪声: %d (%.1f%%)\n", noiseCount, float64(noiseCount)/float64(total)*100)
	fmt.Printf("信号: %d (%.1f%%)\n", signalCount, float64(signalCount)/float64(total)*100)
	if signalCount > 0 {
		fmt.Printf("平均质量分: %.1f\n", qualitySum/float64(signalCount))
	}

	// 按信号类型统计
	fmt.Println("\n========== 信号类型分布 ==========")
	rows2, err := db.QueryContext(ctx, `
		SELECT signal_type, COUNT(*) as count
		FROM intelligence_items
		WHERE is_noise = false AND signal_type IS NOT NULL
		GROUP BY signal_type
		ORDER BY count DESC
	`)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var signalType string
			var count int
			if err := rows2.Scan(&signalType, &count); err == nil {
				fmt.Printf("%s: %d\n", signalType, count)
			}
		}
	}

	// 按来源统计
	fmt.Println("\n========== 来源分布（Top 10） ==========")
	rows3, err := db.QueryContext(ctx, `
		SELECT source, COUNT(*) as count
		FROM intelligence_items
		GROUP BY source
		ORDER BY count DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var source string
			var count int
			if err := rows3.Scan(&source, &count); err == nil {
				fmt.Printf("%s: %d\n", source, count)
			}
		}
	}

	log.Println("\n重新处理完成！")
}
