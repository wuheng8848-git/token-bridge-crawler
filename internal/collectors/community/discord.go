// Package community 提供社区情报采集功能
package community

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"token-bridge-crawler/internal/core"
)

// DiscordCollector Discord采集器（真实API版本）
type DiscordCollector struct {
	name       string
	source     string
	interval   time.Duration
	session    *discordgo.Session
	channelIDs []string
	keywords   []string
}

// DiscordConfig Discord采集器配置
type DiscordConfig struct {
	BotToken   string   // Discord Bot Token
	ChannelIDs []string // 要监控的频道ID列表
	Keywords   []string // 关键词过滤（可选）
}

// NewDiscordCollector 创建Discord采集器
func NewDiscordCollector(config DiscordConfig) (*DiscordCollector, error) {
	if config.BotToken == "" {
		return nil, fmt.Errorf("Discord Bot Token is required")
	}

	// 创建 Discord Session
	session, err := discordgo.New("Bot " + config.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	// 默认关键词（AI相关）
	if len(config.Keywords) == 0 {
		config.Keywords = []string{
			"API", "pricing", "cost", "OpenAI", "Anthropic", "Google",
			"LLM", "GPT", "Claude", "Gemini", "model", "token",
			"rate limit", "error", "integration", "SDK",
		}
	}

	return &DiscordCollector{
		name:       "discord_collector",
		source:     "discord",
		interval:   6 * time.Hour,
		session:    session,
		channelIDs: config.ChannelIDs,
		keywords:   config.Keywords,
	}, nil
}

// Name 返回采集器名称
func (c *DiscordCollector) Name() string {
	return c.name
}

// Source 返回数据源
func (c *DiscordCollector) Source() string {
	return c.source
}

// Interval 返回采集间隔
func (c *DiscordCollector) Interval() time.Duration {
	return c.interval
}

// IntelType 返回情报类型
func (c *DiscordCollector) IntelType() core.IntelType {
	return core.IntelTypeCommunity
}

// Fetch 采集Discord数据
func (c *DiscordCollector) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	var items []core.IntelItem

	// 遍历所有监控的频道
	for _, channelID := range c.channelIDs {
		channelItems, err := c.fetchChannelMessages(ctx, channelID)
		if err != nil {
			log.Printf("[Discord] 采集频道 %s 失败: %v", channelID, err)
			continue
		}
		items = append(items, channelItems...)
	}

	log.Printf("[Discord] 共采集 %d 条消息", len(items))
	return items, nil
}

// RateLimit 返回请求间隔
func (c *DiscordCollector) RateLimit() time.Duration {
	return 2 * time.Second // Discord API 限制
}

// fetchChannelMessages 采集单个频道的消息
func (c *DiscordCollector) fetchChannelMessages(ctx context.Context, channelID string) ([]core.IntelItem, error) {
	var items []core.IntelItem

	// 获取频道信息
	channel, err := c.session.Channel(channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel info: %w", err)
	}

	// 获取最近100条消息
	messages, err := c.session.ChannelMessages(channelID, 100, "", "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	for _, msg := range messages {
		// 过滤关键词
		if !c.matchesKeywords(msg.Content) {
			continue
		}

		// 转换为情报项
		item := c.convertMessageToItem(msg, channel)
		items = append(items, item)
	}

	return items, nil
}

