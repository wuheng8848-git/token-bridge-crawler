// Package main 情报系统主入口
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"token-bridge-crawler/internal/ai"
	"token-bridge-crawler/internal/api"
	"token-bridge-crawler/internal/collectors/apidoc"
	"token-bridge-crawler/internal/collectors/community"
	"token-bridge-crawler/internal/collectors/conversion"
	"token-bridge-crawler/internal/collectors/integration"
	"token-bridge-crawler/internal/collectors/policy"
	"token-bridge-crawler/internal/collectors/price"
	"token-bridge-crawler/internal/collectors/search"
	"token-bridge-crawler/internal/collectors/tool"
	"token-bridge-crawler/internal/collectors/usage"
	"token-bridge-crawler/internal/collectors/useracquisition"
	"token-bridge-crawler/internal/collectors/userpain"
	"token-bridge-crawler/internal/core"
	"token-bridge-crawler/internal/marketing"
	"token-bridge-crawler/internal/reporter"
	"token-bridge-crawler/internal/rules"
	"token-bridge-crawler/internal/scheduler"
	"token-bridge-crawler/internal/storage"
)

func main() {
	// 加载 .env 文件（忽略错误，允许使用环境变量覆盖）
	if err := godotenv.Load(); err != nil {
		log.Printf("[Config] .env 文件未找到，使用环境变量")
	}

	// 解析命令行参数
	once := flag.Bool("once", false, "只执行一次，不启动定时任务")
	flag.Parse()

	// 加载配置
	config := loadConfig()

	// 初始化存储
	baseStore, err := storage.NewPostgresIntelligenceStorage(config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer baseStore.Close()

	// 初始化翻译服务
	var store storage.IntelligenceStorage = baseStore
	if config.TranslationEnabled {
		// 创建带有翻译器队列的翻译服务
		translationService := ai.NewTranslationServiceWithQueue(
			config.OpenRouterAPIKey,
			config.YoudaoAppKey,
			config.YoudaoAppSecret,
			config.BaiduAppID,
			config.BaiduAPIKey,
			config.BaiduAppSecret,
			config.VolcengineAccessKeyID,
			config.VolcengineSecretAccessKey,
		)
		translationService.SetBatchSize(config.TranslationBatchSize)

		// 使用保守翻译策略（只翻译高质量情报，节省API额度）
		strategy := storage.ConservativeTranslationStrategy()
		// 或者使用默认策略：strategy := storage.DefaultTranslationStrategy()
		translatedStore := storage.NewTranslatedStorageWithStrategy(baseStore, translationService, strategy)
		store = translatedStore
		log.Printf("[Translation] 翻译服务已启用，使用保守策略（质量分≥%.0f，每批最多%d条）",
			strategy.MinQualityScore, strategy.MaxItemsPerBatch)
	} else {
		log.Println("[Translation] 翻译服务已禁用")
	}

	// 初始化采集器注册表
	registry := core.NewCollectorRegistry()

	// 注册价格采集器
	registerPriceCollectors(registry, config)

	// 注册API文档采集器
	registerAPIDocCollectors(registry, config)

	// 注册政策变更采集器
	registerPolicyCollectors(registry, config)

	// 注册用户痛点采集器
	registerUserPainCollectors(registry, config)

	// 注册工具生态采集器
	registerToolEcosystemCollectors(registry, config)

	// 注册集成机会采集器
	registerIntegrationCollectors(registry, config)

	// 注册用户获取采集器
	registerUserAcquisitionCollectors(registry, config)

	// 注册转化情况采集器
	registerConversionCollectors(registry, config)

	// 注册使用模式采集器
	registerUsagePatternCollectors(registry, config)

	// 注册社区采集器（Discord 等）
	registerCommunityCollectors(registry, config)

	// 初始化规则引擎（使用数据库规则）
	var ruleEngine rules.RuleEngine
	if config.DatabaseURL != "" {
		// 尝试从数据库加载规则
		// 注意：这里使用独立的连接，因为规则引擎需要在整个生命周期中访问数据库
		db, err := sql.Open("postgres", config.DatabaseURL)
		if err == nil {
			// 测试连接
			if pingErr := db.Ping(); pingErr == nil {
				ruleStorage := rules.NewDBStorage(db)
				engine, err := rules.NewEngine(ruleStorage)
				if err != nil {
					log.Printf("[Rules] Failed to load rules from database, using defaults: %v", err)
					db.Close()
					ruleEngine = rules.NewEngineWithDefaults()
				} else {
					log.Println("[Rules] Loaded rules from database")
					ruleEngine = engine
					// 注意：db 连接不关闭，规则引擎需要持续使用
				}
			} else {
				log.Printf("[Rules] Failed to ping database, using defaults: %v", pingErr)
				db.Close()
				ruleEngine = rules.NewEngineWithDefaults()
			}
		} else {
			log.Printf("[Rules] Failed to connect to database for rules, using defaults: %v", err)
			ruleEngine = rules.NewEngineWithDefaults()
		}
	} else {
		ruleEngine = rules.NewEngineWithDefaults()
	}

	// 初始化调度器
	schedConfig := scheduler.SchedulerConfig{
		DefaultRateLimit: 5 * time.Second,
		MaxRetries:       3,
		RetryDelay:       10 * time.Second,
		CollectorConfigs: make(map[string]scheduler.CollectorConfig),
	}

	sched := scheduler.NewIntelligenceSchedulerWithEngine(registry, store, schedConfig, ruleEngine)

	// 初始化日报生成器
	dailyConfig := reporter.DailyReportConfig{
		Enabled:      true,
		Cron:         "0 9 * * *", // 每天上午9点
		Email:        false,       // 暂时禁用邮件
		EmailTo:      []string{"ops@tokenbridge.local"},
		EmailFrom:    "crawler@tokenbridge.local",
		EmailSubject: "Token Bridge 情报日报",
		IncludeTypes: []core.IntelType{
			// 供给侧信号
			core.IntelTypePrice,
			core.IntelTypeAPIDoc,
			core.IntelTypeProduct,
			core.IntelTypePolicy,
			// 需求侧信号
			core.IntelTypeCommunity,
			core.IntelTypeNews,
			core.IntelTypeUserPain,
			// 入口侧信号
			core.IntelTypeToolEcosystem,
			core.IntelTypeIntegration,
			// 自有经营信号
			core.IntelTypeUserAcquisition,
			core.IntelTypeConversion,
			core.IntelTypeUsagePattern,
		},
	}

	dailyReporter := reporter.NewDailyReporter(store, dailyConfig)

	// 初始化营销信号模型
	signalModel := marketing.NewSignalModel()
	log.Println("[Marketing] 营销信号模型已初始化")

	// 执行模式
	if *once {
		// 单次执行模式
		log.Println("[Intelligence] 执行单次采集...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		// 执行所有采集器
		if err := sched.ExecuteAllNow(ctx); err != nil {
			log.Printf("[Intelligence] 采集执行失败: %v", err)
		} else {
			log.Println("[Intelligence] 采集执行完成")
		}

		// 生成报告
		report, err := dailyReporter.Generate(ctx)
		if err != nil {
			log.Printf("[Intelligence] 报告生成失败: %v", err)
		} else {
			log.Println("[Intelligence] 报告生成完成:")
			log.Println(report)
		}

		// 处理营销信号
		log.Println("[Marketing] 开始处理营销信号...")

		// 查询最新采集的情报
		filter := storage.IntelFilter{
			Limit: 100,
		}
		items, err := store.GetItems(ctx, filter)
		if err != nil {
			log.Printf("[Marketing] 查询情报失败: %v", err)
		} else {
			log.Printf("[Marketing] 处理 %d 条情报...", len(items))

			// 收集所有信号和动作
			var allSignals []storage.CustomerSignal
			var allActions []storage.MarketingAction

			// 处理每条情报
			for _, item := range items {
				signals, actions, err := signalModel.ProcessIntel(item)
				if err != nil {
					log.Printf("[Marketing] 处理情报失败: %v", err)
					continue
				}

				// 输出信号和动作
				if len(signals) > 0 {
					log.Printf("[Marketing] 检测到 %d 个信号:", len(signals))
					for _, signal := range signals {
						log.Printf("  - %s (强度: %d)", signal.Type, signal.Strength)

						// 转换为存储结构
						dbSignal := storage.CustomerSignal{
							IntelItemID: &item.ID,
							SignalType:  string(signal.Type),
							Strength:    int(signal.Strength),
							Content:     signal.Content,
							Platform:    signal.Platform,
							Author:      signal.Author,
							URL:         signal.URL,
							Metadata:    signal.Metadata,
							Status:      "new",
							DetectedAt:  signal.DetectedAt,
						}
						allSignals = append(allSignals, dbSignal)
					}
				}

				if len(actions) > 0 {
					log.Printf("[Marketing] 生成 %d 个营销动作:", len(actions))
					for _, action := range actions {
						log.Printf("  - %s (渠道: %s, 优先级: %d)", action.Type, action.Channel, action.Priority)

						// 转换为存储结构
						dbAction := storage.MarketingAction{
							ActionType:     string(action.Type),
							Channel:        string(action.Channel),
							Title:          action.Title,
							Content:        action.Content,
							TemplateID:     action.TemplateID,
							TargetAudience: action.TargetAudience,
							Priority:       action.Priority,
							SignalIDs:      action.SignalIDs,
							AutoExecute:    action.AutoExecute,
							CustomerStage:  string(action.CustomerStage),
							QualifiedScore: action.QualifiedScore,
							Metadata:       action.Metadata,
							Status:         action.Status,
							ScheduledAt:    action.ScheduledAt,
						}
						allActions = append(allActions, dbAction)
					}
				}
			}

			// 持久化信号
			if len(allSignals) > 0 {
				if err := store.SaveSignals(ctx, allSignals); err != nil {
					log.Printf("[Marketing] 保存信号失败: %v", err)
				} else {
					log.Printf("[Marketing] 已保存 %d 个信号", len(allSignals))
				}
			}

			// 持久化动作
			if len(allActions) > 0 {
				if err := store.SaveActions(ctx, allActions); err != nil {
					log.Printf("[Marketing] 保存动作失败: %v", err)
				} else {
					log.Printf("[Marketing] 已保存 %d 个动作", len(allActions))
				}
			}
		}

		log.Println("[Marketing] 营销信号处理完成")

	} else {
		// 持续运行模式
		log.Println("[Intelligence] 启动情报系统...")

		// 启动调度器
		if err := sched.Start(); err != nil {
			log.Fatalf("[Intelligence] 启动调度器失败: %v", err)
		}

		// 启动 API 服务（使用 8081 端口，避免与主项目 8080 冲突）
		apiPort := getEnv("API_PORT", "8081")
		apiServer := api.NewServer(registry, store, sched, apiPort)
		apiServer.Start()

		// 等待中断信号
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		log.Println("[Intelligence] 系统运行中，按 Ctrl+C 退出...")
		log.Printf("[Intelligence] API 地址: http://localhost:%s", apiPort)
		<-sigChan

		// 停止系统
		log.Println("[Intelligence] 正在停止...")
		sched.Stop()
		log.Println("[Intelligence] 系统已停止")
	}
}

// Config 配置结构
type Config struct {
	DatabaseURL               string
	StaticDir                 string
	OpenRouterAPIKey          string
	YoudaoAppKey              string
	YoudaoAppSecret           string
	BaiduAppID                string
	BaiduAPIKey               string // 大模型翻译API Key
	BaiduAppSecret            string // 通用翻译密钥
	VolcengineAccessKeyID     string // 火山引擎 Access Key ID
	VolcengineSecretAccessKey string // 火山引擎 Secret Access Key
	TranslationEnabled        bool
	TranslationBatchSize      int
	// Discord 配置
	DiscordBotToken   string
	DiscordChannelIDs []string
	// Tavily 搜索配置
	TavilyAPIKey string
}

// loadConfig 加载配置
func loadConfig() *Config {
	config := &Config{
		DatabaseURL:               getEnv("CRAWLER_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/tokenbridge?sslmode=disable"),
		StaticDir:                 getEnv("CRAWLER_STATIC_DIR", "./data"),
		OpenRouterAPIKey:          getEnv("OPENROUTER_API_KEY", ""),
		YoudaoAppKey:              getEnv("YOUDAO_APP_KEY", ""),
		YoudaoAppSecret:           getEnv("YOUDAO_APP_SECRET", ""),
		BaiduAppID:                getEnv("BAIDU_APP_ID", ""),
		BaiduAPIKey:               getEnv("BAIDU_API_KEY", ""),
		BaiduAppSecret:            getEnv("BAIDU_APP_SECRET", ""),
		VolcengineAccessKeyID:     getEnv("VOLCENGINE_ACCESS_KEY_ID", ""),
		VolcengineSecretAccessKey: getEnv("VOLCENGINE_SECRET_ACCESS_KEY", ""),
		TranslationEnabled:        getEnvBool("TRANSLATION_ENABLED", false),
		TranslationBatchSize:      getEnvInt("TRANSLATION_BATCH_SIZE", 5),
		DiscordBotToken:           getEnv("DISCORD_BOT_TOKEN", ""),
		DiscordChannelIDs:         getEnvList("DISCORD_CHANNEL_IDS", []string{}),
		TavilyAPIKey:              getEnv("TAVILY_API_KEY", ""),
	}

	// 确保静态目录存在
	if err := os.MkdirAll(config.StaticDir, 0755); err != nil {
		log.Printf("[Config] 创建静态目录失败: %v", err)
	}

	return config
}

// getEnv 获取环境变量
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool 获取布尔类型的环境变量
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

// getEnvInt 获取整数类型的环境变量
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}

// getEnvList 获取逗号分隔的列表类型环境变量
func getEnvList(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// 分割逗号并去除空白
		items := strings.Split(value, ",")
		result := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item != "" {
				result = append(result, item)
			}
		}
		return result
	}
	return defaultValue
}

