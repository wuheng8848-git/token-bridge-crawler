// Package httpclient 提供统一的 HTTP 客户端，封装反爬策略
package httpclient

import (
	"time"
)

// Config HTTP 客户端配置
type Config struct {
	// Name 采集器名称，用于日志
	Name string

	// RequestsPerSecond 每秒请求数限制
	// 例如: 1.0 = 每秒1个请求, 0.5 = 每2秒1个请求
	RequestsPerSecond float64

	// BurstSize 突发请求数（令牌桶容量）
	// 允许短时间内发送的额外请求数
	BurstSize int

	// MaxRetries 最大重试次数
	MaxRetries int

	// RetryBaseDelay 重试基础延迟
	// 实际延迟 = BaseDelay * 2^attempt
	RetryBaseDelay time.Duration

	// RetryMaxDelay 重试最大延迟
	RetryMaxDelay time.Duration

	// Timeout 请求超时时间
	Timeout time.Duration

	// UserAgents User-Agent 列表
	// 为空则使用默认列表
	UserAgents []string
}

// DefaultConfig 返回默认配置
func DefaultConfig(name string) Config {
	return Config{
		Name:              name,
		RequestsPerSecond: 1.0,
		BurstSize:         3,
		MaxRetries:        3,
		RetryBaseDelay:    1 * time.Second,
		RetryMaxDelay:     30 * time.Second,
		Timeout:           30 * time.Second,
		UserAgents:        DefaultUserAgents,
	}
}

// PresetConfigs 预设配置（针对不同采集源）
var PresetConfigs = map[string]Config{
	// HackerNews: 相对宽松，API 无认证限制
	"hackernews": {
		Name:              "hackernews",
		RequestsPerSecond: 1.0,
		BurstSize:         5,
		MaxRetries:        3,
		RetryBaseDelay:    1 * time.Second,
		RetryMaxDelay:     30 * time.Second,
		Timeout:           30 * time.Second,
	},

	// Reddit: 需要 OAuth，限制较严
	"reddit": {
		Name:              "reddit",
		RequestsPerSecond: 0.5,
		BurstSize:         2,
		MaxRetries:        3,
		RetryBaseDelay:    2 * time.Second,
		RetryMaxDelay:     60 * time.Second,
		Timeout:           30 * time.Second,
	},

	// GitHub: API 限制 5000次/小时（认证），60次/小时（未认证）
	"github": {
		Name:              "github",
		RequestsPerSecond: 0.8,
		BurstSize:         3,
		MaxRetries:        3,
		RetryBaseDelay:    1 * time.Second,
		RetryMaxDelay:     30 * time.Second,
		Timeout:           30 * time.Second,
	},

	// StackExchange: 300次/天（未认证），10000次/天（认证）
	"stackexchange": {
		Name:              "stackexchange",
		RequestsPerSecond: 0.3,
		BurstSize:         2,
		MaxRetries:        3,
		RetryBaseDelay:    2 * time.Second,
		RetryMaxDelay:     60 * time.Second,
		Timeout:           30 * time.Second,
	},

	// Tavily: API 搜索，按配额限制
	"tavily": {
		Name:              "tavily",
		RequestsPerSecond: 0.5,
		BurstSize:         2,
		MaxRetries:        3,
		RetryBaseDelay:    2 * time.Second,
		RetryMaxDelay:     30 * time.Second,
		Timeout:           60 * time.Second,
	},

	// 通用网页爬取
	"web": {
		Name:              "web",
		RequestsPerSecond: 0.5,
		BurstSize:         2,
		MaxRetries:        2,
		RetryBaseDelay:    2 * time.Second,
		RetryMaxDelay:     30 * time.Second,
		Timeout:           30 * time.Second,
	},
}

// GetPresetConfig 获取预设配置，不存在则返回默认配置
func GetPresetConfig(name string) Config {
	if cfg, ok := PresetConfigs[name]; ok {
		return cfg
	}
	return DefaultConfig(name)
}

// MergeConfig 合并配置（用户配置覆盖默认值）
func MergeConfig(base Config, override Config) Config {
	result := base

	if override.Name != "" {
		result.Name = override.Name
	}
	if override.RequestsPerSecond > 0 {
		result.RequestsPerSecond = override.RequestsPerSecond
	}
	if override.BurstSize > 0 {
		result.BurstSize = override.BurstSize
	}
	if override.MaxRetries > 0 {
		result.MaxRetries = override.MaxRetries
	}
	if override.RetryBaseDelay > 0 {
		result.RetryBaseDelay = override.RetryBaseDelay
	}
	if override.RetryMaxDelay > 0 {
		result.RetryMaxDelay = override.RetryMaxDelay
	}
	if override.Timeout > 0 {
		result.Timeout = override.Timeout
	}
	if len(override.UserAgents) > 0 {
		result.UserAgents = override.UserAgents
	}

	return result
}
