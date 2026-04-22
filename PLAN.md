# AI 照片搜索引擎 - 开发计划

> 版本：1.0 | 更新日期：2026-03-21

## 项目概述

### 目标
构建一个基于 AI 的照片搜索引擎，让用户可以通过自然语言描述快速找到照片。

### 范围
- 核心功能：自然语言搜索、智能标签、重复检测
- 目标用户：摄影爱好者、专业摄影师
- 技术栈：Go + Python + React

## 开发阶段

### Phase 1: Python ML 服务 + 基础搜索 (第1-2周)

**目标**：建立 ML 推理能力，实现基础的图像特征提取和搜索

#### 周1：ML 服务基础
| 天 | 任务 | 产出 |
|----|------|------|
| Day 1-2 | 搭建 Python 开发环境，创建 FastAPI 项目骨架 | ml-service 基础结构 |
| Day 2-3 | 集成 CLIP 模型，实现图像编码功能 | clip_model.py |
| Day 3-4 | 实现文本编码功能，支持自然语言查询 | embedding_service.py |
| Day 4-5 | 开发 RESTful API 端点 (/extract, /embed/text) | api/routes.py |
| Day 5 | 编写单元测试，验证模型准确性 | tests/ |

#### 周2：搜索原型
| 天 | 任务 | 产出 |
|----|------|------|
| Day 1-2 | 实现简单向量搜索（内存版 NumPy） | search prototype |
| Day 2-3 | 开发图像分类器（场景/主题） | classifier.py |
| Day 3-4 | 创建 CLI 工具验证搜索效果 | cli tool |
| Day 4-5 | 优化推理性能，添加批量处理 | performance tuning |
| Day 5 | 文档 + API 文档自动生成 | API docs |

**里程碑**：可以通过 Python API 对图片进行特征提取和搜索

---

### Phase 2: Go 核心引擎 + 向量索引 (第3-5周)

**目标**：构建高性能的搜索引擎核心，实现文件管理和向量索引

#### 周3：Go 项目基础
| 天 | 任务 | 产出 |
|----|------|------|
| Day 1-2 | 初始化 Go 项目，搭建目录结构 | project skeleton |
| Day 2-3 | 实现配置管理（Viper）+ 日志（Zap） | config/logger |
| Day 3-4 | 开发文件扫描器（递归扫描） | scanner.go |
| Day 4-5 | 实现文件系统监听（fsnotify） | watcher.go |
| Day 5 | EXIF 元数据提取 | metadata extraction |

#### 周4：向量索引
| 天 | 任务 | 产出 |
|----|------|------|
| Day 1-2 | 集成 FAISS/实现纯 Go 向量搜索 | indexer |
| Day 2-3 | 实现索引构建和持久化 | index storage |
| Day 3-4 | 开发增量索引更新逻辑 | incremental update |
| Day 4-5 | 实现 gRPC 客户端调用 ML 服务 | grpc client |
| Day 5 | SQLite 元数据存储 | database layer |

#### 周5：搜索服务
| 天 | 任务 | 产出 |
|----|------|------|
| Day 1-2 | 开发搜索服务（向量 + 元数据混合搜索） | search service |
| Day 2-3 | 实现结果排序算法 | ranking |
| Day 3-4 | 添加缓存层（内存 LRU） | cache layer |
| Day 4-5 | 开发 HTTP API 层（Gin） | API handlers |
| Day 5 | 集成测试 + 性能测试 | tests |

**里程碑**：Go 引擎可以扫描目录、构建索引、响应搜索请求

---

### Phase 3: React 前端 + 完整集成 (第6-8周)

**目标**：构建用户界面，完成端到端集成

#### 周6：前端基础
| 天 | 任务 | 产出 |
|----|------|------|
| Day 1-2 | 初始化 React + TypeScript 项目 | project setup |
| Day 2-3 | 配置 TailwindCSS，设计 UI 系统 | design system |
| Day 3-4 | 开发布局组件（Header, Sidebar, Layout） | layout components |
| Day 4-5 | 实现搜索框组件 + 防抖 | SearchBar.tsx |
| Day 5 | 搜索建议和历史功能 | suggestions |

#### 周7：核心功能
| 天 | 任务 | 产出 |
|----|------|------|
| Day 1-2 | 开发照片网格组件 + 虚拟滚动 | PhotoGrid.tsx |
| Day 2-3 | 实现懒加载和渐进式图片 | LazyImage.tsx |
| Day 3-4 | 照片详情页面 + 元数据展示 | PhotoDetail.tsx |
| Day 4-5 | 统计图表组件 | StatsChart.tsx |
| Day 5 | 高级筛选功能 | filters |

#### 周8：集成和完善
| 天 | 任务 | 产出 |
|----|------|------|
| Day 1-2 | 前后端 API 集成 | API integration |
| Day 2-3 | 状态管理和错误处理 | state management |
| Day 3-4 | 响应式设计适配 | responsive design |
| Day 4-5 | 用户体验优化（加载状态、动画） | UX polish |
| Day 5 | 端到端测试 | e2e tests |

