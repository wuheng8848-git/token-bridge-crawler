// Package storage 提供情报系统存储功能
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"token-bridge-crawler/internal/core"
)

// IntelligenceStorage 情报存储接口
type IntelligenceStorage interface {
	// 情报项操作
	SaveItem(ctx context.Context, item *core.IntelItem) error
	SaveItems(ctx context.Context, items []core.IntelItem) error
	GetItemByID(ctx context.Context, id string) (*core.IntelItem, error)
	GetItems(ctx context.Context, filter IntelFilter) ([]core.IntelItem, error)
	GetItemsCount(ctx context.Context, filter IntelFilter) (int64, error)
	UpdateItemStatus(ctx context.Context, id string, status core.IntelStatus) error

	// 统计查询
	GetSourceStats(ctx context.Context, timeRange ...time.Time) (map[string]int64, error)
	GetTranslationStats(ctx context.Context) (map[string]int64, error)
	GetQualityAnalysis(ctx context.Context, source string, limit int) (map[string]interface{}, error)

	// 采集器运行日志
	SaveCollectorRun(ctx context.Context, run *CollectorRun) error
	GetCollectorRuns(ctx context.Context, collectorName string, limit int) ([]CollectorRun, error)

	// 告警相关
	GetAlertRules(ctx context.Context, enabledOnly bool) ([]AlertRule, error)
	SaveAlertHistory(ctx context.Context, alert *AlertHistory) error
	UpdateAlertStatus(ctx context.Context, alertID string, status string) error

	// 营销信号与动作
	SaveSignals(ctx context.Context, signals []CustomerSignal) error
	SaveActions(ctx context.Context, actions []MarketingAction) error
	GetPendingActions(ctx context.Context, limit int) ([]MarketingAction, error)
	UpdateActionStatus(ctx context.Context, actionID string, status string) error

	// 统计查询
	GetStats(ctx context.Context, startTime, endTime time.Time) (IntelStats, error)

	// 资源管理
	Close()
}

// IntelFilter 情报查询过滤器
type IntelFilter struct {
	IntelType core.IntelType
	Source    string
	Status    core.IntelStatus
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

// CollectorRun 采集器运行记录
type CollectorRun struct {
	ID            string     `db:"id"`
	CollectorName string     `db:"collector_name"`
	IntelType     string     `db:"intel_type"`
	Source        string     `db:"source"`
	Status        string     `db:"status"` // 'success', 'failed', 'partial'
	ItemsCount    int        `db:"items_count"`
	ErrorMessage  string     `db:"error_message"`
	StartedAt     time.Time  `db:"started_at"`
	CompletedAt   *time.Time `db:"completed_at"`
	DurationMs    int        `db:"duration_ms"`
	StrategyUsed  string     `db:"strategy_used"`
	CreatedAt     time.Time  `db:"created_at"`
}

// AlertRule 告警规则
type AlertRule struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	IntelType   *string   `db:"intel_type"`
	Condition   string    `db:"condition"`
	Severity    string    `db:"severity"`
	Enabled     bool      `db:"enabled"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// AlertHistory 告警历史
type AlertHistory struct {
	ID                   string     `db:"id"`
	RuleID               *string    `db:"rule_id"`
	RuleName             string     `db:"rule_name"`
	IntelItemID          *string    `db:"intel_item_id"`
	IntelType            string     `db:"intel_type"`
	Source               string     `db:"source"`
	Title                string     `db:"title"`
	Content              string     `db:"content"`
	Severity             string     `db:"severity"`
	Status               string     `db:"status"`
	NotifiedAt           *time.Time `db:"notified_at"`
	NotificationChannels []string   `db:"notification_channels"`
	CreatedAt            time.Time  `db:"created_at"`
}

// IntelStats 情报统计
type IntelStats struct {
	TotalItems      int64
	ItemsByType     map[core.IntelType]int64
	ItemsBySource   map[string]int64
	ItemsByStatus   map[core.IntelStatus]int64
	CollectorRuns   int64
	AlertsTriggered int64
}

// CustomerSignal 客户信号
type CustomerSignal struct {
	ID          string                 `json:"id" db:"id"`
	IntelItemID *string                `json:"intel_item_id,omitempty" db:"intel_item_id"`
	SignalType  string                 `json:"signal_type" db:"signal_type"`
	Strength    int                    `json:"strength" db:"strength"`
	Content     string                 `json:"content" db:"content"`
	Platform    string                 `json:"platform" db:"platform"`
	Author      string                 `json:"author" db:"author"`
	URL         string                 `json:"url" db:"url"`
	Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
	Status      string                 `json:"status" db:"status"`
	DetectedAt  time.Time              `json:"detected_at" db:"detected_at"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}

