# AI 照片搜索引擎 - 架构设计文档

> 版本：1.0 | 更新日期：2026-03-21

## 1. 系统概览

AI照片搜索引擎是一个帮助摄影爱好者和专业摄影师通过自然语言描述快速找到照片的工具。系统采用微服务架构，Go负责高性能搜索核心，Python负责ML推理和AI对话，React提供现代化前端体验。

**核心功能：**
- 🔍 自然语言搜索：用中文描述找照片
- 🤖 AI 对话分类：对话式整理照片
- 🏷️ 智能标签：自动识别场景/主题/人物
- 📊 可视化统计：照片分析和洞察

## 2. 系统架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        React 前端 (Web)                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────────────┐   │
│  │ 搜索界面  │  │ 结果网格  │  │ 照片详情  │  │   统计图表    │   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └───────┬───────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    AI 对话界面                            │   │
│  │  聊天气泡 → 快捷操作 → 结果卡片 → 文件夹拖拽             │   │
│  └─────────────────────────┬────────────────────────────────┘   │
│       └────────────────────┴───────────────────────────────┘   │
│                              │ HTTP/WebSocket                   │
└──────────────────────────────┼──────────────────────────────────┘
                               │
┌──────────────────────────────┼──────────────────────────────────┐
│                     Go 核心引擎 (Backend)                        │
│  ┌───────────────┐  ┌──────┴──────┐  ┌───────────────────┐     │
│  │   API Gateway  │  │  搜索服务    │  │   文件管理服务     │     │
│  │   (gin/echo)   │  │  (gRPC/HTTP)│  │   (扫描/监听)     │     │
│  └───────┬───────┘  └──────┬──────┘  └─────────┬─────────┘     │
│          │                 │                    │               │
│  ┌───────┴───────┐  ┌──────┴──────┐  ┌─────────┴─────────┐     │
│  │   缓存管理     │  │  向量索引   │  │   文件扫描器       │     │
│  │  (Redis/内存)  │  │ (FAISS/Annoy)│  │  (递归+监听)      │     │
│  └───────────────┘  └──────┬──────┘  └───────────────────┘     │
│                            │ gRPC                               │
└────────────────────────────┼────────────────────────────────────┘
                             │
┌────────────────────────────┼────────────────────────────────────┐
│                  Python ML 服务 (推理层)                          │
│  ┌───────────────┐  ┌──────┴──────┐  ┌───────────────────┐     │
│  │   REST API     │  │ CLIP 推理   │  │   特征提取器       │     │
│  │   (FastAPI)    │  │   引擎      │  │  (图像/文本)       │     │
│  └───────────────┘  └─────────────┘  └───────────────────┘     │
│  ┌───────────────┐  ┌─────────────┐  ┌───────────────────┐     │
│  │  图像分类器    │  │  人脸识别    │  │   模型管理器       │     │
│  │  (可选)       │  │  (可选)      │  │  (加载/缓存)       │     │
│  └───────────────┘  └─────────────┘  └───────────────────┘     │
│  ┌────────────────────────────────────────────────────────┐     │
│  │                   AI 对话服务                           │     │
│  │  意图识别 → 实体提取 → 对话管理 → 动作执行             │     │
│  │  (搜索/分类/筛选/操作/建议)                             │     │
│  └────────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────┘
                             │
