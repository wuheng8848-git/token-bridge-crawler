// Package profiler 用户画像与客户分级
package profiler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"token-bridge-crawler/internal/core"
)

// Profiler 用户画像获取器
type Profiler struct {
	hnClient *HNClient
	ghClient *GitHubClient
	config   ProfilerConfig
}

// ProfilerConfig 配置
type ProfilerConfig struct {
	// GitHub API Token（可选，用于提高请求限制）
	GitHubToken string `yaml:"github_token" json:"github_token"`

	// HTTP 超时
	HTTPTimeout time.Duration `yaml:"http_timeout" json:"http_timeout"`

	// 是否启用 HackerNews 画像
	EnableHN bool `yaml:"enable_hn" json:"enable_hn"`

	// 是否启用 GitHub 画像
	EnableGitHub bool `yaml:"enable_github" json:"enable_github"`
}

// DefaultProfilerConfig 默认配置
func DefaultProfilerConfig() ProfilerConfig {
	return ProfilerConfig{
		HTTPTimeout:  10 * time.Second,
		EnableHN:     true,
		EnableGitHub: true,
	}
}

// NewProfiler 创建用户画像获取器
func NewProfiler(config ProfilerConfig) *Profiler {
	httpClient := &http.Client{
		Timeout: config.HTTPTimeout,
	}

	return &Profiler{
		hnClient: NewHNClient(httpClient),
		ghClient: NewGitHubClient(httpClient, config.GitHubToken),
		config:   config,
	}
}

// Profile 获取用户画像
func (p *Profiler) Profile(ctx context.Context, item core.IntelItem) ProfileResult {
	result := ProfileResult{
		CustomerTier: "C",
		UserType:     "unknown",
		SocialLinks:  make(map[string]string),
	}

	// 从元数据获取作者信息
	author, ok := item.Metadata["author"].(string)
	if !ok || author == "" {
		return result
	}

	// 根据来源获取画像
	switch item.Source {
	case "hackernews":
		if p.config.EnableHN {
			hnProfile := p.hnClient.GetUser(ctx, author)
			result = p.buildProfileFromHN(hnProfile)
		}
	case "reddit":
		// Reddit 用户画像（暂使用默认）
		result.UserType = "reddit_user"
	}

	// 尝试从 GitHub 获取更多信息（可选）
	if p.config.EnableGitHub && result.SocialLinks["github"] != "" {
		ghProfile := p.ghClient.GetUser(ctx, result.SocialLinks["github"])
		p.enrichFromGitHub(&result, ghProfile)
	}

	return result
}

// buildProfileFromHN 从 HackerNews 信息构建画像
func (p *Profiler) buildProfileFromHN(hnProfile *HNUser) ProfileResult {
	result := ProfileResult{
		UserType:    "hn_user",
		Karma:       hnProfile.Karma,
		SocialLinks: make(map[string]string),
	}

	// 从 about 中提取社交链接
	if hnProfile.About != "" {
		result.SocialLinks = extractSocialLinks(hnProfile.About)
	}

	// 计算客户等级
	result.CustomerTier = p.calculateTier(hnProfile.Karma, result.SocialLinks)

	return result
}

// enrichFromGitHub 用 GitHub 信息丰富画像
func (p *Profiler) enrichFromGitHub(result *ProfileResult, ghProfile *GitHubUser) {
	if ghProfile == nil {
		return
	}

	result.SocialLinks["github"] = ghProfile.Login

	// 从 bio 中提取公司信息
	if ghProfile.Company != "" {
		result.Company = ghProfile.Company
	}

	// 检查是否为知名公司开发者
	if isKnownCompany(ghProfile.Company) {
		result.UserType = "known_company_dev"
		// 提升客户等级
		if result.CustomerTier == "B" || result.CustomerTier == "C" {
			result.CustomerTier = "A"
		}
	}
}

// calculateTier 计算客户等级
func (p *Profiler) calculateTier(karma int, socialLinks map[string]string) string {
	// S 级：高 karma + GitHub 链接 + 可能是知名公司
	if karma >= 10000 && len(socialLinks) >= 2 {
		return "S"
	}

	// A 级：较高 karma + 有 GitHub 链接
	if karma >= 5000 && socialLinks["github"] != "" {
		return "A"
	}

	// B 级：中等 karma
	if karma >= 1000 {
		return "B"
	}

	// C 级：低 karma 或新用户
	return "C"
}

