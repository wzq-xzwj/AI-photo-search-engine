# PhotoLens AI - 智能照片搜索引擎

基于 AI 的照片智能搜索和管理系统，支持自然语言搜索、语义理解和多维度筛选。

## 功能特性

- 🔍 **自然语言搜索** - 支持"去年夏天在故宫的照片"等自然语言查询
- 🧠 **语义理解** - 基于 Chinese CLIP 的图像语义搜索
- 🏷️ **智能标签** - 自动提取照片标签（场景、物体、时间等）
- 📅 **时间筛选** - 支持按年份、季节、月份筛选
- 📁 **目录管理** - 多目录扫描和管理
- 💬 **AI 对话** - 与照片进行自然语言交互

## 技术栈

- **后端**: Go (Gin) + Python (FastAPI)
- **前端**: React + TypeScript + Tailwind CSS + Vite
- **AI/ML**: Chinese CLIP (OFA-Sys/chinese-clip-vit-base-patch16)
- **向量检索**: 余弦相似度 + 向量索引
- **LLM**: DeepSeek / OpenAI 兼容 API

## 项目结构

```
photo-search-engine/
├── cmd/server/          # Go 后端入口
├── internal/            # 内部模块
│   ├── api/handlers/    # HTTP 处理器
│   ├── indexer/         # 照片索引管理
│   ├── query/           # 查询解析 (LLM)
│   ├── scanner/         # 照片扫描
│   └── ml_client.go     # ML 服务客户端
├── ml-service/          # Python ML 服务
│   └── app/
│       └── main.py      # FastAPI 应用
├── web/                 # React 前端
│   ├── src/
│   │   ├── components/  # UI 组件
│   │   ├── pages/       # 页面
│   │   └── App.tsx      # 应用入口
│   └── index.html
├── vector_index.json    # 向量索引文件
├── photo_tags.json      # 照片标签数据
└── start.sh             # 一键启动脚本
```

## 快速开始

### 方式一：使用启动脚本（推荐）

```bash
# 1. 进入项目目录
cd /Users/wuzhaoqing/Pictures/photo-search-engine

# 2. 运行启动脚本
./start.sh
```

脚本会自动：
- 启动 ML 服务（端口 8000）
- 等待 ML 服务就绪
- 启动 Go 后端（端口 8080）
- 启动前端开发服务器（端口 5173）

### 方式二：手动启动

#### 1. 启动 ML 服务

```bash
cd ml-service
source venv/bin/activate
python -m uvicorn app.main:app --host 0.0.0.0 --port 8000
```

#### 2. 启动 Go 后端

```bash
# 设置环境变量
export LLM_PROVIDER=openai
export OPENAI_BASE_URL=https://api.deepseek.com/v1
export OPENAI_API_KEY=your_api_key
export OPENAI_MODEL=deepseek-chat

# 启动服务
./photo-server
```

#### 3. 启动前端

```bash
cd web
npm run dev
```

## 访问应用

- **前端界面**: http://localhost:5173
- **后端 API**: http://localhost:8080
- **ML 服务**: http://localhost:8000

## API 端点

### 照片管理

- `GET /api/v1/photos` - 获取照片列表
- `GET /api/v1/photos/:id` - 获取单张照片
- `GET /api/v1/photos/file?path=xxx` - 获取照片文件
- `POST /api/v1/photos/scan` - 扫描目录
- `POST /api/v1/photos/classify-all` - 批量分类

### 搜索

- `GET /api/v1/search?q=query` - 语义搜索
  - 参数: `q` (查询词), `period` (时间筛选), `dir` (目录筛选)
  - 示例: `/api/v1/search?q=去年拍的照片&period=all`

### 目录

- `GET /api/v1/dirs` - 获取所有目录

### 统计

- `GET /api/v1/stats` - 获取统计信息

### AI 对话

- `POST /api/v1/chat` - 与 AI 对话

## 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `LLM_PROVIDER` | LLM 提供商 | `ollama` |
| `OPENAI_BASE_URL` | OpenAI 兼容 API 地址 | - |
| `OPENAI_API_KEY` | API 密钥 | - |
| `OPENAI_MODEL` | 模型名称 | `deepseek-chat` |
| `PORT` | 后端端口 | `8080` |

## 使用指南

### 1. 首次使用

1. 点击右上角扫描按钮
2. 输入照片目录路径（如 `~/Pictures`）
3. 等待扫描完成

### 2. 搜索照片

- 在搜索框输入自然语言查询
- 支持时间限定："去年"、"夏天"、"2024年"
- 支持场景描述："故宫"、"人像"、"美食"

### 3. 筛选结果

- 时间筛选：本周/本月/本年
- 标签筛选：点击热门标签
- 目录筛选：右上角目录选择器

## 故障排查

### 照片不显示

1. 检查 ML 服务是否运行：`curl http://localhost:8000/api/v1/health`
2. 检查后端日志：`tail -f /tmp/go-backend.log`
3. 确认照片文件存在且路径正确

### 搜索无结果

1. 检查 LLM 配置是否正确
2. 查看后端日志中的 LLM 解析结果
3. 确认向量索引已生成

### 标签显示"未分类"

1. 检查 `photo_tags.json` 文件是否存在
2. 重新运行扫描或分类任务

## 开发说明

### 重新编译后端

```bash
cd /Users/wuzhaoqing/Pictures/photo-search-engine
go build -o photo-server ./cmd/server/main.go
```

### 前端开发

```bash
cd web
npm install
npm run dev
```

### 添加新功能

1. 后端：在 `internal/api/handlers/` 添加 handler
2. 在 `cmd/server/main.go` 注册路由
3. 前端：在 `web/src/` 添加组件或页面

## 数据文件

- `vector_index.json` - 照片向量索引（自动生成）
- `photo_tags.json` - 照片标签数据（自动生成）

## 注意事项

- 首次扫描可能需要较长时间（取决于照片数量）
- 建议定期备份 `vector_index.json` 和 `photo_tags.json`
- 大目录扫描时可能需要调整系统文件描述符限制

## License

MIT
