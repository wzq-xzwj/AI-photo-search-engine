# PhotoLens AI 🖼️

> AI 驱动的智能照片搜索引擎 — 用自然语言找到你的每一张照片

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![Python](https://img.shields.io/badge/Python-3.10+-3776AB?logo=python)](https://python.org)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react)](https://react.dev)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

## ✨ 功能

- 🔍 **自然语言搜索** — "去年夏天在海边的照片" 秒出结果
- 🧠 **语义理解** — Chinese CLIP 图像语义搜索，不依赖文件名
- 🏷️ **智能标签** — 自动分析场景、物体，生成多维度标签
- 📅 **时间筛选** — 按年份、季节、月份自由筛选
- 📁 **多目录管理** — 支持扫描多个文件夹，按目录筛选
- 🗣️ **AI 对话** — 跟照片库聊天，自然交互
- 😊 **人脸识别** — 自动检测并分组人物

## 🚀 一键部署

```bash
git clone https://github.com/YOUR_USERNAME/photo-search-engine.git
cd photo-search-engine
bash setup.sh
```

脚本会自动：安装依赖 → 创建虚拟环境 → 启动全部服务。

## 📋 环境要求

| 依赖 | 最低版本 |
|------|---------|
| Go | 1.21+ |
| Python | 3.10+ |
| Node.js | 18+ |
| npm | 9+ |

## 🏗️ 架构

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  React 前端  │────▶│  Go API 后端  │────▶│  Python ML   │
│  :5173      │     │  :8080       │     │  :8000      │
└─────────────┘     └──────┬───────┘     └──────┬───────┘
                           │                    │
                    ┌──────▼───────┐    ┌───────▼───────┐
                    │   SQLite DB  │    │  Chinese CLIP │
                    │              │    │  向量提取      │
                    └──────────────┘    └───────────────┘
```

## 📁 项目结构

```
photo-search-engine/
├── cmd/server/          # Go 后端入口
├── internal/
│   ├── api/handlers/    # HTTP 处理器（搜索/照片/扫描/AI对话/人脸）
│   ├── indexer/         # 照片索引 & 向量检索
│   ├── query/           # LLM 自然语言查询解析
│   ├── scanner/         # 照片目录扫描
│   ├── db/              # SQLite 数据库
│   └── deepseek/        # DeepSeek API 客户端
├── ml-service/          # Python ML 服务 (FastAPI)
│   ├── app/main.py      # FastAPI 应用
│   ├── services/        # CLIP 嵌入、图像分类
│   └── face_detection.py # 人脸检测
├── web/                 # React + TypeScript 前端
│   └── src/
│       ├── components/  # Navbar, PhotoGrid, SearchBar, ChatPanel
│       └── pages/       # Home, Search, Persons
├── config/              # 标签空间、L2规则
├── scripts/             # 辅助脚本
├── setup.sh             # 一键部署脚本
└── .env.example         # 环境变量模板
```

## ⚙️ 配置

复制环境变量模板并编辑：

```bash
cp .env.example .env
```

必填项：

```env
# LLM API（用于自然语言解析，至少配置一个）
DEEPSEEK_API_KEY=sk-your-key-here
# 或 OPENAI_API_KEY=sk-your-key-here
```

## 🖥️ 手动启动

如果不使用一键脚本，也可以手动分步启动：

### 1. ML 服务

```bash
cd ml-service
python3 -m venv venv && source venv/bin/activate
pip install -r requirements.txt
python -m uvicorn app.main:app --host 0.0.0.0 --port 8000
```

### 2. Go 后端

```bash
# 配置环境变量
export $(cat .env | xargs)

# 编译运行
go run cmd/server/main.go
```

### 3. 前端

```bash
cd web
npm install
npm run dev
```

## 🌐 API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/search?q=&period=&dir=&tags=` | 语义搜索 |
| `GET` | `/api/v1/photos?page=&page_size=&dir=` | 照片列表 |
| `GET` | `/api/v1/photos/file?path=` | 获取照片文件 |
| `POST` | `/api/v1/files/scan?dir=` | 扫描目录 |
| `GET` | `/api/v1/dirs` | 目录列表 |
| `GET` | `/api/v1/stats` | 统计信息 |
| `POST` | `/api/v1/chat` | AI 对话 |
| `POST` | `/api/v1/faces/detect` | 人脸检测 |
| `GET` | `/api/v1/persons` | 人物列表 |

## 💡 使用指南

1. **首次使用**：点击右上角扫描按钮 → 选择照片目录 → 等待扫描完成
2. **搜索照片**：输入自然语言描述，如 "2024年冬天在雪地里的照片"
3. **筛选结果**：使用时间筛选、标签筛选、目录筛选
4. **查看人物**：点击导航栏"人物"查看自动分组的人脸

## 🔧 故障排查

| 问题 | 检查方法 |
|------|---------|
| ML 服务未启动 | `curl http://localhost:8000/api/v1/health` |
| 后端未启动 | `curl http://localhost:8080/api/v1/dirs` |
| 搜索无结果 | 检查 LLM API Key 配置是否正确 |
| 内存不足 | 停止 Ollama 等大型服务释放内存 |

## 📄 License

MIT