// registerPriceCollectors 注册价格采集器
func registerPriceCollectors(registry *core.CollectorRegistry, config *Config) {
	// OpenAI 价格采集器
	openaiStatic := config.StaticDir + "/openai_prices.json"
	openaiCollector := price.NewOpenAICollector(openaiStatic)
	registry.Register(openaiCollector)
	log.Println("[Registry] 注册 OpenAI 价格采集器")

	// Google 价格采集器
	googleStatic := config.StaticDir + "/google_prices.json"
	googleCollector := price.NewGoogleCollector(googleStatic)
	registry.Register(googleCollector)
	log.Println("[Registry] 注册 Google 价格采集器")

	// Anthropic 价格采集器
	anthropicStatic := config.StaticDir + "/anthropic_prices.json"
	anthropicCollector := price.NewAnthropicCollector(anthropicStatic)
	registry.Register(anthropicCollector)
	log.Println("[Registry] 注册 Anthropic 价格采集器")

	// OpenRouter 市场情报采集器
	openrouterCollector := price.NewOpenRouterCollector()
	registry.Register(openrouterCollector)
	log.Println("[Registry] 注册 OpenRouter 市场情报采集器")
}

// registerAPIDocCollectors 注册API文档采集器
func registerAPIDocCollectors(registry *core.CollectorRegistry, config *Config) {
	// OpenAI API文档采集器
	openaiAPIDocCollector := apidoc.NewOpenAIAPIDocCollector()
	registry.Register(openaiAPIDocCollector)
	log.Println("[Registry] 注册 OpenAI API文档采集器")

	// 可以在这里添加其他厂商的API文档采集器
	// Google API文档采集器
	// Anthropic API文档采集器
}

