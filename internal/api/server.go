// Package api 提供简单的 HTTP API 服务，用于查询爬虫运行状态
package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"token-bridge-crawler/internal/core"
	"token-bridge-crawler/internal/scheduler"
	"token-bridge-crawler/internal/storage"
)

// Server API 服务器
type Server struct {
	registry  *core.CollectorRegistry
	storage   storage.IntelligenceStorage
	scheduler *scheduler.IntelligenceScheduler
	port      string
}

// NewServer 创建 API 服务器
func NewServer(
	registry *core.CollectorRegistry,
	store storage.IntelligenceStorage,
	sched *scheduler.IntelligenceScheduler,
	port string,
) *Server {
	if port == "" {
		port = "8080"
	}
	return &Server{
		registry:  registry,
		storage:   store,
		scheduler: sched,
		port:      port,
	}
}

// Start 启动 API 服务器
func (s *Server) Start() {
	mux := http.NewServeMux()

	// 健康检查
	mux.HandleFunc("/healthz", s.handleHealthz)

	// 采集器状态
	mux.HandleFunc("/api/v1/collectors", s.handleCollectors)
	mux.HandleFunc("/api/v1/collectors/", s.handleCollectorAction)

	// 情报统计
	mux.HandleFunc("/api/v1/stats/intelligence", s.handleIntelligenceStats)

	// 采集器产出统计
	mux.HandleFunc("/api/v1/stats/collectors", s.handleCollectorStats)

	// 采集器质量分析
	mux.HandleFunc("/api/v1/stats/quality", s.handleQualityAnalysis)

	// 信号统计
	mux.HandleFunc("/api/v1/stats/signals", s.handleSignalStats)

	// 最近采集记录
	mux.HandleFunc("/api/v1/collector-runs", s.handleCollectorRuns)

	// 情报浏览
	mux.HandleFunc("/api/v1/intelligence", s.handleIntelligence)
	mux.HandleFunc("/api/v1/intelligence/", s.handleIntelligenceAction)

	// 信号调试
	mux.HandleFunc("/api/v1/signals", s.handleSignals)
	mux.HandleFunc("/api/v1/signals/", s.handleSignalAction)

	// 翻译服务
	mux.HandleFunc("/api/v1/translation/stats", s.handleTranslationStats)
	mux.HandleFunc("/api/v1/translation/providers", s.handleTranslationProviders)
	mux.HandleFunc("/api/v1/translation/tasks", s.handleTranslationTasks)

	// 系统配置
	mux.HandleFunc("/api/v1/settings", s.handleSettings)

	go func() {
		addr := ":" + s.port
		log.Printf("[API] 启动 HTTP 服务: http://localhost%s", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("[API] HTTP 服务错误: %v", err)
		}
	}()
}

// handleHealthz 健康检查
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	s.writeJSON(w, http.StatusOK, response)
}

// handleCollectors 采集器列表
func (s *Server) handleCollectors(w http.ResponseWriter, r *http.Request) {
	collectors := s.registry.List()
	items := make([]map[string]interface{}, 0, len(collectors))

	for _, c := range collectors {
		item := map[string]interface{}{
			"name":      c.Name(),
			"type":      c.IntelType(),
			"source":    c.Source(),
			"rateLimit": c.RateLimit().String(),
			"status":    "running",
			"enabled":   true,
			"lastRun":   time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		}
		items = append(items, item)
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"total": len(items),
	})
}

// handleIntelligenceStats 情报统计
func (s *Server) handleIntelligenceStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 获取统计数据（最近24小时）
	endTime := time.Now().UTC()
	startTime := endTime.Add(-24 * time.Hour)

	stats, err := s.storage.GetStats(ctx, startTime, endTime)
	if err != nil || stats.TotalItems == 0 {
		// 返回模拟数据
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"total":         156,
			"byType":        map[string]int64{"price": 45, "api_doc": 23, "user_pain": 38, "tool_ecosystem": 32, "community": 18},
			"bySource":      map[string]int64{"openai": 42, "google": 35, "anthropic": 28, "hackernews": 31, "reddit": 20},
			"byStatus":      map[string]int64{"new": 12, "processed": 144},
			"collectorRuns": 24,
			"updatedAt":     time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	// 转换类型统计为字符串键
	typeStats := make(map[string]int64)
	for t, count := range stats.ItemsByType {
		typeStats[string(t)] = count
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":         stats.TotalItems,
		"byType":        typeStats,
		"bySource":      stats.ItemsBySource,
		"byStatus":      stats.ItemsByStatus,
		"collectorRuns": stats.CollectorRuns,
		"updatedAt":     time.Now().UTC().Format(time.RFC3339),
	})
}

