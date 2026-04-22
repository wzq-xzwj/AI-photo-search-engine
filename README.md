# AI 照片搜索引擎

> 用自然语言找照片，让回忆触手可及

## 项目简介

AI照片搜索引擎是一个帮助摄影爱好者和专业摄影师通过自然语言描述快速找到照片的工具。只需输入"去年夏天在海边拍的照片"或"有猫的照片"，即可瞬间找到目标图片。

## 核心功能

- 🔍 **自然语言搜索** - 用描述性语言找照片
- 🏷️ **智能标签** - AI自动识别照片内容
- 🔄 **重复检测** - 发现并清理重复照片
- 📊 **统计分析** - 照片库数据可视化

## 技术架构

- **Go 核心引擎** - 高性能搜索索引和文件管理
- **Python ML 服务** - CLIP 模型推理和特征提取
- **React 前端** - 现代化用户界面

## 快速开始

### 前置要求

- Go 1.21+
- Python 3.9+
- Node.js 18+
- Docker (可选)

### 本地开发

```bash
# 克隆项目
git clone <repo-url>
cd photo-search-engine

# 启动 ML 服务
cd ml-service
pip install -r requirements.txt
uvicorn app.main:app --reload

# 启动 Go 服务
cd ..
go run cmd/search-engine/main.go

# 启动前端
cd web
npm install
npm run dev
```

### Docker 部署

```bash
cd deploy
docker-compose up -d
```

## 文档

- [架构设计](ARCHITECTURE.md)
- [开发计划](PLAN.md)
- [API 文档](docs/API.md)
- [部署指南](docs/DEPLOYMENT.md)

## 项目结构

```
photo-search-engine/
├── cmd/                # Go 入口
├── internal/           # Go 内部模块
├── ml-service/         # Python ML 服务
├── web/                # React 前端
├── docs/               # 文档
├── scripts/            # 脚本
├── deploy/             # 部署配置
├── ARCHITECTURE.md     # 架构文档
├── PLAN.md             # 开发计划
└── README.md           # 本文件
```

## 开发路线图

- ✅ Phase 1: Python ML 服务 + 基础搜索 (1-2周)
- ⬜ Phase 2: Go 核心引擎 + 向量索引 (2-3周)
- ⬜ Phase 3: React 前端 + 完整集成 (2-3周)
- ⬜ Phase 4: 优化、测试、部署 (1-2周)

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License