// registerPolicyCollectors 注册政策变更采集器
func registerPolicyCollectors(registry *core.CollectorRegistry, config *Config) {
	// 基础政策采集器
	policyCollector := policy.NewBasePolicyCollector("policy_collector", "system", 24*time.Hour)
	registry.Register(policyCollector)
	log.Println("[Registry] 注册基础政策变更采集器")
}

// registerUserPainCollectors 注册用户痛点采集器
func registerUserPainCollectors(registry *core.CollectorRegistry, config *Config) {
	// HackerNews用户痛点采集器
	hnCollector := userpain.NewHackerNewsCollector()
	registry.Register(hnCollector)
	log.Println("[Registry] 注册 HackerNews 用户痛点采集器")

	// Reddit用户痛点采集器
	redditCollector := userpain.NewRedditCollector()
	registry.Register(redditCollector)
	log.Println("[Registry] 注册 Reddit 用户痛点采集器")

	// StackExchange 用户痛点采集器 - 已停用（数据质量差，标题为空）
	// stackexchangeCollector := userpain.NewStackExchangeCollector()
	// registry.Register(stackexchangeCollector)
	// log.Println("[Registry] 注册 StackExchange 用户痛点采集器")

	// OpenAI 社区论坛采集器 - 已停用（数据质量差）
	// openaiCommunityCollector := userpain.NewOpenAICommunityCollector()
	// registry.Register(openaiCommunityCollector)
	// log.Println("[Registry] 注册 OpenAI 社区论坛采集器")

	// Dev.to 采集器 - 已停用（待优化）
	// devtoCollector := userpain.NewDevToCollector()
	// registry.Register(devtoCollector)
	// log.Println("[Registry] 注册 Dev.to 用户痛点采集器")

	// 配置痛点采集器
	configPainCollector := userpain.NewConfigPainCollector()
	registry.Register(configPainCollector)
	log.Println("[Registry] 注册配置痛点采集器")

	// Tavily 搜索采集器（需要 API Key）
	if config.TavilyAPIKey != "" {
		tavilyCollector := search.NewTavilyCollector(config.TavilyAPIKey)
		registry.Register(tavilyCollector)
		log.Println("[Registry] 注册 Tavily 搜索采集器")
	} else {
		log.Println("[Registry] Tavily API Key 未配置，跳过注册")
	}
}