// MarketingAction 营销动作
type MarketingAction struct {
	ID             string                 `json:"id" db:"id"`
	ActionType     string                 `json:"action_type" db:"action_type"`
	Channel        string                 `json:"channel" db:"channel"`
	Title          string                 `json:"title" db:"title"`
	Content        string                 `json:"content" db:"content"`
	TemplateID     string                 `json:"template_id,omitempty" db:"template_id"`
	TargetAudience string                 `json:"target_audience" db:"target_audience"`
	Priority       int                    `json:"priority" db:"priority"`
	SignalIDs      []string               `json:"signal_ids" db:"signal_ids"`
	AutoExecute    bool                   `json:"auto_execute" db:"auto_execute"`
	CustomerStage  string                 `json:"customer_stage,omitempty" db:"customer_stage"`
	QualifiedScore float64                `json:"qualified_score,omitempty" db:"qualified_score"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	Status         string                 `json:"status" db:"status"`
	ScheduledAt    *time.Time             `json:"scheduled_at,omitempty" db:"scheduled_at"`
	ExecutedAt     *time.Time             `json:"executed_at,omitempty" db:"executed_at"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

// PostgresIntelligenceStorage PostgreSQL实现
type PostgresIntelligenceStorage struct {
	pool *pgxpool.Pool
}

// NewPostgresIntelligenceStorage 创建PostgreSQL存储
func NewPostgresIntelligenceStorage(databaseURL string) (IntelligenceStorage, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return &PostgresIntelligenceStorage{pool: pool}, nil
}

// Close 关闭连接
func (s *PostgresIntelligenceStorage) Close() {
	s.pool.Close()
}

// SaveItem 保存单个情报项
func (s *PostgresIntelligenceStorage) SaveItem(ctx context.Context, item *core.IntelItem) error {
	metadataJSON, err := json.Marshal(item.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO intelligence_items
		(id, intel_type, source, source_id, title, content, url, metadata, captured_at, published_at, status, created_at,
		 quality_score, is_noise, filter_reason, customer_tier, signal_type, pain_score)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (source, source_id, captured_at) WHERE source_id IS NOT NULL
		DO UPDATE SET
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			metadata = EXCLUDED.metadata,
			quality_score = EXCLUDED.quality_score,
			is_noise = EXCLUDED.is_noise,
			filter_reason = EXCLUDED.filter_reason,
			customer_tier = EXCLUDED.customer_tier,
			signal_type = EXCLUDED.signal_type,
			pain_score = EXCLUDED.pain_score,
			updated_at = NOW()
	`, item.ID, item.IntelType, item.Source, item.SourceID,
		item.Title, item.Content, item.URL, metadataJSON,
		item.CapturedAt, item.PublishedAt, item.Status, item.CreatedAt,
		item.QualityScore, item.IsNoise, item.FilterReason, item.CustomerTier, item.SignalType, item.PainScore)

	return err
}

// SaveItems 批量保存情报项
func (s *PostgresIntelligenceStorage) SaveItems(ctx context.Context, items []core.IntelItem) error {
	if len(items) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, item := range items {
		metadataJSON, _ := json.Marshal(item.Metadata)
		batch.Queue(`
			INSERT INTO intelligence_items
			(id, intel_type, source, source_id, title, content, url, metadata, captured_at, published_at, status, created_at,
			 quality_score, is_noise, filter_reason, customer_tier, signal_type, pain_score)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
			ON CONFLICT (source, source_id, captured_at) WHERE source_id IS NOT NULL
			DO UPDATE SET
				title = EXCLUDED.title,
				content = EXCLUDED.content,
				metadata = EXCLUDED.metadata,
				quality_score = EXCLUDED.quality_score,
				is_noise = EXCLUDED.is_noise,
				filter_reason = EXCLUDED.filter_reason,
				customer_tier = EXCLUDED.customer_tier,
				signal_type = EXCLUDED.signal_type,
				pain_score = EXCLUDED.pain_score,
				updated_at = NOW()
		`, item.ID, item.IntelType, item.Source, item.SourceID,
			item.Title, item.Content, item.URL, metadataJSON,
			item.CapturedAt, item.PublishedAt, item.Status, item.CreatedAt,
			item.QualityScore, item.IsNoise, item.FilterReason, item.CustomerTier, item.SignalType, item.PainScore)
	}

	results := s.pool.SendBatch(ctx, batch)
	return results.Close()
}

// GetItemByID 根据ID获取情报项
func (s *PostgresIntelligenceStorage) GetItemByID(ctx context.Context, id string) (*core.IntelItem, error) {
	var item core.IntelItem
	var metadataJSON []byte

	err := s.pool.QueryRow(ctx, `
		SELECT id, intel_type, source, source_id, title, content, url, metadata,
		       captured_at, published_at, status, created_at
		FROM intelligence_items
		WHERE id = $1
	`, id).Scan(&item.ID, &item.IntelType, &item.Source, &item.SourceID,
		&item.Title, &item.Content, &item.URL, &metadataJSON,
		&item.CapturedAt, &item.PublishedAt, &item.Status, &item.CreatedAt)

	if err != nil {
		return nil, err
	}

	if len(metadataJSON) > 0 {
		_ = json.Unmarshal(metadataJSON, &item.Metadata)
	}

	return &item, nil
}

// GetItems 查询情报项列表
func (s *PostgresIntelligenceStorage) GetItems(ctx context.Context, filter IntelFilter) ([]core.IntelItem, error) {
	query := `
		SELECT id, intel_type, source, source_id, title, content, url, metadata,
		       captured_at, published_at, status, created_at
		FROM intelligence_items
		WHERE 1=1
	`
	var args []interface{}
	argIdx := 1

	if filter.IntelType != "" {
		query += fmt.Sprintf(" AND intel_type = $%d", argIdx)
		args = append(args, filter.IntelType)
		argIdx++
	}

	if filter.Source != "" {
		query += fmt.Sprintf(" AND source = $%d", argIdx)
		args = append(args, filter.Source)
		argIdx++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND captured_at >= $%d", argIdx)
		args = append(args, *filter.StartTime)
		argIdx++
	}

	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND captured_at <= $%d", argIdx)
		args = append(args, *filter.EndTime)
		argIdx++
	}

	query += " ORDER BY captured_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []core.IntelItem
	for rows.Next() {
		var item core.IntelItem
		var metadataJSON []byte

		err := rows.Scan(&item.ID, &item.IntelType, &item.Source, &item.SourceID,
			&item.Title, &item.Content, &item.URL, &metadataJSON,
			&item.CapturedAt, &item.PublishedAt, &item.Status, &item.CreatedAt)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			_ = json.Unmarshal(metadataJSON, &item.Metadata)
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

// UpdateItemStatus 更新情报项状态
func (s *PostgresIntelligenceStorage) UpdateItemStatus(ctx context.Context, id string, status core.IntelStatus) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE intelligence_items
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`, status, id)
	return err
}

// GetItemsCount 获取符合条件的情报项总数
func (s *PostgresIntelligenceStorage) GetItemsCount(ctx context.Context, filter IntelFilter) (int64, error) {
	query := `SELECT COUNT(*) FROM intelligence_items WHERE 1=1`
	var args []interface{}
	argIdx := 1

	if filter.IntelType != "" {
		query += fmt.Sprintf(" AND intel_type = $%d", argIdx)
		args = append(args, filter.IntelType)
		argIdx++
	}

	if filter.Source != "" {
		query += fmt.Sprintf(" AND source = $%d", argIdx)
		args = append(args, filter.Source)
		argIdx++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND captured_at >= $%d", argIdx)
		args = append(args, *filter.StartTime)
		argIdx++
	}

	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND captured_at <= $%d", argIdx)
		args = append(args, *filter.EndTime)
	}

	var count int64
	err := s.pool.QueryRow(ctx, query, args...).Scan(&count)
	return count, err
}