**里程碑**：完整的可交互 Web 应用，支持自然语言搜索照片

---

### Phase 3.5: AI 对话式分类 (第8-9周)

**目标**：支持自然语言对话，实现智能分类和整理

#### 功能设计

**对话式分类流程：**
```
用户："帮我找所有美食照片"
AI：  "找到 23 张美食照片，按场景分类如下：
       - 餐厅内景：12张
       - 食物特写：8张  
       - 街头小吃：3张"
用户："把餐厅内景的按菜系分"
AI：  "已分类：
       - 火锅：5张
       - 日料：4张
       - 西餐：3张"
```

**支持的对话类型：**
1. **搜索对话**："找去年在故宫拍的照片"
2. **分类对话**："把这些照片按风格整理"
3. **筛选对话**："只看人像，去掉模糊的"
4. **批量操作**："把所有夜景照片移到新文件夹"
5. **智能建议**："你有很多相似照片，要清理吗？"

#### 开发任务

| 天 | 任务 | 产出 |
|----|------|------|
| Day 1-2 | 集成 LLM 对话能力（调用 OpenAI/本地模型） | chat_service.py |
| Day 2-3 | 设计对话状态管理 | dialog_manager.py |
| Day 3-4 | 实现意图识别（搜索/分类/筛选/操作） | intent_classifier.py |
| Day 4-5 | 开发分类执行器（调用搜索+分类API） | action_executor.py |
| Day 5 | 前端聊天界面组件 | ChatInterface.tsx |

#### 技术方案

**后端：**
- FastAPI 对话端点 `/api/v1/chat`
- 对话历史存储（SQLite）
- 意图解析 + 实体提取
- 动态调用搜索/分类API

**前端：**
- 聊天气泡组件
- 快捷操作按钮
- 结果卡片展示
- 文件夹拖拽操作

**核心 API：**
```python
POST /api/v1/chat
{
    "message": "帮我找所有美食照片",
    "context": {
        "current_folder": "/photos",
        "last_results": [...]
    }
}

Response:
{
    "reply": "找到 23 张美食照片...",
    "actions": [
        {"type": "search", "query": "美食"},
        {"type": "classify", "by": "scene"}
    ],
    "results": [...]
}
```

**里程碑**：支持自然语言对话进行照片搜索、分类、整理

---

### Phase 4: 优化、测试、部署 (第10-11周)

**目标**：性能优化，完善测试，准备生产部署

#### 周9：优化和测试
| 天 | 任务 | 产出 |
|----|------|------|
| Day 1-2 | 性能分析和优化 | profiling |
| Day 2-3 | 大规模数据测试（10万+图片） | stress test |
| Day 3-4 | 安全审计和修复 | security |
| Day 4-5 | 编写集成测试和端到端测试 | test coverage |
| Day 5 | 文档完善 | documentation |

#### 周10：部署准备
| 天 | 任务 | 产出 |
|----|------|------|
| Day 1-2 | Docker 容器化 | Dockerfiles |
| Day 2-3 | Docker Compose 编排 | docker-compose.yml |
| Day 3-4 | 编写部署文档和脚本 | deploy scripts |
| Day 4-5 | 生产环境配置 | production config |
| Day 5 | 发布准备和版本标记 | v1.0.0 release |

**里程碑**：生产就绪的 v1.0 版本

---

## 详细任务分解

### 文件清单（按阶段）

#### Phase 1 文件
```
ml-service/
├── app/
│   ├── __init__.py
│   ├── main.py
│   ├── config.py
│   └── dependencies.py
├── models/
│   ├── __init__.py
│   ├── clip_model.py
│   └── classifier.py
├── services/
│   ├── __init__.py
│   ├── feature_extractor.py
│   ├── embedding_service.py
│   └── image_processor.py
├── api/
│   ├── __init__.py
│   ├── routes.py
│   └── schemas.py
├── utils/
│   ├── __init__.py
│   └── image_utils.py
├── tests/
│   ├── __init__.py
│   ├── test_api.py
│   ├── test_models.py
│   └── test_services.py
├── requirements.txt
├── Dockerfile
└── README.md
```

#### Phase 2 文件
```
cmd/
├── search-engine/
│   └── main.go
└── scanner/
    └── main.go

internal/
├── scanner/
│   ├── scanner.go
│   ├── watcher.go
│   └── types.go
├── indexer/
│   ├── index.go
│   ├── search.go
│   └── storage.go
├── search/
│   ├── service.go
│   ├── ranking.go
│   └── filter.go
├── cache/
│   ├── cache.go
│   ├── memory.go
│   └── redis.go
├── api/
│   ├── router.go
│   ├── handlers/
│   │   ├── search.go
│   │   ├── files.go
│   │   └── stats.go
│   └── middleware/
│       ├── cors.go
│       └── logger.go
├── models/
│   ├── photo.go
│   ├── search.go
│   └── vector.go
└── config/
    └── config.go

go.mod
go.sum
Dockerfile
```