┌────────────────────────────┼────────────────────────────────────┐
│                        数据存储层                                 │
│  ┌───────────────┐  ┌──────┴──────┐  ┌───────────────────┐     │
│  │   SQLite       │  │  向量存储   │  │   文件系统         │     │
│  │  (元数据)      │  │ (FAISS索引) │  │  (原始图片)        │     │
│  └───────────────┘  └─────────────┘  └───────────────────┘     │
└─────────────────────────────────────────────────────────────────┘
```

## 3. 目录结构

```
photo-search-engine/
├── cmd/                            # Go 应用入口
│   ├── search-engine/              # 主搜索引擎服务入口
│   │   └── main.go
│   └── scanner/                    # 文件扫描服务入口
│       └── main.go
├── internal/                       # Go 内部模块（不对外暴露）
│   ├── scanner/                    # 文件扫描器
│   │   ├── scanner.go              # 递归扫描逻辑
│   │   ├── watcher.go              # 文件系统监听
│   │   └── types.go                # 扫描相关类型定义
│   ├── indexer/                    # 向量索引管理
│   │   ├── index.go                # 索引创建/更新
│   │   ├── search.go               # 向量搜索
│   │   └── storage.go              # 索引持久化
│   ├── search/                     # 搜索服务
│   │   ├── service.go              # 搜索业务逻辑
│   │   ├── ranking.go              # 结果排序
│   │   └── filter.go               # 过滤条件
│   ├── cache/                      # 缓存管理
│   │   ├── cache.go                # 缓存接口
│   │   ├── memory.go               # 内存缓存实现
│   │   └── redis.go                # Redis 缓存实现
│   ├── api/                        # HTTP API 层
│   │   ├── router.go               # 路由定义
│   │   ├── handlers/               # 请求处理器
│   │   │   ├── search.go
│   │   │   ├── files.go
│   │   │   └── stats.go
│   │   └── middleware/              # 中间件
│   │       ├── auth.go
│   │       ├── cors.go
│   │       └── logger.go
│   ├── models/                     # 数据模型
│   │   ├── photo.go                # 照片元数据模型
│   │   ├── search.go               # 搜索请求/响应模型
│   │   └── vector.go               # 向量数据模型
│   └── config/                     # 配置管理
│       └── config.go
├── ml-service/                     # Python ML 服务
│   ├── app/                        # FastAPI 应用
│   │   ├── main.py                 # 应用入口
│   │   ├── config.py               # 配置
│   │   └── dependencies.py         # 依赖注入
│   ├── models/                     # ML 模型定义
│   │   ├── clip_model.py           # CLIP 模型封装
│   │   ├── classifier.py           # 图像分类器
│   │   └── face_recognition.py     # 人脸识别（可选）
│   ├── services/                   # 业务服务
│   │   ├── feature_extractor.py    # 特征提取服务
│   │   ├── embedding_service.py    # 嵌入向量服务
│   │   └── image_processor.py      # 图像预处理
│   ├── api/                        # API 路由
│   │   ├── routes.py               # 路由定义
│   │   └── schemas.py              # 请求/响应模型
│   ├── utils/                      # 工具函数
│   │   ├── image_utils.py          # 图像处理工具
│   │   └── model_utils.py          # 模型工具
│   ├── chat/                       # AI 对话服务
│   │   ├── dialog_manager.py       # 对话状态管理
│   │   ├── intent_classifier.py    # 意图识别
│   │   ├── action_executor.py      # 动作执行器
│   │   └── chat_service.py         # 对话服务主逻辑
│   ├── tests/                      # 测试
│   │   ├── test_api.py
│   │   ├── test_models.py
│   │   └── test_services.py
│   ├── requirements.txt            # Python 依赖
│   └── Dockerfile                  # ML 服务容器
├── web/                            # React 前端
│   ├── public/                     # 静态资源
│   │   ├── index.html
│   │   └── favicon.ico
│   ├── src/                        # 源代码
│   │   ├── components/             # 组件
│   │   │   ├── Search/             # 搜索相关组件
│   │   │   │   ├── SearchBar.tsx    # 搜索框
│   │   │   │   ├── SearchSuggestions.tsx  # 搜索建议
│   │   │   │   └── SearchHistory.tsx     # 搜索历史
│   │   │   ├── Gallery/            # 图库组件
│   │   │   │   ├── PhotoGrid.tsx    # 照片网格
│   │   │   │   ├── PhotoCard.tsx    # 照片卡片
│   │   │   │   └── LazyImage.tsx    # 懒加载图片
│   │   │   ├── Detail/             # 详情组件
│   │   │   │   ├── PhotoDetail.tsx  # 照片详情
│   │   │   │   └── MetadataViewer.tsx  # 元数据查看
│   │   │   ├── Stats/              # 统计组件
│   │   │   │   ├── StatsChart.tsx   # 统计图表
│   │   │   │   └── TagCloud.tsx     # 标签云
│   │   │   ├── Chat/               # AI 对话组件
│   │   │   │   ├── ChatInterface.tsx # 对话界面
│   │   │   │   ├── ChatBubble.tsx   # 聊天气泡
│   │   │   │   └── ChatActions.tsx  # 快捷操作
│   │   │   └── Common/             # 通用组件
│   │   │       ├── Header.tsx
│   │   │       ├── Sidebar.tsx
│   │   │       └── Loading.tsx
│   │   ├── pages/                  # 页面组件
│   │   │   ├── Home.tsx            # 首页
│   │   │   ├── Search.tsx          # 搜索页
│   │   │   ├── Gallery.tsx         # 图库页
│   │   │   └── Settings.tsx        # 设置页
│   │   ├── hooks/                  # 自定义 Hooks
│   │   │   ├── useSearch.ts
│   │   │   ├── usePhotos.ts
│   │   │   └── useDebounce.ts
│   │   ├── services/               # API 服务
│   │   │   ├── api.ts              # API 客户端
│   │   │   └── search.ts           # 搜索 API
│   │   ├── types/                  # TypeScript 类型
│   │   │   ├── photo.ts
│   │   │   ├── search.ts
│   │   │   └── api.ts
│   │   ├── utils/                  # 工具函数
│   │   │   ├── formatters.ts
│   │   │   └── validators.ts
│   │   ├── App.tsx                 # 应用入口
│   │   └── index.tsx               # React 入口
│   ├── package.json
│   ├── tsconfig.json
│   ├── tailwind.config.js
│   └── Dockerfile                  # 前端容器
├── docs/                           # 文档
│   ├── API.md                      # API 文档
│   ├── DEPLOYMENT.md               # 部署文档
│   └── DEVELOPMENT.md              # 开发文档
├── scripts/                        # 脚本
│   ├── setup.sh                    # 环境初始化
│   └── build.sh                    # 构建脚本
├── deploy/                         # 部署配置
│   ├── docker-compose.yml          # Docker Compose
│   └── nginx.conf                  # Nginx 配置
├── ARCHITECTURE.md                 # 本文档
├── PLAN.md                         # 开发计划
└── README.md                       # 项目说明
```

## 4. 核心模块设计

### 4.1 Go 核心引擎

#### 文件扫描器 (scanner)
- **递归扫描**：遍历指定目录，识别支持的图片格式（JPG, PNG, RAW, HEIC等）
- **增量扫描**：基于文件修改时间的增量更新
- **文件监听**：使用 fsnotify 监听文件系统变化，实时更新索引
- **去重检测**：基于文件哈希（MD5/SHA256）检测重复文件

#### 向量索引 (indexer)
- **索引构建**：调用 ML 服务提取特征向量，构建 FAISS/Annoy 索引
- **增量更新**：支持新图片的增量索引，无需全量重建
- **索引持久化**：定期将索引保存到磁盘，支持快速加载
- **多索引支持**：支持图像特征和文本特征的多模态索引

#### 搜索服务 (search)
- **向量搜索**：基于余弦相似度的近似最近邻搜索
- **混合搜索**：结合向量搜索和元数据过滤（时间、地点、标签）
- **结果排序**：综合相关性、时间、质量等因素的智能排序
- **搜索缓存**：热门查询结果缓存，提升响应速度

#### 缓存管理 (cache)
- **多级缓存**：内存缓存（LRU）+ 可选 Redis 缓存
- **缓存策略**：TTL 过期 + LRU 淘汰
- **缓存预热**：启动时预加载热门索引

### 4.2 Python ML 服务

#### CLIP 推理引擎
- **模型加载**：支持 OpenAI CLIP 和 OpenCLIP 模型
- **批量推理**：支持批量图像/文本编码，提升吞吐量
- **GPU 加速**：自动检测 CUDA，支持 GPU 推理
- **模型量化**：支持 INT8 量化，降低内存占用

#### 特征提取器
- **图像特征**：提取 512/768 维图像嵌入向量
- **文本特征**：将自然语言查询转换为向量
- **特征归一化**：L2 归一化，确保向量可比性

#### 图像分类器（可选）
- **场景分类**：风景、人像、建筑、动物等
- **属性识别**：颜色、构图、光照条件
- **EXIF 解析**：提取相机、镜头、参数信息

#### RESTful API
- **/api/extract**：提取图像特征向量
- **/api/classify**：图像分类
- **/api/similar**：查找相似图片
- **/api/health**：健康检查

### 4.3 React 前端

#### 搜索框组件
- **自然语言输入**：支持中英文自然语言描述
- **搜索建议**：基于历史和热门标签的智能建议
- **高级筛选**：时间范围、地点、标签等筛选条件
- **语音输入**：支持语音搜索（可选）

#### 结果展示
- **瀑布流/网格**：多种布局模式切换
- **无限滚动**：懒加载 + 虚拟滚动，支持大量结果
- **预览缩略图**：快速加载缩略图，点击查看详情
- **相似推荐**：基于当前结果推荐相似图片

#### 照片详情
- **高清预览**：支持缩放和平移
- **元数据展示**：EXIF 信息、拍摄参数
- **标签管理**：查看和编辑 AI 生成的标签
- **操作菜单**：下载、分享、删除、移动等

#### 统计图表
- **时间分布**：照片拍摄时间热力图
- **地点分布**：基于地理位置的照片分布
- **标签云**：AI 生成标签的可视化
- **存储统计**：文件大小、格式分布

## 5. 数据流设计

### 5.1 索引构建流程

```
用户指定目录
    │
    ▼
