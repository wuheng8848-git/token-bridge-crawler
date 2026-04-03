# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

---

## [2.1.0] - 2026-04-01

### Added

- **Discord 真实 API 采集器**：
  - 使用 DiscordGo 库接入 Discord Bot API
  - 支持多频道监控（配置 `DISCORD_CHANNEL_IDS`）
  - 关键词过滤（API、pricing、cost、OpenAI 等）
  - 采集消息内容、作者、反应、附件等元数据
- **环境变量配置**：
  - `DISCORD_BOT_TOKEN`：Discord Bot Token
  - `DISCORD_CHANNEL_IDS`：监控频道列表（逗号分隔）

### Changed

- 重构 `community/discord.go`：从模拟数据改为真实 API

---

## [2.0.0] - 2026-03-31

### Changed

- **项目定位升级**：从"厂商刊例价抓取服务"升级为"情报感知系统"
- **项目名称**：Token Bridge Crawler → Token Bridge Intelligence
- **主入口**：`cmd/crawler/` → `cmd/intelligence/`

### Added

- **情报采集系统**：支持价格、用户痛点、配置痛点、工具生态等多类型情报采集
- **营销决策层**：
  - 6 种信号检测器（成本压力、配置摩擦、工具碎片化、治理起点、迁移意愿、泛兴趣）
  - 5 种动作生成器（内部备注、策略建议、短回应、技术文章、竞品对比）
- **翻译服务**：
  - 火山引擎翻译（200万字符/月免费，172ms）
  - 百度大模型翻译（100万字符/月免费）
  - 百度通用翻译（5万字符/月免费）
  - 自动故障切换机制
- **数据表**：
  - `intelligence_items` 情报主表
  - `customer_signals` 客户信号表
  - `marketing_actions` 营销动作表
- **文档体系**：
  - `index.md` 项目总览
  - `docs/00-quick-reference.md` 速查表
  - `docs/07-customer-signal-model.md` 信号模型说明

### Fixed

- 火山引擎翻译 API 签名错误（`credentialScope` 缺少 `/request` 后缀）

### Security

- 所有营销动作为"内部建议层"，不自动外发（`Channel: internal`, `AutoExecute: false`, `Status: draft`）

---

## [1.0.0] - 2026-03-27

### Added

- 项目初始化
- 厂商刊例价抓取：Google Gemini、OpenAI、Anthropic
- 定时调度：Cron 表达式配置
- 限流降级：自动检测限流，失败后指数退避
- 历史版本：价格快照存储
- AI 日报：自动生成总结并邮件发送
- 主项目推送：通过 API 导入到 TB Staging

---

## 版本说明

- **主版本号**：重大架构变更或定位调整
- **次版本号**：新功能添加
- **修订号**：Bug 修复和小改进