#### Phase 3 文件
```
web/
├── public/
│   ├── index.html
│   └── favicon.ico
├── src/
│   ├── components/
│   │   ├── Search/
│   │   │   ├── SearchBar.tsx
│   │   │   ├── SearchSuggestions.tsx
│   │   │   └── SearchHistory.tsx
│   │   ├── Gallery/
│   │   │   ├── PhotoGrid.tsx
│   │   │   ├── PhotoCard.tsx
│   │   │   └── LazyImage.tsx
│   │   ├── Detail/
│   │   │   ├── PhotoDetail.tsx
│   │   │   └── MetadataViewer.tsx
│   │   ├── Stats/
│   │   │   ├── StatsChart.tsx
│   │   │   └── TagCloud.tsx
│   │   └── Common/
│   │       ├── Header.tsx
│   │       ├── Sidebar.tsx
│   │       └── Loading.tsx
│   ├── pages/
│   │   ├── Home.tsx
│   │   ├── Search.tsx
│   │   ├── Gallery.tsx
│   │   └── Settings.tsx
│   ├── hooks/
│   │   ├── useSearch.ts
│   │   ├── usePhotos.ts
│   │   └── useDebounce.ts
│   ├── services/
│   │   ├── api.ts
│   │   └── search.ts
│   ├── types/
│   │   ├── photo.ts
│   │   ├── search.ts
│   │   └── api.ts
│   ├── utils/
│   │   ├── formatters.ts
│   │   └── validators.ts
│   ├── App.tsx
│   └── index.tsx
├── package.json
├── tsconfig.json
├── tailwind.config.js
├── Dockerfile
└── README.md
```

#### Phase 4 文件
```
deploy/
├── docker-compose.yml
├── docker-compose.prod.yml
├── nginx.conf
└── .env.example

scripts/
├── setup.sh
├── build.sh
├── deploy.sh
└── backup.sh

docs/
├── API.md
├── DEPLOYMENT.md
├── DEVELOPMENT.md
└── CHANGELOG.md
```

## 技术依赖图

```
Phase 1 (ML) ─────────┐
                       ├──► Phase 3 (Frontend) ──► Phase 4 (Deploy)
Phase 2 (Go Engine) ──┘
```

- Phase 1 和 Phase 2 可以并行开发
- Phase 3 依赖 Phase 1 和 Phase 2 的 API
- Phase 4 依赖所有前置阶段完成

## 风险和缓解

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| CLIP 模型性能不足 | 高 | 中 | 支持多模型切换，预留优化空间 |
| FAISS 编译复杂 | 中 | 高 | 准备纯 Go 备选方案 |
| 大规模索引内存占用 | 高 | 中 | 实现索引分片和磁盘缓存 |
| 前端大量图片渲染卡顿 | 中 | 中 | 虚拟滚动 + 图片懒加载 |
| 跨语言 gRPC 调试困难 | 低 | 中 | 完善日志和错误处理 |

## 质量标准

### 代码质量
- Go: golangci-lint 通过
- Python: black + flake8 + mypy
- TypeScript: ESLint + Prettier
- 测试覆盖率 > 70%

### 性能目标
- 搜索响应时间 < 500ms (10万图片)
- 索引构建速度 > 1000 图片/分钟
- 前端首次加载 < 3s
- 图片缩略图加载 < 200ms

### 可用性
- 支持 100万+ 图片库
- 支持增量更新，无需全量重建
- 支持断点续扫
- 支持离线使用（索引本地存储）

## 验收标准

### Phase 1 验收
- [ ] 可以通过 API 提取图像特征
- [ ] 可以将文本转换为向量
- [ ] 支持基本的图像分类
- [ ] API 文档完整

### Phase 2 验收
- [ ] 可以扫描指定目录
- [ ] 可以构建向量索引
- [ ] 可以响应搜索请求
- [ ] 支持增量更新

### Phase 3 验收
- [ ] 搜索界面可用
- [ ] 可以展示搜索结果
- [ ] 可以查看照片详情
- [ ] 响应式设计

### Phase 4 验收
- [ ] Docker 部署成功
- [ ] 性能达标
- [ ] 文档完整
- [ ] 可以发布 v1.0

## 后续迭代规划

### v1.1 (Phase 5)
- 人脸识别功能
- 相册智能整理
- 照片质量评估

### v1.2 (Phase 6)
- 视频搜索支持
- 多用户支持
- 云端同步

### v2.0
- 移动端 App
- 实时协作
- 插件系统

---

*详细技术规格请参考 ARCHITECTURE.md*
