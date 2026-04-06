// Package httpclient 提供统一的 HTTP 客户端，封装反爬策略
package httpclient

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Client 统一 HTTP 客户端
// 整合速率限制、重试、User-Agent 轮换等反爬策略
type Client struct {
	httpClient    *http.Client
	config        Config
	rateLimiter   *RateLimiter
	userAgentPool *UserAgentPool
	retryPolicy   RetryPolicy

	// 状态追踪
	totalRequests  int64
	failedRequests int64
	lastError      error
	lastErrorAt    *time.Time
}

// NewClient 创建新的 HTTP 客户端
func NewClient(config Config) *Client {
	// 应用默认值
	cfg := MergeConfig(DefaultConfig(config.Name), config)

	// 创建底层 HTTP 客户端
	httpClient := &http.Client{
		Timeout: cfg.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: false,
		},
	}

	return &Client{
		httpClient:    httpClient,
		config:        cfg,
		rateLimiter:   NewRateLimiter(cfg),
		userAgentPool: NewUserAgentPool(cfg.UserAgents),
		retryPolicy: RetryPolicy{
			MaxRetries:    cfg.MaxRetries,
			BaseDelay:     cfg.RetryBaseDelay,
			MaxDelay:      cfg.RetryMaxDelay,
			RetryOnStatus: []int{429, 500, 502, 503, 504},
			backoffFactor: 2.0,
		},
	}
}

// NewClientWithPreset 使用预设配置创建客户端
func NewClientWithPreset(presetName string) *Client {
	return NewClient(GetPresetConfig(presetName))
}

// Do 执行 HTTP 请求（带所有反爬策略）
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.DoWithContext(context.Background(), req)
}

// DoWithContext 带上下文的请求执行
func (c *Client) DoWithContext(ctx context.Context, req *http.Request) (*http.Response, error) {
	// 1. 速率限制等待
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	// 2. 设置随机 User-Agent
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.userAgentPool.Random())
	}

	// 3. 执行请求（带重试）
	return c.doWithRetry(ctx, req)
}

// doWithRetry 内部重试逻辑
func (c *Client) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response

	for attempt := 0; attempt <= c.retryPolicy.MaxRetries; attempt++ {
		// 检查上下文
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// 记录请求
		c.totalRequests++

		// 执行请求
		resp, err := c.httpClient.Do(req)

		// 成功
		if err == nil && resp.StatusCode < 400 {
			return resp, nil
		}

		// 记录错误
		lastErr = err
		if resp != nil {
			lastResp = resp
		}

		// 判断是否重试
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
			resp.Body.Close()
		}

		// 检测到 429 限流，记录警告
		if statusCode == 429 {
			log.Printf("[%s] Rate limited (429), attempt %d/%d", c.config.Name, attempt+1, c.retryPolicy.MaxRetries)
		}

		// 检查是否应该重试
		if !IsRetryable(statusCode, err) || attempt >= c.retryPolicy.MaxRetries {
			break
		}

		// 计算延迟
		delay := c.retryPolicy.CalculateDelay(attempt)
		log.Printf("[%s] Retrying in %v (attempt %d/%d, status %d)",
			c.config.Name, delay, attempt+1, c.retryPolicy.MaxRetries, statusCode)

		// 等待
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// 继续重试
		}
	}

	// 所有重试失败
	c.failedRequests++
	now := time.Now()
	c.lastError = lastErr
	c.lastErrorAt = &now

	if lastResp != nil {
		return lastResp, &RetryableError{
			Err:        lastErr,
			StatusCode: lastResp.StatusCode,
		}
	}
	return nil, &RetryableError{Err: lastErr}
}

// Get 执行 GET 请求
func (c *Client) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// GetWithContext 带上下文的 GET 请求
func (c *Client) GetWithContext(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post 执行 POST 请求
func (c *Client) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}

// PostWithContext 带上下文的 POST 请求
func (c *Client) PostWithContext(ctx context.Context, url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}

// SetTimeout 设置超时时间
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
	c.config.Timeout = timeout
}

// SetRateLimit 设置速率限制
func (c *Client) SetRateLimit(requestsPerSecond float64) {
	c.rateLimiter.SetLimit(requestsPerSecond)
	c.config.RequestsPerSecond = requestsPerSecond
}

// SetUserAgents 设置 User-Agent 列表
func (c *Client) SetUserAgents(userAgents []string) {
	c.userAgentPool = NewUserAgentPool(userAgents)
	c.config.UserAgents = userAgents
}

// Stats 客户端统计信息
type Stats struct {
	Name           string
	TotalRequests  int64
	FailedRequests int64
	SuccessRate    float64
	LastError      string
	LastErrorAt    *time.Time
	CurrentRate    float64
}

// GetStats 获取统计信息
func (c *Client) GetStats() Stats {
	stats := Stats{
		Name:           c.config.Name,
		TotalRequests:  c.totalRequests,
		FailedRequests: c.failedRequests,
		CurrentRate:    c.rateLimiter.Limit(),
		LastErrorAt:    c.lastErrorAt,
	}

	if c.totalRequests > 0 {
		stats.SuccessRate = float64(c.totalRequests-c.failedRequests) / float64(c.totalRequests) * 100
	}

	if c.lastError != nil {
		stats.LastError = c.lastError.Error()
	}

	return stats
}

// Config 返回配置
func (c *Client) Config() Config {
	return c.config
}

// RateLimiter 返回速率限制器
func (c *Client) RateLimiter() *RateLimiter {
	return c.rateLimiter
}
