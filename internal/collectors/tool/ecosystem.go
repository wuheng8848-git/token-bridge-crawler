// Package tool 提供工具生态情报采集功能
package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"token-bridge-crawler/internal/core"
)

// EcosystemCollector 工具生态采集器
type EcosystemCollector struct {
	*BaseToolEcosystemCollector
	client *http.Client
}

// ToolInfo 工具信息
type ToolInfo struct {
	Name            string
	Category        string
	GitHubRepo      string
	Website         string
	OpennessLevel   string
	IntegrationAPIs []string
}

// NewEcosystemCollector 创建工具生态采集器
func NewEcosystemCollector() *EcosystemCollector {
	return &EcosystemCollector{
		BaseToolEcosystemCollector: NewBaseToolEcosystemCollector(
			"tool_ecosystem",
			"github",
			24*time.Hour,
		),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CollectToolEcosystem 采集工具生态数据
func (c *EcosystemCollector) CollectToolEcosystem(ctx context.Context) ([]core.IntelItem, error) {
	var items []core.IntelItem

	// 定义要追踪的工具列表
	tools := []ToolInfo{
		{
			Name:            "Cursor",
			Category:        "code_editor",
			GitHubRepo:      "getcursor/cursor",
			Website:         "https://cursor.sh",
			OpennessLevel:   "semi-open",
			IntegrationAPIs: []string{"OpenAI", "Claude", "Custom"},
		},
		{
			Name:            "Claude Code",
			Category:        "cli_tool",
			GitHubRepo:      "anthropics/claude-code",
			Website:         "https://claude.ai/code",
			OpennessLevel:   "closed",
			IntegrationAPIs: []string{"Claude"},
		},
		{
			Name:            "Aider",
			Category:        "cli_tool",
			GitHubRepo:      "paul-gauthier/aider",
			Website:         "https://aider.chat",
			OpennessLevel:   "open",
			IntegrationAPIs: []string{"OpenAI", "Claude", "Gemini", "Local"},
		},
		{
			Name:            "Continue",
			Category:        "vscode_extension",
			GitHubRepo:      "continuedev/continue",
			Website:         "https://continue.dev",
			OpennessLevel:   "open",
			IntegrationAPIs: []string{"OpenAI", "Claude", "Gemini", "Ollama", "Custom"},
		},
		{
			Name:            "Cline",
			Category:        "vscode_extension",
			GitHubRepo:      "saoudrizwan/claude-dev",
			Website:         "https://github.com/saoudrizwan/claude-dev",
			OpennessLevel:   "open",
			IntegrationAPIs: []string{"Claude", "OpenAI"},
		},
		{
			Name:            "Supermaven",
			Category:        "code_editor",
			GitHubRepo:      "",
			Website:         "https://supermaven.com",
			OpennessLevel:   "closed",
			IntegrationAPIs: []string{"Proprietary"},
		},
		{
			Name:            "Codeium",
			Category:        "code_editor",
			GitHubRepo:      "",
			Website:         "https://codeium.com",
			OpennessLevel:   "semi-open",
			IntegrationAPIs: []string{"Proprietary", "Self-hosted"},
		},
		{
			Name:            "Tabnine",
			Category:        "code_editor",
			GitHubRepo:      "",
			Website:         "https://tabnine.com",
			OpennessLevel:   "closed",
			IntegrationAPIs: []string{"Proprietary"},
		},
	}

	for _, tool := range tools {
		if tool.GitHubRepo != "" {
			item, err := c.fetchGitHubStats(ctx, tool)
			if err == nil && item != nil {
				items = append(items, *item)
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	return items, nil
}

// fetchGitHubStats 获取GitHub统计数据
func (c *EcosystemCollector) fetchGitHubStats(ctx context.Context, tool ToolInfo) (*core.IntelItem, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s", tool.GitHubRepo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 如果有GitHub Token，可以添加认证
	// req.Header.Set("Authorization", "token YOUR_GITHUB_TOKEN")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var repoData struct {
		Stars       int       `json:"stargazers_count"`
		Forks       int       `json:"forks_count"`
		OpenIssues  int       `json:"open_issues_count"`
		UpdatedAt   time.Time `json:"updated_at"`
		CreatedAt   time.Time `json:"created_at"`
		Description string    `json:"description"`
		Language    string    `json:"language"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&repoData); err != nil {
		return nil, err
	}

	// 计算增长率（简化计算：总星标/存在天数）
	daysSinceCreated := time.Since(repoData.CreatedAt).Hours() / 24
	growthRate := 0.0
	if daysSinceCreated > 0 {
		growthRate = float64(repoData.Stars) / daysSinceCreated
	}

	item := core.NewIntelItem(core.IntelTypeToolEcosystem, "github")
	item.Title = fmt.Sprintf("%s - AI Coding Tool Stats", tool.Name)
	item.Content = fmt.Sprintf("GitHub Stars: %d, Forks: %d, Open Issues: %d\nDescription: %s",
		repoData.Stars, repoData.Forks, repoData.OpenIssues, repoData.Description)
	item.URL = fmt.Sprintf("https://github.com/%s", tool.GitHubRepo)
	item.SourceID = tool.GitHubRepo

	// 设置元数据
	item.Metadata = core.Metadata{
		"tool_name":         tool.Name,
		"category":          tool.Category,
		"stars":             repoData.Stars,
		"forks":             repoData.Forks,
		"open_issues":       repoData.OpenIssues,
		"language":          repoData.Language,
		"growth_rate":       growthRate,
		"user_base":         repoData.Stars, // 用stars作为用户基数估算
		"integration_apis":  tool.IntegrationAPIs,
		"openness_level":    tool.OpennessLevel,
		"website":           tool.Website,
	}

	item.PublishedAt = &repoData.UpdatedAt

	return &item, nil
}

// GetToolsByCategory 按类别获取工具
func (c *EcosystemCollector) GetToolsByCategory(category string) []ToolInfo {
	allTools := []ToolInfo{
		{Name: "Cursor", Category: "code_editor"},
		{Name: "Claude Code", Category: "cli_tool"},
		{Name: "Aider", Category: "cli_tool"},
		{Name: "Continue", Category: "vscode_extension"},
		{Name: "Cline", Category: "vscode_extension"},
		{Name: "Supermaven", Category: "code_editor"},
		{Name: "Codeium", Category: "code_editor"},
		{Name: "Tabnine", Category: "code_editor"},
	}

	var filtered []ToolInfo
	for _, tool := range allTools {
		if tool.Category == category {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

// GetOpenTools 获取开放程度高的工具
func (c *EcosystemCollector) GetOpenTools() []ToolInfo {
	return []ToolInfo{
		{Name: "Aider", OpennessLevel: "open"},
		{Name: "Continue", OpennessLevel: "open"},
		{Name: "Cline", OpennessLevel: "open"},
	}
}
