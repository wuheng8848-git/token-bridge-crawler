// Package internal 提供爬虫调度功能
package internal

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"token-bridge-crawler/internal/adapters"
	"token-bridge-crawler/internal/ai"
	"token-bridge-crawler/internal/mail"
	"token-bridge-crawler/internal/storage"

	"github.com/google/uuid"
)

// Scheduler 爬虫调度器
type Scheduler struct {
	vendors      []adapters.VendorAdapter
	storage      storage.Storage
	summarizer   *ai.Summarizer
	mailSender   *mail.Sender
	mainProject  *TBClient
	pushBatch    int
	vendorStates map[string]*VendorState
}

// VendorState 厂商调度状态
type VendorState struct {
	LastAttempt         time.Time
	LastSuccess         time.Time
	ConsecutiveFailures int
	CurrentInterval     time.Duration
}

// Config 调度器配置
type Config struct {
	Vendors    []adapters.VendorAdapter
	Storage    storage.Storage
	Summarizer *ai.Summarizer
	MailSender *mail.Sender
	// MainProject 非 nil 时，抓取成功写入本地库后推送 POST /v1/admin/supplier_catalog_staging/import（Bearer 见 TB_ADMIN_API_TOKEN）。
	MainProject *TBClient
	// PushBatchSize ImportPrices 每批条数，≤0 时默认 50。
	PushBatchSize int
}

// NewScheduler 创建调度器
func NewScheduler(cfg Config) *Scheduler {
	states := make(map[string]*VendorState)
	for _, v := range cfg.Vendors {
		states[v.Name()] = &VendorState{
			CurrentInterval: 24 * time.Hour,
		}
	}

	batch := cfg.PushBatchSize
	if batch <= 0 {
		batch = 50
	}

	return &Scheduler{
		vendors:      cfg.Vendors,
		storage:      cfg.Storage,
		summarizer:   cfg.Summarizer,
		mailSender:   cfg.MailSender,
		mainProject:  cfg.MainProject,
		pushBatch:    batch,
		vendorStates: states,
	}
}

// RunDaily 执行每日抓取
func (s *Scheduler) RunDaily(ctx context.Context) error {
	log.Println("[Scheduler] 开始每日抓取任务")

	for _, vendor := range s.vendors {
		state := s.vendorStates[vendor.Name()]

		if !s.shouldRunNow(vendor.Name()) {
			log.Printf("[Scheduler] %s 跳过，距离下次执行还有 %s",
				vendor.DisplayName(), s.timeUntilNextRun(vendor.Name()))
			continue
		}

		log.Printf("[Scheduler] 开始抓取 %s", vendor.DisplayName())
		snapshot, err := s.crawlVendor(ctx, vendor)

		state.LastAttempt = time.Now()
		if err != nil {
			state.ConsecutiveFailures++
			state.CurrentInterval = s.calculateBackoff(state.ConsecutiveFailures)
			log.Printf("[Scheduler] %s 抓取失败: %v, 连续失败 %d 次, 下次间隔: %s",
				vendor.DisplayName(), err, state.ConsecutiveFailures, state.CurrentInterval)
		} else {
			state.LastSuccess = time.Now()
			state.ConsecutiveFailures = 0
			state.CurrentInterval = 24 * time.Hour
			log.Printf("[Scheduler] %s 抓取成功, 共 %d 个模型",
				vendor.DisplayName(), snapshot.TotalModels)
		}

		time.Sleep(vendor.RateLimit())
	}

	log.Println("[Scheduler] 每日抓取任务完成")
	return nil
}

func (s *Scheduler) shouldRunNow(vendorName string) bool {
	state := s.vendorStates[vendorName]
	if state.LastAttempt.IsZero() {
		return true
	}
	return time.Since(state.LastAttempt) >= state.CurrentInterval
}

func (s *Scheduler) timeUntilNextRun(vendorName string) time.Duration {
	state := s.vendorStates[vendorName]
	nextRun := state.LastAttempt.Add(state.CurrentInterval)
	return time.Until(nextRun)
}

func (s *Scheduler) calculateBackoff(failures int) time.Duration {
	days := math.Pow(2, float64(failures))
	if days > 7 {
		days = 7
	}
	return time.Duration(days) * 24 * time.Hour
}

func (s *Scheduler) crawlVendor(ctx context.Context, vendor adapters.VendorAdapter) (*storage.VendorPriceSnapshot, error) {
	snapshotDate := time.Now().UTC().Truncate(24 * time.Hour)
	snapshot := &storage.VendorPriceSnapshot{
		ID:           uuid.New().String(),
		Vendor:       vendor.Name(),
		SnapshotDate: snapshotDate,
		SnapshotAt:   time.Now().UTC(),
		Status:       "success",
	}

	prices, err := vendor.Fetch(ctx)
	if err != nil {
		snapshot.Status = "failed"
		snapshot.ErrorLog = err.Error()
		_ = s.storage.SaveSnapshot(ctx, snapshot)
		return snapshot, err
	}

	snapshot.TotalModels = len(prices)

	prevSnapshot, _ := s.storage.GetLatestSnapshot(ctx, vendor.Name())
	var prevPrices map[string]storage.VendorPriceDetail
	if prevSnapshot != nil {
		prevDetails, _ := s.storage.GetPriceHistory(ctx, vendor.Name(), "", 1)
		prevPrices = make(map[string]storage.VendorPriceDetail)
		for _, d := range prevDetails {
			prevPrices[d.ModelCode] = d
		}
	}

	details := storage.AdapterPricesToDetails(snapshot.ID, snapshotDate, prices, prevPrices)

	for _, d := range details {
		switch d.ChangeType {
		case "new":
			snapshot.NewModels++
		case "updated":
			snapshot.UpdatedModels++
		}
	}

	if err := s.storage.SaveSnapshot(ctx, snapshot); err != nil {
		return snapshot, fmt.Errorf("save snapshot failed: %w", err)
	}

	if err := s.storage.SavePriceDetails(ctx, details); err != nil {
		return snapshot, fmt.Errorf("save details failed: %w", err)
	}

	if s.mainProject != nil && len(prices) > 0 {
		source := fmt.Sprintf("%s-%s", vendor.Name(), snapshotDate.Format("2006-01-02"))
		if err := s.mainProject.ImportPrices(ctx, prices, source, nil, s.pushBatch); err != nil {
			log.Printf("[Scheduler] 推送主系统 supplier_catalog_staging 失败: %v", err)
		} else {
			log.Printf("[Scheduler] 已推送主系统 supplier_catalog_staging: vendor=%s models=%d", vendor.Name(), len(prices))
		}
	}

	if s.summarizer != nil && s.mailSender != nil {
		summary, err := s.summarizer.GenerateReportFromDetails(vendor.Name(), snapshotDate, details)
		if err != nil {
			log.Printf("[Scheduler] AI 总结生成失败: %v", err)
		} else {
			reportData := mail.ReportData{
				Vendor:  vendor.Name(),
				Date:    snapshotDate.Format("2006-01-02"),
				Summary: summary,
				Details: details,
			}
			if err := s.mailSender.SendReport(reportData, true); err != nil {
				log.Printf("[Scheduler] 邮件发送失败: %v", err)
			}
		}
	}

	return snapshot, nil
}