文件扫描器 (Go)
    │ 递归扫描 + 文件监听
    │ 提取 EXIF 元数据
    │ 计算文件哈希
    ▼
ML 服务 (Python)
    │ CLIP 特征提取
    │ 图像分类（可选）
    │ 返回向量 + 标签
    ▼
向量索引 (Go)
    │ 构建 FAISS 索引
    │ 存储元数据到 SQLite
    │ 更新缓存
    ▼
索引就绪
```

### 5.2 搜索流程

```
用户输入查询
    │
    ▼
React 前端
    │ 输入验证
    │ 防抖处理
    │ 发送请求
    ▼
Go API 网关
    │ 请求路由
    │ 参数解析
    │ 缓存检查
    ▼
Python ML 服务
    │ 文本编码
    │ 返回查询向量
    ▼
Go 搜索服务
    │ 向量相似度搜索
    │ 元数据过滤
    │ 结果排序
    ▼
返回结果列表
    │
    ▼
React 前端
    │ 渲染缩略图
    │ 懒加载详情
    │ 展示结果
```

### 5.3 文件变更处理流程

```
文件系统变化
    │
    ▼
文件监听器 (Go)
    │ 检测 CREATE/MODIFY/DELETE
    │ 去抖动处理
    ▼
扫描器
    │ 新文件：提取元数据 → ML 特征提取 → 更新索引
    │ 修改文件：重新提取 → 更新向量 → 更新索引
    │ 删除文件：从索引移除 → 清理元数据