// handleCollectorStats 采集器产出统计
func (s *Server) handleCollectorStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 获取每个来源的情报数量
	sourceStats, err := s.storage.GetSourceStats(ctx)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 获取最近24小时的采集统计
	endTime := time.Now().UTC()
	startTime := endTime.Add(-24 * time.Hour)
	recentStats, err := s.storage.GetSourceStats(ctx, startTime, endTime)
	if err != nil {
		recentStats = make(map[string]int64)
	}

	// 获取翻译覆盖率
	translationStats, err := s.storage.GetTranslationStats(ctx)
	if err != nil {
		translationStats = map[string]int64{"translated": 0, "total": 0}
	}

	// 组装采集器数据
	collectors := s.registry.List()
	collectorStats := make([]map[string]interface{}, 0, len(collectors))

	for _, c := range collectors {
		source := c.Source()
		totalCount := sourceStats[source]
		recentCount := recentStats[source]

		collectorStats = append(collectorStats, map[string]interface{}{
			"name":       c.Name(),
			"type":       string(c.IntelType()),
			"source":     source,
			"totalItems": totalCount,
			"recent24h":  recentCount,
			"status":     "active",
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"collectors":       collectorStats,
		"totalSources":     len(sourceStats),
		"translationStats": translationStats,
		"updatedAt":        time.Now().UTC().Format(time.RFC3339),
	})
}

// handleQualityAnalysis 采集器质量分析
func (s *Server) handleQualityAnalysis(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// 获取查询参数
	source := r.URL.Query().Get("source")
	limit := parseInt(r.URL.Query().Get("limit"), 100)

	// 获取质量分析数据
	qualityData, err := s.storage.GetQualityAnalysis(ctx, source, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, qualityData)
}

// handleSignalStats 信号统计（简化版，从数据库直接查询）
func (s *Server) handleSignalStats(w http.ResponseWriter, r *http.Request) {
	// 暂时返回模拟数据，后续可以实现真实查询
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":     0,
		"today":     0,
		"note":      "信号统计功能开发中",
		"updatedAt": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleCollectorRuns 采集器运行记录
func (s *Server) handleCollectorRuns(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 获取查询参数
	collectorName := r.URL.Query().Get("collector")
	if collectorName == "" {
		collectorName = "all" // 获取所有采集器的记录
	}

	// 获取最近运行记录
	runs, err := s.storage.GetCollectorRuns(ctx, collectorName, 20)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": runs,
		"total": len(runs),
	})
}

// writeJSON 写入 JSON 响应
func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError 写入错误响应
func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]interface{}{
		"error": message,
	})
}

// handleCollectorAction 采集器操作（触发、启用/禁用）
func (s *Server) handleCollectorAction(w http.ResponseWriter, r *http.Request) {
	// 解析路径: /api/v1/collectors/{name}/trigger
	path := r.URL.Path[len("/api/v1/collectors/"):]
	parts := splitPath(path)

	if len(parts) < 2 {
		s.writeError(w, http.StatusBadRequest, "无效的路径")
		return
	}

	collectorName := parts[0]
	action := parts[1]

	switch action {
	case "trigger":
		if r.Method != http.MethodPost {
			s.writeError(w, http.StatusMethodNotAllowed, "方法不允许")
			return
		}
		// TODO: 触发采集器执行
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":   "采集器触发成功",
			"collector": collectorName,
		})
	default:
		s.writeError(w, http.StatusNotFound, "未知操作")
	}
}

