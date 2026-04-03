// Package userpain 提供用户痛点情报采集功能
package userpain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"token-bridge-crawler/internal/core"
)

// RedditCollector Reddit用户痛点采集器
type RedditCollector struct {
	*BaseUserPainCollector
	client *http.Client
}

// NewRedditCollector 创建Reddit采集器
func NewRedditCollector() *RedditCollector {
	return &RedditCollector{
		BaseUserPainCollector: NewBaseUserPainCollector(
			"reddit_pain",
			"reddit",
			6*time.Hour,
		),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CollectUserPains 从Reddit采集用户痛点
func (c *RedditCollector) CollectUserPains(ctx context.Context) ([]core.IntelItem, error) {
	var items []core.IntelItem

	// 关注的subreddit列表
	subreddits := []string{
		"OpenAI",
		"ClaudeAI",
		"LocalLLaMA",
		"MachineLearning",
		"webdev",
		"programming",
	}

	// 搜索关键词
	keywords := []string{
		"API cost",
		"pricing",
		"expensive",
		"alternative",
		"rate limit",
		"billing",
	}

	for _, subreddit := range subreddits {
		for _, keyword := range keywords {
			searchItems, err := c.searchPosts(ctx, subreddit, keyword)
			if err != nil {
				continue
			}
			items = append(items, searchItems...)
			time.Sleep(500 * time.Millisecond)
		}
	}

	return items, nil
}

// searchPosts 搜索Reddit帖子
func (c *RedditCollector) searchPosts(ctx context.Context, subreddit, keyword string) ([]core.IntelItem, error) {
	// 使用Reddit JSON API (无需认证，有限制)
	url := fmt.Sprintf("https://www.reddit.com/r/%s/search.json?q=%s&sort=new&restrict_sr=1&t=month",
		subreddit,
		strings.ReplaceAll(keyword, " ", "+"),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 设置User-Agent
	req.Header.Set("User-Agent", "TokenBridge-Intelligence/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Reddit API returned status %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			Children []struct {
				Data struct {
					ID          string  `json:"id"`
					Title       string  `json:"title"`
					SelfText    string  `json:"selftext"`
					URL         string  `json:"url"`
					Author      string  `json:"author"`
					Score       int     `json:"score"`
					NumComments int     `json:"num_comments"`
					CreatedUTC  float64 `json:"created_utc"`
					Subreddit   string  `json:"subreddit"`
					Permalink   string  `json:"permalink"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var items []core.IntelItem
	for _, child := range result.Data.Children {
		post := child.Data

		// 只关注有一定热度的帖子
		if post.Score < 10 && post.NumComments < 5 {
			continue
		}

		item := core.NewIntelItem(core.IntelTypeUserPain, "reddit")
		item.Title = post.Title
		item.Content = post.SelfText
		item.URL = "https://reddit.com" + post.Permalink
		item.SourceID = post.ID

		// 分析痛点类型
		painType := c.detectPainType(post.Title + " " + post.SelfText)

		// 分析情感倾向
		sentiment := c.analyzeSentiment(post.Title + " " + post.SelfText)

		// 设置元数据
		item.Metadata = core.Metadata{
			"platform":       "reddit",
			"subreddit":      post.Subreddit,
			"points":         post.Score,
			"comments_count": post.NumComments,
			"author":         post.Author,
			"pain_type":      painType,
			"sentiment":      sentiment,
			"query_matched":  keyword,
		}

		// 解析发布时间
		createdAt := time.Unix(int64(post.CreatedUTC), 0)
		item.PublishedAt = &createdAt

		items = append(items, item)
	}

	return items, nil
}

// detectPainType 检测痛点类型
func (c *RedditCollector) detectPainType(text string) string {
	textLower := strings.ToLower(text)

	if strings.Contains(textLower, "cost") || strings.Contains(textLower, "expensive") || strings.Contains(textLower, "price") || strings.Contains(textLower, "bill") || strings.Contains(textLower, "pricing") {
		return "cost"
	}
	if strings.Contains(textLower, "complex") || strings.Contains(textLower, "difficult") || strings.Contains(textLower, "confusing") || strings.Contains(textLower, "hard to") {
		return "complexity"
	}
	if strings.Contains(textLower, "compliance") || strings.Contains(textLower, "regulation") || strings.Contains(textLower, "gdpr") || strings.Contains(textLower, "privacy") {
		return "compliance"
	}
	if strings.Contains(textLower, "payment") || strings.Contains(textLower, "billing") || strings.Contains(textLower, "invoice") || strings.Contains(textLower, "charge") {
		return "payment"
	}
	if strings.Contains(textLower, "rate limit") || strings.Contains(textLower, "throttle") || strings.Contains(textLower, "quota") || strings.Contains(textLower, "limit") {
		return "rate_limit"
	}
	if strings.Contains(textLower, "alternative") || strings.Contains(textLower, "switch") || strings.Contains(textLower, "migrate") || strings.Contains(textLower, "move to") {
		return "switching"
	}

	return "general"
}

// analyzeSentiment 分析情感倾向
func (c *RedditCollector) analyzeSentiment(text string) string {
	textLower := strings.ToLower(text)

	negativeWords := []string{
		"frustrated", "annoying", "terrible", "awful", "hate",
		"disappointed", "problem", "issue", "bug", "broken",
		"expensive", "too much", "ridiculous", "outrageous",
	}

	positiveWords := []string{
		"great", "awesome", "love", "perfect", "excellent",
		"amazing", "helpful", "easy", "simple", "cheap",
	}

	negativeCount := 0
	positiveCount := 0

	for _, word := range negativeWords {
		if strings.Contains(textLower, word) {
			negativeCount++
		}
	}

	for _, word := range positiveWords {
		if strings.Contains(textLower, word) {
			positiveCount++
		}
	}

	if negativeCount > positiveCount {
		return "negative"
	} else if positiveCount > negativeCount {
		return "positive"
	}
	return "neutral"
}
