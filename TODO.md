# Token Bridge Crawler 待办事项

## 数据源采集器待完善

### 用户痛点采集（需要 API Key）

| 数据源 | 状态 | 所需配置 | 备注 |
|--------|------|----------|------|
| Discord | ❌ 放弃 | `DISCORD_BOT_TOKEN` | 只能采集自己频道，无法进入他人频道 |
| Reddit | ⚠️ 待定 | OAuth 应用凭据 | 可能需要 Reddit App 注册 |
| HackerNews | ⚠️ 调试中 | 无需 Key | 目前无数据返回，需进一步调试 |

### 待调研的无 Key 数据源（高优先级）

#### 技术社区（可采集用户痛点讨论）

| 数据源 | API 情况 | 内容类型 | 实现难度 |
|--------|----------|----------|----------|
| **OpenAI 社区论坛** | 可爬取 | API 定价抱怨、成本问题 | ⭐ 简单 |
| **StackExchange/Stack Overflow** | 免费公开 API | API 使用问题、定价讨论 | ⭐ 简单 |
| **Lobste.rs** | 无官方 API，可爬取 | 技术讨论、AI 工具反馈 | ⭐⭐ 中等 |
| **Tildes.net** | 无官方 API，可爬取 | 技术社区讨论 | ⭐⭐ 中等 |
| **Dev.to** | 免费公开 API | 技术文章、评论区 | ⭐ 简单 |
| **Indie Hackers** | 可爬取 | 创业者讨论、AI 成本抱怨 | ⭐⭐ 中等 |
| **Hashnode** | 免费 GraphQL API | 技术博客文章 | ⭐ 简单 |

#### RSS 源（无需认证）

- AI 相关技术博客 RSS 订阅
- 各厂商官方博客（OpenAI、Anthropic、Google AI）
- 技术新闻聚合 RSS

#### GitHub（无需认证的公开数据）

- GitHub Issues（各 AI SDK 仓库的问题讨论）
- GitHub Discussions（用户反馈）
- GitHub Trending（热门项目动向）

## 功能待完善

- [ ] AI 日报功能需要配置 `CRAWLER_AI_API_KEY`
- [ ] 邮件通知功能需要配置 SMTP 凭据
- [ ] 推送主系统需要启动主项目服务或配置远程地址

## 已完成

- [x] 价格采集器（Google、OpenAI、Anthropic、OpenRouter）
- [x] 翻译服务（火山引擎、百度）
- [x] 数据库存储
- [x] 营销信号检测与动作生成

## 调研备注

### OpenAI 社区论坛
- URL: https://community.openai.com/
- 有大量关于 API 成本、定价的抱怨帖子
- 可通过关键词搜索：`pricing`, `cost`, `expensive`, `bill`
- 无需认证即可访问公开内容

### StackExchange API
- 官方文档: https://api.stackexchange.com/
- 完全免费，有速率限制但无需 API Key
- 可搜索标签：`openai`, `chatgpt`, `llm`, `api-pricing`

### Dev.to API
- 官方 API: https://developers.forem.com/
- 免费访问，可按标签搜索文章
- 标签示例：`ai`, `llm`, `openai`, `pricing`