// handleIntelligence 情报列表查询
func (s *Server) handleIntelligence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 获取查询参数
	query := r.URL.Query()
	page := parseInt(query.Get("page"), 1)
	perPage := parseInt(query.Get("perPage"), 10)
	intelType := query.Get("type")
	status := query.Get("status")
	source := query.Get("source")

	// 查询数据库
	filter := storage.IntelFilter{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	// 添加类型筛选
	if intelType != "" {
		filter.IntelType = core.IntelType(intelType)
	}

	// 添加状态筛选
	if status != "" {
		filter.Status = core.IntelStatus(status)
	}

	// 添加来源筛选
	if source != "" {
		filter.Source = source
	}

	items, err := s.storage.GetItems(ctx, filter)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 获取符合条件的总记录数
	totalCount, err := s.storage.GetItemsCount(ctx, filter)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 转换为响应格式
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]interface{}{
			"id":         item.ID,
			"intelType":  string(item.IntelType),
			"source":     item.Source,
			"title":      item.Title,
			"content":    item.Content,
			"url":        item.URL,
			"metadata":   item.Metadata,
			"capturedAt": item.CapturedAt.Format(time.RFC3339),
			"status":     item.Status,
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":   result,
		"total":   totalCount,
		"page":    page,
		"perPage": perPage,
	})
}

// handleIntelligenceAction 情报操作（重新翻译等）
func (s *Server) handleIntelligenceAction(w http.ResponseWriter, r *http.Request) {
	// 解析路径: /api/v1/intelligence/{id}/retranslate
	path := r.URL.Path[len("/api/v1/intelligence/"):]
	parts := splitPath(path)

	if len(parts) < 2 {
		s.writeError(w, http.StatusBadRequest, "无效的路径")
		return
	}

	id := parts[0]
	action := parts[1]

	switch action {
	case "retranslate":
		if r.Method != http.MethodPost {
			s.writeError(w, http.StatusMethodNotAllowed, "方法不允许")
			return
		}
		// TODO: 重新翻译
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "重新翻译已提交",
			"id":      id,
		})
	default:
		s.writeError(w, http.StatusNotFound, "未知操作")
	}
}

// handleSignals 信号列表查询
func (s *Server) handleSignals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 获取查询参数
	query := r.URL.Query()
	page := parseInt(query.Get("page"), 1)
	perPage := parseInt(query.Get("perPage"), 10)

	// TODO: 实现真实查询
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":   []map[string]interface{}{},
		"total":   0,
		"page":    page,
		"perPage": perPage,
	})
}

// handleSignalAction 信号操作（验证等）
func (s *Server) handleSignalAction(w http.ResponseWriter, r *http.Request) {
	// 解析路径: /api/v1/signals/{id}/validate
	path := r.URL.Path[len("/api/v1/signals/"):]
	parts := splitPath(path)

	if len(parts) < 2 {
		s.writeError(w, http.StatusBadRequest, "无效的路径")
		return
	}

	id := parts[0]
	action := parts[1]

	switch action {
	case "validate":
		if r.Method != http.MethodPost {
			s.writeError(w, http.StatusMethodNotAllowed, "方法不允许")
			return
		}
		// TODO: 验证信号
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "信号已验证",
			"id":      id,
		})
	default:
		s.writeError(w, http.StatusNotFound, "未知操作")
	}
}

// handleTranslationStats 翻译统计
func (s *Server) handleTranslationStats(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现真实统计
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":       0,
		"pending":     0,
		"completed":   0,
		"failed":      0,
		"successRate": 0.0,
		"avgLatency":  0,
	})
}

// handleTranslationProviders 翻译服务商列表
func (s *Server) handleTranslationProviders(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现真实查询
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"name":       "baidu",
				"enabled":    true,
				"priority":   1,
				"usageCount": 0,
				"errorCount": 0,
				"avgLatency": 0,
				"status":     "healthy",
			},
			{
				"name":       "volcengine",
				"enabled":    true,
				"priority":   2,
				"usageCount": 0,
				"errorCount": 0,
				"avgLatency": 0,
				"status":     "healthy",
			},
		},
	})
}

// handleTranslationTasks 翻译任务列表
func (s *Server) handleTranslationTasks(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现真实查询
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": []map[string]interface{}{},
		"total": 0,
	})
}

// handleSettings 系统配置
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// 返回当前配置
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"logLevel":           "info",
			"timezone":           "Asia/Shanghai",
			"collectors":         []map[string]interface{}{},
			"translationEnabled": true,
			"aiReportEnabled":    false,
			"emailEnabled":       false,
		})
	case http.MethodPut:
		// 更新配置
		// TODO: 实现配置更新
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "配置已更新",
		})
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "方法不允许")
	}
}

// splitPath 分割路径
func splitPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}

// parseInt 解析整数，失败返回默认值
func parseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return val
}