// GetSourceStats 获取按来源分组的统计数据
func (s *PostgresIntelligenceStorage) GetSourceStats(ctx context.Context, timeRange ...time.Time) (map[string]int64, error) {
	query := `SELECT source, COUNT(*) as count FROM intelligence_items`
	var args []interface{}

	if len(timeRange) >= 2 {
		query += " WHERE captured_at >= $1 AND captured_at < $2"
		args = append(args, timeRange[0], timeRange[1])
	}

	query += " GROUP BY source"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int64)
	for rows.Next() {
		var source string
		var count int64
		if err := rows.Scan(&source, &count); err != nil {
			return nil, err
		}
		stats[source] = count
	}

	return stats, rows.Err()
}

// GetTranslationStats 获取翻译覆盖率统计
func (s *PostgresIntelligenceStorage) GetTranslationStats(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE metadata ? 'title_zh') as translated
		FROM intelligence_items
	`

	var total, translated int64
	err := s.pool.QueryRow(ctx, query).Scan(&total, &translated)
	if err != nil {
		return nil, err
	}

	return map[string]int64{
		"total":      total,
		"translated": translated,
	}, nil
}

// GetQualityAnalysis 获取采集器质量分析
// 分析情报内容与项目焦点的匹配度
func (s *PostgresIntelligenceStorage) GetQualityAnalysis(ctx context.Context, source string, limit int) (map[string]interface{}, error) {
	// 项目核心关注关键词
	coreKeywords := []string{
		"openai", "chatgpt", "gpt-4", "gpt-3.5", "gpt-4o", "o1", "o3",
		"anthropic", "claude", "claude-3", "claude-4",
		"google", "gemini", "bard", "vertex ai",
		"llm", "large language model", "ai api", "token",
		"pricing", "cost", "expensive", "billing",
		"api key", "rate limit", "quota",
		"embedding", "completion", "fine-tuning",
		"azure openai", "bedrock", "huggingface",
		"langchain", "llamaindex", "vector",
	}

	// 主题分类
	themes := map[string][]string{
		"pricing_cost":  {"pricing", "cost", "expensive", "billing", "price", "charge", "subscription"},
		"api_technical": {"api", "endpoint", "rate limit", "quota", "throttle", "error", "bug"},
		"model_usage":   {"gpt-4", "claude", "gemini", "llm", "model", "completion", "token"},
		"integration":   {"langchain", "sdk", "integration", "library", "framework"},
		"comparison":    {"vs", "alternative", "comparison", "switch", "migrate"},
	}

	// 构建查询
	query := `
		SELECT id, source, title, content, metadata
		FROM intelligence_items
		WHERE 1=1
	`
	var args []interface{}
	argIdx := 1

	if source != "" {
		query += fmt.Sprintf(" AND source = $%d", argIdx)
		args = append(args, source)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY captured_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 分析结果
	sourceAnalysis := make(map[string]map[string]interface{})
	totalItems := 0
	keywordHits := make(map[string]int)
	themeDistribution := make(map[string]int)

	for rows.Next() {
		var id, src string
		var title, content string
		var metadata []byte

		if err := rows.Scan(&id, &src, &title, &content, &metadata); err != nil {
			continue
		}

		totalItems++
		text := strings.ToLower(title + " " + content)

		// 初始化来源统计
		if _, exists := sourceAnalysis[src]; !exists {
			sourceAnalysis[src] = map[string]interface{}{
				"total":        0,
				"keywordHits":  0,
				"themes":       make(map[string]int),
				"matchedItems": 0,
			}
		}

		sourceStats := sourceAnalysis[src]
		sourceStats["total"] = sourceStats["total"].(int) + 1

		// 检查关键词命中
		itemHasMatch := false
		for _, kw := range coreKeywords {
			if strings.Contains(text, kw) {
				keywordHits[kw]++
				sourceStats["keywordHits"] = sourceStats["keywordHits"].(int) + 1
				itemHasMatch = true
			}
		}

		if itemHasMatch {
			sourceStats["matchedItems"] = sourceStats["matchedItems"].(int) + 1
		}

		// 主题分类
		itemThemes := sourceStats["themes"].(map[string]int)
		for theme, kws := range themes {
			for _, kw := range kws {
				if strings.Contains(text, kw) {
					itemThemes[theme]++
					themeDistribution[theme]++
					break
				}
			}
		}
	}

	// 计算相关性得分
	for _, stats := range sourceAnalysis {
		total := stats["total"].(int)
		matched := stats["matchedItems"].(int)
		if total > 0 {
			stats["relevanceScore"] = float64(matched) / float64(total) * 100
		} else {
			stats["relevanceScore"] = 0.0
		}
	}

	// 获取总体统计
	overallStats, _ := s.getOverallStats(ctx)

	return map[string]interface{}{
		"summary": map[string]interface{}{
			"totalItems":       totalItems,
			"overallRelevance": calculateOverallRelevance(sourceAnalysis),
			"topKeywords":      getTopKeywords(keywordHits, 10),
		},
		"themeDistribution": themeDistribution,
		"sourceAnalysis":    sourceAnalysis,
		"overallStats":      overallStats,
		"keywords":          coreKeywords,
		"updatedAt":         time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// getOverallStats 获取总体统计
func (s *PostgresIntelligenceStorage) getOverallStats(ctx context.Context) (map[string]int64, error) {
	stats := make(map[string]int64)

	// 总数
	var total int64
	err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM intelligence_items").Scan(&total)
	if err != nil {
		return nil, err
	}
	stats["total"] = total

	// 按类型统计
	rows, err := s.pool.Query(ctx, "SELECT intel_type, COUNT(*) FROM intelligence_items GROUP BY intel_type")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var intelType string
			var count int64
			if rows.Scan(&intelType, &count) == nil {
				stats["type_"+intelType] = count
			}
		}
	}

	return stats, nil
}

// calculateOverallRelevance 计算整体相关性得分
func calculateOverallRelevance(sourceAnalysis map[string]map[string]interface{}) float64 {
	var totalItems, totalMatched int
	for _, stats := range sourceAnalysis {
		totalItems += stats["total"].(int)
		totalMatched += stats["matchedItems"].(int)
	}
	if totalItems == 0 {
		return 0
	}
	return float64(totalMatched) / float64(totalItems) * 100
}

// getTopKeywords 获取热门关键词
func getTopKeywords(keywordHits map[string]int, limit int) []map[string]interface{} {
	type kwCount struct {
		Keyword string
		Count   int
	}

	var list []kwCount
	for kw, count := range keywordHits {
		list = append(list, kwCount{kw, count})
	}

	// 简单排序
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].Count > list[i].Count {
				list[i], list[j] = list[j], list[i]
			}
		}
	}

	// 限制数量
	if len(list) > limit {
		list = list[:limit]
	}

	result := make([]map[string]interface{}, len(list))
	for i, kw := range list {
		result[i] = map[string]interface{}{
			"keyword": kw.Keyword,
			"count":   kw.Count,
		}
	}
	return result
}

// SaveCollectorRun 保存采集器运行记录
func (s *PostgresIntelligenceStorage) SaveCollectorRun(ctx context.Context, run *CollectorRun) error {
	if run.ID == "" {
		run.ID = uuid.New().String()
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO collector_runs
		(id, collector_name, intel_type, source, status, items_count, error_message,
		 started_at, completed_at, duration_ms, strategy_used, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, run.ID, run.CollectorName, run.IntelType, run.Source, run.Status,
		run.ItemsCount, run.ErrorMessage, run.StartedAt, run.CompletedAt,
		run.DurationMs, run.StrategyUsed, time.Now().UTC())

	return err
}

// GetCollectorRuns 获取采集器运行记录
func (s *PostgresIntelligenceStorage) GetCollectorRuns(ctx context.Context, collectorName string, limit int) ([]CollectorRun, error) {
	query := `
		SELECT id, collector_name, intel_type, source, status, items_count, error_message,
		       started_at, completed_at, duration_ms, strategy_used, created_at
		FROM collector_runs
	`
	var args []interface{}

	if collectorName != "" {
		query += " WHERE collector_name = $1"
		args = append(args, collectorName)
	}

	query += " ORDER BY started_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", len(args)+1)
		args = append(args, limit)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []CollectorRun
	for rows.Next() {
		var run CollectorRun
		err := rows.Scan(&run.ID, &run.CollectorName, &run.IntelType, &run.Source,
			&run.Status, &run.ItemsCount, &run.ErrorMessage,
			&run.StartedAt, &run.CompletedAt, &run.DurationMs,
			&run.StrategyUsed, &run.CreatedAt)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}

	return runs, rows.Err()
}

// GetAlertRules 获取告警规则
func (s *PostgresIntelligenceStorage) GetAlertRules(ctx context.Context, enabledOnly bool) ([]AlertRule, error) {
	query := `
		SELECT id, name, description, intel_type, condition, severity, enabled, created_at, updated_at
		FROM alert_rules
	`
	var args []interface{}

	if enabledOnly {
		query += " WHERE enabled = $1"
		args = append(args, true)
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []AlertRule
	for rows.Next() {
		var rule AlertRule
		err := rows.Scan(&rule.ID, &rule.Name, &rule.Description, &rule.IntelType,
			&rule.Condition, &rule.Severity, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

// SaveAlertHistory 保存告警历史
func (s *PostgresIntelligenceStorage) SaveAlertHistory(ctx context.Context, alert *AlertHistory) error {
	if alert.ID == "" {
		alert.ID = uuid.New().String()
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO alert_history
		(id, rule_id, rule_name, intel_item_id, intel_type, source, title, content,
		 severity, status, notification_channels, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, alert.ID, alert.RuleID, alert.RuleName, alert.IntelItemID,
		alert.IntelType, alert.Source, alert.Title, alert.Content,
		alert.Severity, alert.Status, alert.NotificationChannels, time.Now().UTC())

	return err
}

// UpdateAlertStatus 更新告警状态
func (s *PostgresIntelligenceStorage) UpdateAlertStatus(ctx context.Context, alertID string, status string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE alert_history
		SET status = $1, notified_at = CASE WHEN $1 = 'sent' THEN NOW() ELSE notified_at END
		WHERE id = $2
	`, status, alertID)
	return err
}