// ProfileResult 用户画像结果
type ProfileResult struct {
	CustomerTier string            `json:"customer_tier"` // S/A/B/C
	UserType     string            `json:"user_type"`
	Karma        int               `json:"karma"`
	Company      string            `json:"company"`
	SocialLinks  map[string]string `json:"social_links"`
}

// extractSocialLinks 从文本中提取社交链接
func extractSocialLinks(text string) map[string]string {
	links := make(map[string]string)
	text = strings.ToLower(text)

	// GitHub
	if strings.Contains(text, "github.com/") {
		parts := strings.Split(text, "github.com/")
		if len(parts) > 1 {
			username := strings.Fields(parts[1])[0]
			username = strings.TrimSuffix(username, ")")
			username = strings.TrimSuffix(username, "]")
			links["github"] = username
		}
	}

	// Twitter
	if strings.Contains(text, "twitter.com/") || strings.Contains(text, "@") {
		if strings.Contains(text, "twitter.com/") {
			parts := strings.Split(text, "twitter.com/")
			if len(parts) > 1 {
				username := strings.Fields(parts[1])[0]
				username = strings.TrimPrefix(username, "@")
				username = strings.TrimSuffix(username, ")")
				links["twitter"] = username
			}
		}
	}

	// LinkedIn
	if strings.Contains(text, "linkedin.com/in/") {
		parts := strings.Split(text, "linkedin.com/in/")
		if len(parts) > 1 {
			username := strings.Fields(parts[1])[0]
			username = strings.TrimSuffix(username, ")")
			links["linkedin"] = username
		}
	}

	return links
}

// knownCompanies 知名公司列表
var knownCompanies = []string{
	"google", "meta", "facebook", "amazon", "apple", "microsoft",
	"netflix", "twitter", "uber", "lyft", "airbnb", "stripe",
	"openai", "anthropic", "deepmind", "nvidia", "tesla",
	"shopify", "notion", "figma", "vercel", "cloudflare",
}

// isKnownCompany 检查是否为知名公司
func isKnownCompany(company string) bool {
	if company == "" {
		return false
	}
	company = strings.ToLower(company)
	for _, known := range knownCompanies {
		if strings.Contains(company, known) {
			return true
		}
	}
	return false
}

// HNUser HackerNews 用户信息
type HNUser struct {
	ID      string `json:"id"`
	Karma   int    `json:"karma"`
	About   string `json:"about"`
	Created int64  `json:"created"`
}

// GitHubUser GitHub 用户信息
type GitHubUser struct {
	Login    string `json:"login"`
	Name     string `json:"name"`
	Company  string `json:"company"`
	Bio      string `json:"bio"`
	Location string `json:"location"`
}

// HNClient HackerNews API 客户端
type HNClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewHNClient 创建 HN 客户端
func NewHNClient(httpClient *http.Client) *HNClient {
	return &HNClient{
		httpClient: httpClient,
		baseURL:    "https://hacker-news.firebaseio.com/v0",
	}
}

// GetUser 获取用户信息
func (c *HNClient) GetUser(ctx context.Context, username string) *HNUser {
	url := c.baseURL + "/user/" + username + ".json"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("[HNClient] Failed to create request: %v", err)
		return nil
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[HNClient] Failed to get user %s: %v", username, err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[HNClient] User %s not found: %d", username, resp.StatusCode)
		return nil
	}

	var user HNUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		log.Printf("[HNClient] Failed to decode user: %v", err)
		return nil
	}

	return &user
}

// GitHubClient GitHub API 客户端
type GitHubClient struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// NewGitHubClient 创建 GitHub 客户端
func NewGitHubClient(httpClient *http.Client, token string) *GitHubClient {
	return &GitHubClient{
		httpClient: httpClient,
		baseURL:    "https://api.github.com",
		token:      token,
	}
}

// GetUser 获取用户信息
func (c *GitHubClient) GetUser(ctx context.Context, username string) *GitHubUser {
	if username == "" {
		return nil
	}

	url := c.baseURL + "/users/" + username

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("[GitHubClient] Failed to create request: %v", err)
		return nil
	}

	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[GitHubClient] Failed to get user %s: %v", username, err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[GitHubClient] User %s not found: %d", username, resp.StatusCode)
		return nil
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		log.Printf("[GitHubClient] Failed to decode user: %v", err)
		return nil
	}

	return &user
}
