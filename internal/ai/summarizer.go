// Package ai 提供 AI 日报生成功能
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"

	"token-bridge-crawler/internal/storage"
)

// Summarizer AI 总结器
type Summarizer struct {
	provider   string // openai, deepseek, anthropic
	model      string
	apiKey     string
	baseURL    string
	template   *template.Template
	httpClient *http.Client
}

// Config AI 配置
type Config struct {
	Provider string
	Model    string
	APIKey   string
	BaseURL  string
	Prompt   string
}

// NewSummarizer 创建 AI 总结器
func NewSummarizer(cfg Config) (*Summarizer, error) {
	tmpl, err := template.New("report").Parse(cfg.Prompt)
	if err != nil {
		return nil, err
	}

	return &Summarizer{
		provider:   cfg.Provider,
		model:      cfg.Model,
		apiKey:     cfg.APIKey,
		baseURL:    cfg.BaseURL,
		template:   tmpl,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

// ReportData 报告数据
type ReportData struct {
	Vendor          string
	Date            string
	TotalModels     int
	NewModels       int
	NewList         string
	UpdatedModels   int
	IncreasedModels int
	AvgIncreasePct  float64
	DecreasedModels int
	AvgDecreasePct  float64
}

// GenerateReport 生成日报
func (s *Summarizer) GenerateReport(ctx context.Context, data ReportData) (string, error) {
	var promptBuf bytes.Buffer
	if err := s.template.Execute(&promptBuf, data); err != nil {
		return "", err
	}

	switch s.provider {
	case "openai":
		return s.callOpenAI(ctx, promptBuf.String())
	case "deepseek":
		return s.callDeepSeek(ctx, promptBuf.String())
	case "anthropic":
		return s.callAnthropic(ctx, promptBuf.String())
	default:
		return "", fmt.Errorf("unsupported provider: %s", s.provider)
	}
}

// GenerateReportFromDetails 从价格明细生成报告
func (s *Summarizer) GenerateReportFromDetails(vendor string, date time.Time, details []storage.VendorPriceDetail) (string, error) {
	data := ReportData{
		Vendor:      vendor,
		Date:        date.Format("2006-01-02"),
		TotalModels: len(details),
	}

	var newModels []string
	var increasedCount, decreasedCount int
	var totalIncreasePct, totalDecreasePct float64

	for _, d := range details {
		switch d.ChangeType {
		case "new":
			data.NewModels++
			newModels = append(newModels, d.ModelCode)
		case "updated":
			data.UpdatedModels++
			// 计算涨跌
			if d.PrevPrice != nil {
				var prev map[string]float64
				if err := json.Unmarshal(d.PrevPrice, &prev); err == nil {
					oldOutput := prev["output"]
					if oldOutput > 0 {
						changePct := ((d.OutputUSDPerMillion - oldOutput) / oldOutput) * 100
						if changePct > 0 {
							increasedCount++
							totalIncreasePct += changePct
						} else if changePct < 0 {
							decreasedCount++
							totalDecreasePct += -changePct
						}
					}
				}
			}
		}
	}

	data.NewList = strings.Join(newModels, ", ")
	data.IncreasedModels = increasedCount
	data.DecreasedModels = decreasedCount

	if increasedCount > 0 {
		data.AvgIncreasePct = totalIncreasePct / float64(increasedCount)
	}
	if decreasedCount > 0 {
		data.AvgDecreasePct = totalDecreasePct / float64(decreasedCount)
	}

	return s.GenerateReport(context.Background(), data)
}

// OpenAI 请求/响应结构
type openAIRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type openAIResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
}

func (s *Summarizer) callOpenAI(ctx context.Context, prompt string) (string, error) {
	reqBody := openAIRequest{
		Model: s.model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/v1/chat/completions", bytes.NewReader(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("no response from AI")
}

// DeepSeek 使用与 OpenAI 兼容的 API 格式
func (s *Summarizer) callDeepSeek(ctx context.Context, prompt string) (string, error) {
	// DeepSeek API 与 OpenAI 兼容
	return s.callOpenAI(ctx, prompt)
}

// Anthropic Claude API
func (s *Summarizer) callAnthropic(ctx context.Context, prompt string) (string, error) {
	// TODO: 实现 Claude API 调用
	return "", nil
}