```

## 6. API 设计

### 6.1 Go 搜索引擎 API

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/v1/search` | GET | 自然语言搜索，参数：`q`（查询）、`limit`、`offset` |
| `/api/v1/search/similar` | GET | 相似图片搜索，参数：`photo_id` |
| `/api/v1/photos` | GET | 获取照片列表，支持分页和筛选 |
| `/api/v1/photos/:id` | GET | 获取照片详情和元数据 |
| `/api/v1/photos/:id/tags` | PUT | 更新照片标签 |
| `/api/v1/files/scan` | POST | 触发目录扫描 |
| `/api/v1/files/status` | GET | 获取扫描状态 |
| `/api/v1/stats` | GET | 获取统计信息 |
| `/api/v1/suggestions` | GET | 获取搜索建议 |

### 6.2 Python ML 服务 API

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/v1/extract` | POST | 提取图像特征，参数：`image_path` 或 `image_base64` |
| `/api/v1/extract/batch` | POST | 批量提取特征 |
| `/api/v1/classify` | POST | 图像分类 |
| `/api/v1/embed/text` | POST | 文本编码为向量 |
| `/api/v1/similar` | POST | 计算两图片相似度 |
| `/api/v1/chat` | POST | AI对话，支持搜索/分类/筛选/操作 |
| `/api/v1/chat/history` | GET | 获取对话历史 |
| `/api/v1/health` | GET | 健康检查 |

### 6.3 AI 对话 API

**对话请求：**
```json
POST /api/v1/chat
{
    "message": "帮我找所有美食照片",
    "context": {
        "current_folder": "/photos",
        "last_results": [...]
    }
}
```

**对话响应：**
```json
{
    "reply": "找到 23 张美食照片，按场景分类如下：...",
    "actions": [
        {"type": "search", "query": "美食"},
        {"type": "classify", "by": "scene"}
    ],
    "results": [...],
    "suggestions": ["按餐厅分类", "只看高清的", "导出列表"]
}
```

**支持的意图类型：**
| 意图 | 示例 | 动作 |
|------|------|------|
| 搜索 | "找去年在故宫拍的照片" | 执行搜索 |
| 分类 | "把这些照片按风格整理" | 调用分类器 |
| 筛选 | "只看人像，去掉模糊的" | 过滤结果 |
| 操作 | "把所有夜景照片移到新文件夹" | 文件操作 |
| 建议 | "你有很多相似照片，要清理吗？" | 智能建议 |

### 6.4 API 响应格式

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "results": [...],
    "total": 100,
    "page": 1,
    "limit": 20
  }
}
```

