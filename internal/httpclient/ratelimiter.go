// Package httpclient 提供统一的 HTTP 客户端，封装反爬策略
package httpclient

import (
	"context"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter 速率限制器
// 使用令牌桶算法，借鉴 Social-Analyzer 的并发限制思想
type RateLimiter struct {
	limiter *rate.Limiter
	config  Config
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(config Config) *RateLimiter {
	// rate.Every 将间隔转换为频率
	// 例如: 1秒1个请求 = rate.Every(1秒) = 1 Hz
	interval := time.Duration(float64(time.Second) / config.RequestsPerSecond)

	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Every(interval), config.BurstSize),
		config:  config,
	}
}

// Wait 等待直到可以发送请求
// 阻塞直到令牌桶中有可用令牌
func (rl *RateLimiter) Wait(ctx context.Context) error {
	return rl.limiter.Wait(ctx)
}

// WaitN 等待 N 个令牌
func (rl *RateLimiter) WaitN(ctx context.Context, n int) error {
	return rl.limiter.WaitN(ctx, n)
}

// Allow 检查是否可以立即发送请求（不阻塞）
func (rl *RateLimiter) Allow() bool {
	return rl.limiter.Allow()
}

// AllowN 检查是否可以立即发送 N 个请求
func (rl *RateLimiter) AllowN(n int) bool {
	return rl.limiter.AllowN(time.Now(), n)
}

// Reserve 预留令牌（返回需要等待的时间）
func (rl *RateLimiter) Reserve() time.Duration {
	r := rl.limiter.Reserve()
	if !r.OK() {
		return -1
	}
	return r.Delay()
}

// SetLimit 动态调整速率
func (rl *RateLimiter) SetLimit(requestsPerSecond float64) {
	interval := time.Duration(float64(time.Second) / requestsPerSecond)
	rl.limiter.SetLimit(rate.Every(interval))
}

// SetBurst 动态调整突发容量
func (rl *RateLimiter) SetBurst(burst int) {
	rl.limiter.SetBurst(burst)
}

// Limit 返回当前速率（每秒请求数）
func (rl *RateLimiter) Limit() float64 {
	return float64(rl.limiter.Limit())
}

// Burst 返回当前突发容量
func (rl *RateLimiter) Burst() int {
	return rl.limiter.Burst()
}

// Tokens 返回当前可用令牌数
func (rl *RateLimiter) Tokens() float64 {
	return rl.limiter.Tokens()
}