// matchesKeywords 检查消息是否匹配关键词
func (c *DiscordCollector) matchesKeywords(content string) bool {
	if len(c.keywords) == 0 {
		return true // 无关键词过滤，全部采集
	}

	contentLower := strings.ToLower(content)
	for _, keyword := range c.keywords {
		if strings.Contains(contentLower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// convertMessageToItem 将 Discord 消息转换为情报项
func (c *DiscordCollector) convertMessageToItem(msg *discordgo.Message, channel *discordgo.Channel) core.IntelItem {
	item := core.NewIntelItem(core.IntelTypeCommunity, "discord")

	// 基本信息
	item.SourceID = msg.ID
	item.URL = fmt.Sprintf("https://discord.com/channels/%s/%s/%s",
		msg.GuildID, channel.ID, msg.ID)

	// 标题：使用作者名 + 消息摘要
	title := fmt.Sprintf("Discord/%s: %s", channel.Name, truncate(msg.Content, 50))
	item.Title = title

	// 内容：完整消息
	item.Content = msg.Content

	// 发布时间
	item.PublishedAt = &msg.Timestamp

	// 元数据
	item.Metadata = core.Metadata{
		"platform":        "discord",
		"channel_id":      channel.ID,
		"channel_name":    channel.Name,
		"guild_id":        msg.GuildID,
		"author_id":       msg.Author.ID,
		"author_name":     msg.Author.Username,
		"author_avatar":   msg.Author.AvatarURL(""),
		"message_id":      msg.ID,
		"content_type":    "message",
		"has_attachments": len(msg.Attachments) > 0,
		"has_embeds":      len(msg.Embeds) > 0,
		"reactions":       c.extractReactions(msg.Reactions),
		"reply_count":     0, // 需要额外查询
	}

	// 如果有附件，记录附件信息
	if len(msg.Attachments) > 0 {
		attachments := make([]map[string]string, len(msg.Attachments))
		for i, att := range msg.Attachments {
			attachments[i] = map[string]string{
				"filename": att.Filename,
				"url":      att.URL,
				"size":     fmt.Sprintf("%d", att.Size),
			}
		}
		item.Metadata["attachments"] = attachments
	}

	// 如果有嵌入内容（链接预览等）
	if len(msg.Embeds) > 0 {
		embeds := make([]map[string]string, len(msg.Embeds))
		for i, embed := range msg.Embeds {
			embeds[i] = map[string]string{
				"title":       embed.Title,
				"description": embed.Description,
				"url":         embed.URL,
			}
		}
		item.Metadata["embeds"] = embeds
	}

	return item
}

// extractReactions 提取反应信息
func (c *DiscordCollector) extractReactions(reactions []*discordgo.MessageReactions) []map[string]interface{} {
	if len(reactions) == 0 {
		return nil
	}

	result := make([]map[string]interface{}, len(reactions))
	for i, r := range reactions {
		result[i] = map[string]interface{}{
			"emoji": r.Emoji.Name,
			"count": r.Count,
			"me":    r.Me,
		}
	}
	return result
}

// Close 关闭 Discord 连接
func (c *DiscordCollector) Close() {
	if c.session != nil {
		c.session.Close()
	}
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// --- 模拟数据版本（备用）---

// NewDiscordCollectorMock 创建模拟版本的Discord采集器（用于测试）
func NewDiscordCollectorMock() *DiscordCollectorMock {
	return &DiscordCollectorMock{
		name:     "discord_collector_mock",
		interval: 24 * time.Hour,
		source:   "discord",
	}
}

// DiscordCollectorMock 模拟版本的Discord采集器
type DiscordCollectorMock struct {
	name     string
	source   string
	interval time.Duration
}

// Name 返回采集器名称
func (c *DiscordCollectorMock) Name() string {
	return c.name
}

// Source 返回数据源
func (c *DiscordCollectorMock) Source() string {
	return c.source
}

// Interval 返回采集间隔
func (c *DiscordCollectorMock) Interval() time.Duration {
	return c.interval
}

// IntelType 返回情报类型
func (c *DiscordCollectorMock) IntelType() core.IntelType {
	return core.IntelTypeCommunity
}

// Fetch 采集模拟数据
func (c *DiscordCollectorMock) Fetch(ctx context.Context) ([]core.IntelItem, error) {
	// 返回模拟数据（原版本的逻辑）
	return []core.IntelItem{}, nil
}

// RateLimit 返回请求间隔
func (c *DiscordCollectorMock) RateLimit() time.Duration {
	return 5 * time.Second
}