## 7. 技术选型详情

### 7.1 Go 后端

| 组件 | 选型 | 理由 |
|------|------|------|
| Web 框架 | Gin | 高性能、社区活跃、中间件丰富 |
| gRPC | grpc-go | 高效的服务间通信 |
| 向量索引 | go-faiss 或自研 | 依赖 CGO，备选纯 Go 方案 |
| 数据库 | SQLite (go-sqlite3) | 轻量、无需额外服务 |
| 文件监听 | fsnotify | 跨平台文件系统事件 |
| 配置 | Viper | 灵活的配置管理 |
| 日志 | Zap | 高性能结构化日志 |

### 7.2 Python ML 服务

| 组件 | 选型 | 理由 |
|------|------|------|
| Web 框架 | FastAPI | 异步支持、自动生成文档 |
| CLIP 模型 | OpenCLIP | 开源、支持多种模型 |
| 深度学习 | PyTorch | 生态成熟、GPU 支持好 |
| 图像处理 | Pillow + OpenCV | 完善的图像处理能力 |
| 任务队列 | Celery（可选） | 批量处理支持 |

### 7.3 React 前端

| 组件 | 选型 | 理由 |
|------|------|------|
| 框架 | React 18 + TypeScript | 类型安全、生态丰富 |
| 样式 | TailwindCSS | 快速开发、一致性强 |
| 状态管理 | Zustand | 轻量、简洁 |
| 路由 | React Router | 成熟稳定 |
| HTTP 客户端 | Axios + React Query | 请求管理 + 缓存 |
| 虚拟滚动 | react-window | 大量图片性能优化 |
| 图表 | Recharts | React 友好的图表库 |
| 图标 | Lucide React | 精美统一的图标 |

### 7.4 基础设施

| 组件 | 选型 | 理由 |
|------|------|------|
| 容器化 | Docker + Compose | 开发部署一致性 |
| 反向代理 | Nginx | 静态资源 + 负载均衡 |
| 缓存 | Redis（可选） | 生产环境性能提升 |
| 监控 | Prometheus + Grafana（可选） | 系统监控 |

## 8. 数据模型

### 8.1 照片 (Photo)

```sql
CREATE TABLE photos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path TEXT UNIQUE NOT NULL,
    file_name TEXT NOT NULL,
    file_hash TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    mime_type TEXT NOT NULL,
    width INTEGER,
    height INTEGER,
    created_at DATETIME,
    modified_at DATETIME,
    indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    exif_data JSON,
    tags JSON,
    vector_id INTEGER,  -- FAISS 索引中的 ID
    quality_score REAL DEFAULT 0.0,
    is_deleted BOOLEAN DEFAULT FALSE
);
```

### 8.2 搜索历史 (SearchHistory)

```sql
CREATE TABLE search_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    query TEXT NOT NULL,
    query_vector BLOB,
    result_count INTEGER,
    searched_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 8.3 标签 (Tag)

```sql
CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    category TEXT,
    photo_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE photo_tags (
    photo_id INTEGER,
    tag_id INTEGER,
    confidence REAL DEFAULT 1.0,
    source TEXT DEFAULT 'auto',  -- 'auto' or 'manual'
    PRIMARY KEY (photo_id, tag_id),
    FOREIGN KEY (photo_id) REFERENCES photos(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);
```

## 9. 性能考虑

### 9.1 索引优化
- 使用 FAISS 的 IVF 索引，支持百万级图片
- 批量索引构建，减少重建次数
- 索引分片，支持增量更新

### 9.2 搜索优化
- 向量搜索使用 HNSW 算法，O(logN) 复杂度
- 热门查询缓存，减少重复计算
- 异步预加载，提升用户体验

### 9.3 前端优化
- 图片懒加载 + 渐进式加载
- 虚拟滚动，支持十万级图片展示
- WebP 格式缩略图，减少传输大小
- Service Worker 离线缓存

## 10. 安全考虑

- API 请求限流，防止滥用
- 文件路径验证，防止路径遍历攻击
- 输入 sanitization，防止注入攻击
- CORS 配置，限制跨域访问
- 敏感配置环境变量化

## 11. 扩展性设计

- 插件化架构：ML 模型可插拔替换
- 微服务就绪：服务间通过 gRPC 通信，易于拆分
- 多后端支持：向量存储可替换（FAISS → Milvus → Pinecone）
- 多模态扩展：预留视频、音频搜索接口

---

*文档结束 - 如有问题请联系架构团队*