// GetStats 获取统计信息
func (s *PostgresIntelligenceStorage) GetStats(ctx context.Context, startTime, endTime time.Time) (IntelStats, error) {
	stats := IntelStats{
		ItemsByType:   make(map[core.IntelType]int64),
		ItemsBySource: make(map[string]int64),
		ItemsByStatus: make(map[core.IntelStatus]int64),
	}

	// 总数量
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM intelligence_items
		WHERE captured_at BETWEEN $1 AND $2
	`, startTime, endTime).Scan(&stats.TotalItems)
	if err != nil {
		return stats, err
	}

	// 按类型统计
	rows, err := s.pool.Query(ctx, `
		SELECT intel_type, COUNT(*) FROM intelligence_items
		WHERE captured_at BETWEEN $1 AND $2
		GROUP BY intel_type
	`, startTime, endTime)
	if err != nil {
		return stats, err
	}
	for rows.Next() {
		var intelType core.IntelType
		var count int64
		if err := rows.Scan(&intelType, &count); err == nil {
			stats.ItemsByType[intelType] = count
		}
	}
	rows.Close()

	// 按来源统计
	rows, err = s.pool.Query(ctx, `
		SELECT source, COUNT(*) FROM intelligence_items
		WHERE captured_at BETWEEN $1 AND $2
		GROUP BY source
	`, startTime, endTime)
	if err != nil {
		return stats, err
	}
	for rows.Next() {
		var source string
		var count int64
		if err := rows.Scan(&source, &count); err == nil {
			stats.ItemsBySource[source] = count
		}
	}
	rows.Close()

	// 按状态统计
	rows, err = s.pool.Query(ctx, `
		SELECT status, COUNT(*) FROM intelligence_items
		WHERE captured_at BETWEEN $1 AND $2
		GROUP BY status
	`, startTime, endTime)
	if err != nil {
		return stats, err
	}
	for rows.Next() {
		var status core.IntelStatus
		var count int64
		if err := rows.Scan(&status, &count); err == nil {
			stats.ItemsByStatus[status] = count
		}
	}
	rows.Close()

	return stats, nil
}

// SaveSignals 批量保存客户信号
func (s *PostgresIntelligenceStorage) SaveSignals(ctx context.Context, signals []CustomerSignal) error {
	if len(signals) == 0 {
		return nil
	}

	for _, signal := range signals {
		if signal.ID == "" {
			signal.ID = uuid.New().String()
		}
		if signal.Status == "" {
			signal.Status = "new"
		}
		if signal.DetectedAt.IsZero() {
			signal.DetectedAt = time.Now().UTC()
		}

		metadataJSON, _ := json.Marshal(signal.Metadata)

		_, err := s.pool.Exec(ctx, `
			INSERT INTO customer_signals
			(id, intel_item_id, signal_type, strength, content, platform, author, url, metadata, status, detected_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`, signal.ID, signal.IntelItemID, signal.SignalType, signal.Strength,
			signal.Content, signal.Platform, signal.Author, signal.URL,
			metadataJSON, signal.Status, signal.DetectedAt, time.Now().UTC())

		if err != nil {
			return fmt.Errorf("failed to save signal: %w", err)
		}
	}

	return nil
}

// SaveActions 批量保存营销动作
func (s *PostgresIntelligenceStorage) SaveActions(ctx context.Context, actions []MarketingAction) error {
	if len(actions) == 0 {
		return nil
	}

	for _, action := range actions {
		if action.ID == "" {
			action.ID = uuid.New().String()
		}
		if action.Status == "" {
			action.Status = "pending"
		}

		metadataJSON, _ := json.Marshal(action.Metadata)
		signalIDsJSON, _ := json.Marshal(action.SignalIDs)

		_, err := s.pool.Exec(ctx, `
			INSERT INTO marketing_actions
			(id, action_type, channel, title, content, template_id, target_audience, priority, signal_ids, auto_execute, customer_stage, qualified_score, metadata, status, scheduled_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		`, action.ID, action.ActionType, action.Channel, action.Title,
			action.Content, action.TemplateID, action.TargetAudience, action.Priority,
			signalIDsJSON, action.AutoExecute, action.CustomerStage, action.QualifiedScore,
			metadataJSON, action.Status, action.ScheduledAt, time.Now().UTC())

		if err != nil {
			return fmt.Errorf("failed to save action: %w", err)
		}
	}

	return nil
}

// GetPendingActions 获取待处理的营销动作
func (s *PostgresIntelligenceStorage) GetPendingActions(ctx context.Context, limit int) ([]MarketingAction, error) {
	query := `
		SELECT id, action_type, channel, title, content, template_id, target_audience, priority, signal_ids, auto_execute, customer_stage, qualified_score, metadata, status, scheduled_at, executed_at, created_at
		FROM marketing_actions
		WHERE status = 'pending'
		ORDER BY priority DESC, created_at ASC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []MarketingAction
	for rows.Next() {
		var action MarketingAction
		var metadataJSON []byte
		var signalIDsJSON []byte
		var qualifiedScoreNullable *float64

		err := rows.Scan(&action.ID, &action.ActionType, &action.Channel,
			&action.Title, &action.Content, &action.TemplateID, &action.TargetAudience, &action.Priority,
			&signalIDsJSON, &action.AutoExecute, &action.CustomerStage, &qualifiedScoreNullable,
			&metadataJSON, &action.Status, &action.ScheduledAt,
			&action.ExecutedAt, &action.CreatedAt)
		if err != nil {
			return nil, err
		}

		if qualifiedScoreNullable != nil {
			action.QualifiedScore = *qualifiedScoreNullable
		}

		if len(metadataJSON) > 0 {
			_ = json.Unmarshal(metadataJSON, &action.Metadata)
		}
		if len(signalIDsJSON) > 0 {
			_ = json.Unmarshal(signalIDsJSON, &action.SignalIDs)
		}

		actions = append(actions, action)
	}

	return actions, rows.Err()
}

// UpdateActionStatus 更新营销动作状态
func (s *PostgresIntelligenceStorage) UpdateActionStatus(ctx context.Context, actionID string, status string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE marketing_actions
		SET status = $1, executed_at = CASE WHEN $1 = 'executed' THEN NOW() ELSE executed_at END, updated_at = NOW()
		WHERE id = $2
	`, status, actionID)
	return err
}
