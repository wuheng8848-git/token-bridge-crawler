// Token Bridge Crawler - 厂商刊例价抓取服务
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"

	"token-bridge-crawler/internal"
	"token-bridge-crawler/internal/adapters"
	"token-bridge-crawler/internal/ai"
	"token-bridge-crawler/internal/mail"
	"token-bridge-crawler/internal/storage"
)

func main() {
	var (
		configPath = flag.String("config", "config.yaml", "配置文件路径")
		once       = flag.Bool("once", false, "只执行一次，不启动定时任务")
	)
	flag.Parse()

	// 加载 .env 文件（如果存在）
	if err := godotenv.Load(); err != nil {
		log.Printf("警告: 未找到 .env 文件，使用环境变量")
	}

	// 加载配置
	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建存储
	databaseURL := cfg.Storage.DatabaseURL
	if databaseURL == "" {
		databaseURL = os.Getenv("CRAWLER_DATABASE_URL")
	}
	if databaseURL == "" {
		log.Fatal("数据库 URL 未配置，请设置 CRAWLER_DATABASE_URL 环境变量或在 config.yaml 中配置")
	}
	
	store, err := storage.NewPostgresStorage(databaseURL)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer store.Close()

	// 创建厂商适配器
	var vendorAdapters []adapters.VendorAdapter
	
	if cfg.Vendors.Google.Enabled {
		vendorAdapters = append(vendorAdapters, adapters.NewGoogleAdapter())
	}
	if cfg.Vendors.OpenAI.Enabled && cfg.Vendors.OpenAI.APIKey != "" {
		vendorAdapters = append(vendorAdapters, adapters.NewOpenAIAdapter(cfg.Vendors.OpenAI.APIKey))
	}
	if cfg.Vendors.Anthropic.Enabled {
		vendorAdapters = append(vendorAdapters, adapters.NewAnthropicAdapter())
	}

	if len(vendorAdapters) == 0 {
		log.Fatal("没有启用的厂商适配器")
	}

	// 创建 AI 总结器（可选）
	var summarizer *ai.Summarizer
	if cfg.AIReport.Enabled {
		summarizer, err = ai.NewSummarizer(ai.Config{
			Provider: cfg.AIReport.Provider.Name,
			Model:    cfg.AIReport.Provider.Model,
			APIKey:   cfg.AIReport.Provider.APIKey,
			BaseURL:  cfg.AIReport.Provider.BaseURL,
			Prompt:   cfg.AIReport.PromptTemplate,
		})
		if err != nil {
			log.Printf("创建 AI 总结器失败: %v", err)
		}
	}

	// 创建邮件发送器（可选）
	var mailSender *mail.Sender
	if cfg.Email.Enabled {
		mailSender = mail.NewSender(mail.Config{
			Host:     cfg.Email.SMTP.Host,
			Port:     cfg.Email.SMTP.Port,
			Username: cfg.Email.SMTP.Username,
			Password: cfg.Email.SMTP.Password,
			TLS:      cfg.Email.SMTP.TLS,
			From:     cfg.Email.From,
			To:       cfg.Email.To,
			Cc:       cfg.Email.Cc,
			Subject:  cfg.Email.SubjectTemplate,
		})
	}

	// 创建调度器
	scheduler := internal.NewScheduler(internal.Config{
		Vendors:    vendorAdapters,
		Storage:    store,
		Summarizer: summarizer,
		MailSender: mailSender,
	})

	if *once {
		// 只执行一次
		log.Println("执行单次抓取任务")
		if err := scheduler.RunDaily(context.Background()); err != nil {
			log.Printf("执行失败: %v", err)
			os.Exit(1)
		}
		return
	}

	// 启动定时任务
	c := cron.New(cron.WithLocation(time.UTC))
	
	_, err = c.AddFunc(cfg.Scheduler.Cron, func() {
		log.Println("[Cron] 触发定时抓取任务")
		if err := scheduler.RunDaily(context.Background()); err != nil {
			log.Printf("[Cron] 执行失败: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("创建定时任务失败: %v", err)
	}

	c.Start()
	log.Printf("爬虫服务启动，定时规则: %s", cfg.Scheduler.Cron)

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("正在关闭爬虫服务...")
	c.Stop()
}

// Config 配置结构
type Config struct {
	Scheduler struct {
		Cron string `mapstructure:"cron"`
	} `mapstructure:"scheduler"`
	
	Vendors struct {
		Google struct {
			Enabled bool   `mapstructure:"enabled"`
			APIKey  string `mapstructure:"api_key"`
		} `mapstructure:"google"`
		OpenAI struct {
			Enabled bool   `mapstructure:"enabled"`
			APIKey  string `mapstructure:"api_key"`
		} `mapstructure:"openai"`
		Anthropic struct {
			Enabled bool `mapstructure:"enabled"`
		} `mapstructure:"anthropic"`
	} `mapstructure:"vendors"`
	
	Storage struct {
		DatabaseURL string `mapstructure:"database_url"`
	} `mapstructure:"storage"`
	
	AIReport struct {
		Enabled      bool `mapstructure:"enabled"`
		Provider     struct {
			Name    string `mapstructure:"name"`
			Model   string `mapstructure:"model"`
			APIKey  string `mapstructure:"api_key"`
			BaseURL string `mapstructure:"base_url"`
		} `mapstructure:"provider"`
		PromptTemplate string `mapstructure:"prompt_template"`
	} `mapstructure:"ai_report"`
	
	Email struct {
		Enabled bool `mapstructure:"enabled"`
		SMTP    struct {
			Host     string `mapstructure:"host"`
			Port     int    `mapstructure:"port"`
			Username string `mapstructure:"username"`
			Password string `mapstructure:"password"`
			TLS      bool   `mapstructure:"tls"`
		} `mapstructure:"smtp"`
		From            string   `mapstructure:"from"`
		To              []string `mapstructure:"to"`
		Cc              []string `mapstructure:"cc"`
		SubjectTemplate string   `mapstructure:"subject_template"`
	} `mapstructure:"email"`
}

func loadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv()

	// 设置默认值
	viper.SetDefault("scheduler.cron", "0 2 * * *")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
