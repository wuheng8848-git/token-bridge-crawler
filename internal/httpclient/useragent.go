// Package httpclient 提供统一的 HTTP 客户端，封装反爬策略
package httpclient

import (
	"math/rand"
	"sync"
	"time"
)

// DefaultUserAgents 默认 User-Agent 列表
// 包含常见浏览器 UA，借鉴 Photon 的做法
var DefaultUserAgents = []string{
	// Chrome on Windows
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36",

	// Firefox on Windows
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0",

	// Chrome on macOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",

	// Safari on macOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",

	// Chrome on Linux
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",

	// Firefox on Linux
	"Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0",

	// Edge on Windows
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
}

// UserAgentPool User-Agent 池
type UserAgentPool struct {
	userAgents []string
	current    int
	mu         sync.Mutex
	rng        *rand.Rand
}

// NewUserAgentPool 创建 User-Agent 池
func NewUserAgentPool(userAgents []string) *UserAgentPool {
	if len(userAgents) == 0 {
		userAgents = DefaultUserAgents
	}

	return &UserAgentPool{
		userAgents: userAgents,
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Random 随机选择一个 User-Agent
// 借鉴 Photon 的随机选择策略
func (p *UserAgentPool) Random() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.userAgents[p.rng.Intn(len(p.userAgents))]
}

// Next 轮询选择下一个 User-Agent
func (p *UserAgentPool) Next() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	ua := p.userAgents[p.current]
	p.current = (p.current + 1) % len(p.userAgents)
	return ua
}

// All 返回所有 User-Agent
func (p *UserAgentPool) All() []string {
	return p.userAgents
}

// Add 添加 User-Agent
func (p *UserAgentPool) Add(ua string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.userAgents = append(p.userAgents, ua)
}
