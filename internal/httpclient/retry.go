// Package httpclient 提供统一的 HTTP 客户端，封装反爬策略
package httpclient

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"
)

// RetryableError 可重试的错误
type RetryableError struct {
	Err        error
	StatusCode int
}

func (e *RetryableError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("status %d: %v", e.StatusCode, e.Err)
	}
	return e.Err.Error()
}

// IsRetryable 判断是否应该重试
// 借鉴 Spiderfoot 的 429 检测 + Social-Analyzer 的重试策略
func IsRetryable(statusCode int, err error) bool {
	// 429 Too Many Requests - 必须重试
	if statusCode == http.StatusTooManyRequests {
		return true
	}

	// 5xx 服务器错误 - 可以重试
	if statusCode >= 500 && statusCode < 600 {
		return true
	}

	// 网络错误 - 可以重试
	if err != nil {
		return true
	}

	return false
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries    int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	RetryOnStatus []int // 触发重试的状态码
	backoffFactor float64
}

// DefaultRetryPolicy 默认重试策略
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:    3,
		BaseDelay:     1 * time.Second,
		MaxDelay:      30 * time.Second,
		RetryOnStatus: []int{429, 500, 502, 503, 504},
		backoffFactor: 2.0,
	}
}

// CalculateDelay 计算重试延迟（指数退避）
// delay = baseDelay * factor^attempt
// 借鉴 Social-Analyzer 的多轮重试思想
func (p RetryPolicy) CalculateDelay(attempt int) time.Duration {
	delay := float64(p.BaseDelay) * math.Pow(p.backoffFactor, float64(attempt))
	if delay > float64(p.MaxDelay) {
		delay = float64(p.MaxDelay)
	}
	return time.Duration(delay)
}

// ShouldRetry 判断是否应该重试
func (p RetryPolicy) ShouldRetry(statusCode int, attempt int) bool {
	if attempt >= p.MaxRetries {
		return false
	}

	for _, s := range p.RetryOnStatus {
		if statusCode == s {
			return true
		}
	}
	return false
}

// RetryableClient 可重试的客户端接口
type RetryableClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// DoWithRetry 带重试的请求执行
func DoWithRetry(ctx context.Context, client RetryableClient, req *http.Request, policy RetryPolicy) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		// 检查上下文是否取消
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// 执行请求
		resp, err := client.Do(req)

		// 成功
		if err == nil && resp.StatusCode < 400 {
			return resp, nil
		}

		// 记录最后一次错误
		lastErr = err
		if resp != nil {
			lastResp = resp
		}

		// 判断是否应该重试
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
			// 关闭响应体，避免资源泄漏
			resp.Body.Close()
		}

		if !policy.ShouldRetry(statusCode, attempt) {
			break
		}

		// 计算延迟并等待
		delay := policy.CalculateDelay(attempt)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// 继续重试
		}
	}

	// 所有重试都失败了
	if lastResp != nil {
		return lastResp, &RetryableError{
			Err:        lastErr,
			StatusCode: lastResp.StatusCode,
		}
	}
	return nil, &RetryableError{Err: lastErr}
}

// BackoffStrategy 退避策略类型
type BackoffStrategy int

const (
	// BackoffExponential 指数退避
	BackoffExponential BackoffStrategy = iota
	// BackoffLinear 线性退避
	BackoffLinear
	// BackoffConstant 固定延迟
	BackoffConstant
)

// CalculateBackoff 计算退避延迟
func CalculateBackoff(strategy BackoffStrategy, baseDelay time.Duration, attempt int) time.Duration {
	switch strategy {
	case BackoffExponential:
		return baseDelay * time.Duration(1<<uint(attempt))
	case BackoffLinear:
		return baseDelay * time.Duration(attempt+1)
	case BackoffConstant:
		return baseDelay
	default:
		return baseDelay
	}
}