// registerToolEcosystemCollectors 注册工具生态采集器
func registerToolEcosystemCollectors(registry *core.CollectorRegistry, config *Config) {
	// 工具生态采集器
	ecosystemCollector := tool.NewEcosystemCollector()
	registry.Register(ecosystemCollector)
	log.Println("[Registry] 注册工具生态采集器")
}

// registerIntegrationCollectors 注册集成机会采集器
func registerIntegrationCollectors(registry *core.CollectorRegistry, config *Config) {
	// 基础集成机会采集器
	integrationCollector := integration.NewBaseIntegrationCollector("integration_collector", "system", 24*time.Hour)
	registry.Register(integrationCollector)
	log.Println("[Registry] 注册基础集成机会采集器")
}

// registerUserAcquisitionCollectors 注册用户获取采集器
func registerUserAcquisitionCollectors(registry *core.CollectorRegistry, config *Config) {
	// 基础用户获取采集器
	userAcquisitionCollector := useracquisition.NewBaseUserAcquisitionCollector("user_acquisition_collector", "system", 12*time.Hour)
	registry.Register(userAcquisitionCollector)
	log.Println("[Registry] 注册基础用户获取采集器")
}

// registerConversionCollectors 注册转化情况采集器
func registerConversionCollectors(registry *core.CollectorRegistry, config *Config) {
	// 基础转化情况采集器
	conversionCollector := conversion.NewBaseConversionCollector("conversion_collector", "system", 12*time.Hour)
	registry.Register(conversionCollector)
	log.Println("[Registry] 注册基础转化情况采集器")
}

// registerUsagePatternCollectors 注册使用模式采集器
func registerUsagePatternCollectors(registry *core.CollectorRegistry, config *Config) {
	// 基础使用模式采集器
	usageCollector := usage.NewBaseUsagePatternCollector("usage_pattern_collector", "system", 12*time.Hour)
	registry.Register(usageCollector)
	log.Println("[Registry] 注册基础使用模式采集器")
}

// registerCommunityCollectors 注册社区采集器
func registerCommunityCollectors(registry *core.CollectorRegistry, config *Config) {
	// Discord 采集器（真实 API）
	if config.DiscordBotToken != "" && len(config.DiscordChannelIDs) > 0 {
		discordCollector, err := community.NewDiscordCollector(community.DiscordConfig{
			BotToken:   config.DiscordBotToken,
			ChannelIDs: config.DiscordChannelIDs,
		})
		if err != nil {
			log.Printf("[Registry] 创建 Discord 采集器失败: %v", err)
		} else {
			registry.Register(discordCollector)
			log.Printf("[Registry] 注册 Discord 采集器（监控 %d 个频道）", len(config.DiscordChannelIDs))
		}
	} else {
		log.Println("[Registry] Discord 采集器未配置，跳过注册")
	}
}
