# Token Bridge Crawler 待办事项

## 数据源采集器待完善

### 用户痛点采集（需要 API Key）

| 数据源 | 状态 | 所需配置 | 备注 |
|--------|------|----------|------|
| Discord | ❌ 放弃 | `DISCORD_BOT_TOKEN` | 只能采集自己频道，无法进入他人频道 |
| Reddit | ⚠️ 待定 | OAuth 应用凭据 | 可能需要 Reddit App 注册 |
| HackerNews | ⚠️ 调试中 | 无需 Key | 目前无数据返回，需进一步调试 |

### 待调研的无 Key 数据源

- [ ] **Product Hunt** - 产品评论区用户反馈
- [ ] **Twitter/X 公开数据** - 无需认证的公开推文
- [ ] **GitHub Issues/Discussions** - 开源项目的用户反馈
- [ ] **Stack Overflow** - API 相关问答
- [ ] **Indie Hackers** - 创业者社区讨论
- [ ] **Lobsters** - 技术社区
- [ ] **RSS 源** - 各类技术博客 RSS 订阅

## 功能待完善

- [ ] AI 日报功能需要配置 `CRAWLER_AI_API_KEY`
- [ ] 邮件通知功能需要配置 SMTP 凭据
- [ ] 推送主系统需要启动主项目服务或配置远程地址

## 已完成

- [x] 价格采集器（Google、OpenAI、Anthropic、OpenRouter）
- [x] 翻译服务（火山引擎、百度）
- [x] 数据库存储
- [x] 营销信号检测与动作生成
