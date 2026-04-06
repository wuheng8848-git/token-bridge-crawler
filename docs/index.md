# Token Bridge Intelligence - 文档索引

> 本文档为项目文档导航，帮助快速定位所需文档。

---

## 一、文档目录

| 目录 | 作用 | 受众 |
|------|------|------|
| **product/** | 产品方向、功能需求、PRD | 产品、运营、新成员 |
| **architecture/** | 系统设计、数据模型、API | 开发者、架构师 |
| **deployment/** | 环境配置、安装部署 | 运维、开发者 |
| **progress/** | 路线图、变更记录、决策 | 所有人 |

---

## 二、文档索引

### 产品文档 (product/)

| 文档 | 内容 | 状态 |
|------|------|------|
| [00-product-charter.md](./product/00-product-charter.md) | ⭐ **产品总纲**（必读，所有 PRD 的上位文档） | 必读 |
| [01-overview.md](./product/01-overview.md) | 产品概述、定位 | 已完成 |
| [PRD-noise-filtering.md](./product/PRD-noise-filtering.md) | 噪声清洗优化 PRD | 已确认 |

### 技术架构 (architecture/)

| 文档 | 内容 | 状态 |
|------|------|------|
| [00-quick-reference.md](./architecture/00-quick-reference.md) | 开发者速查表 | 必读 |
| [01-system-design.md](./architecture/01-system-design.md) | 系统架构设计 | 已完成 |
| [02-data-model.md](./architecture/02-data-model.md) | 数据模型 | 已完成 |
| [03-api.md](./architecture/03-api.md) | API 接口规范 | 已完成 |
| [10-collector-design.md](./architecture/10-collector-design.md) | Tavily 采集器设计 | ✅ 已完成 |
| [11-processor-design.md](./architecture/11-processor-design.md) | 处理层设计 | ✅ 已完成 |
| [12-rules-engine.md](./architecture/12-rules-engine.md) | 规则引擎设计 | ✅ 已完成 |
| [13-profiler-design.md](./architecture/13-profiler-design.md) | 用户画像设计 | ✅ 已完成 |
| [14-sentiment-analyzer.md](./architecture/14-sentiment-analyzer.md) | 情感分析模块 | ✅ 已完成 |
| crawler-deps.dot / .svg | 包依赖关系图 | 参考 |

### 部署实施 (deployment/)

| 文档 | 内容 | 状态 |
|------|------|------|
| [01-environment.md](./deployment/01-environment.md) | 本地开发环境配置 | 已完成 |
| [02-installation.md](./deployment/02-installation.md) | 生产部署 | 已完成 |
| [03-deploy.md](./deployment/03-deploy.md) | 容器化部署指南 | 已完成 |

### 项目进度 (progress/)

| 文档 | 内容 | 状态 |
|------|------|------|
| [01-roadmap.md](./progress/01-roadmap.md) | 功能路线图 | 进行中 |
| decisions/ | 架构决策记录（ADR） | 待补充 |

### 规范文档

| 文档 | 内容 | 状态 |
|------|------|------|
| [README.md](../README.md) | 快速开始、部署说明 | 已完成 |
| [CHANGELOG.md](../CHANGELOG.md) | 版本更新记录 | 已完成 |
| [DOCUMENTATION.md](./DOCUMENTATION.md) | 文档规范（PRD 流程、模板） | 已完成 |

---

## 三、阅读路径

```
新人/AI 入口：

1. README.md              → 快速了解项目
2. product/00-product-charter.md  → 理解产品总纲
3. product/PRD-noise-filtering.md → 理解当前需求
4. architecture/00-quick-reference.md → 速查命令配置
5. architecture/01-system-design.md → 理解技术架构
```

---

**最后更新**：2026-04-05
