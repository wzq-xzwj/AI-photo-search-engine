# PhotoLens AI — 影子的项目分析与改进计划

> 分析日期: 2026-05-04
> 分析者: 影子 🦞

---

## 📊 项目概况

| 维度 | 数据 |
|------|------|
| **项目名** | PhotoLens AI — AI 驱动的智能照片搜索引擎 |
| **技术栈** | Go 1.21 (Gin) + Python 3.10 (FastAPI) + React 18 (Vite + TailwindCSS) |
| **代码量** | Go 6628行 + Python 2127行 + 前端 2633行 + 测试 377行 = **~11,765 行** |
| **数据库** | SQLite (10MB, photos.db) |
| **向量引擎** | Milvus (已接入) + 内存 JSON 备选 |
| **ML 模型** | Chinese CLIP (语义搜索) + InsightFace (人脸检测) |
| **LLM** | DeepSeek (自然语言查询解析) |
| **最近提交** | 人脸识别系统优化 (2026-05-03) |

---

## 🏗️ 架构评估

### ✅ 已经做得不错的

1. **三层微服务架构** — Go API / Python ML / React 前端，职责清晰
2. **SQLite 持久化** — 已实现，有完整 schema（photos/tags/faces/persons 等表）
3. **LRU 缓存** — indexer 已用 `hashicorp/golang-lru`，1000 条查询 + 500 条结果
4. **连接池** — MLClient 已配置 HTTP 连接池（MaxIdleConns=100）
5. **路径安全** — `sanitizeScanPath` 有白名单 + 符号链接解析 + 敏感目录拦截
6. **CORS 限制** — 已从 `*` 改为白名单，支持环境变量覆盖
7. **统一配置** — `internal/config` 有单例 + 环境变量 + 文件多层覆盖
8. **人脸识别** — InsightFace + Top-K 聚类 + Unknown 状态 + 质量过滤
9. **Milvus 向量索引** — 已从暴力搜索升级到 Milvus

### ⚠️ 仍需改进的问题

| 优先级 | 问题 | 影响 | 位置 |
|--------|------|------|------|
| 🔴 P0 | Milvus 依赖过重，离线时搜索完全不可用 | 单机部署复杂 | `vector_index.go` |
| 🔴 P0 | 无 graceful shutdown，强退可能丢数据 | 数据完整性 | `cmd/server/main.go` |
| 🟡 P1 | 前端无虚拟滚动，>200 张照片性能差 | 用户体验 | `PhotoGrid.tsx` |
| 🟡 P1 | scanner `similarityLabels` 与 config 重复加载 | 维护性 | `scanner.go` |
| 🟡 P1 | 无错误恢复机制，ML 服务断开时直接 panic | 可用性 | 多处 handler |
| 🟢 P2 | 前端无 TypeScript 类型定义文件 | 开发体验 | `web/src/` |
| 🟢 P2 | 无 API 请求/响应统一封装 | 一致性 | handlers |
| 🟢 P2 | 日志使用 `fmt.Printf` 与 `zap.Logger` 混用 | 可观测性 | 全局 |

---

## 🎯 改进计划

### 第一阶段：稳定性修复（立即执行）

#### 1.1 Graceful Shutdown
**问题**: `cmd/server/main.go` 直接 `engine.Run()`，无法优雅关闭
**方案**: 添加 signal 监听 + context cancel + 等待进行中的请求完成

#### 1.2 Milvus 连接容错
**问题**: Milvus 不可达时 VectorIndex 为 nil，但调用方未全部检查
**方案**: 添加内存 fallback — Milvus 不可用时退化为 brute-force 搜索（小数据集可接受）

#### 1.3 ML 服务断连容错
**问题**: ML 服务挂了之后，搜索/扫描直接报错
**方案**: 健康检查 + 重试 + 降级为文件名/标签搜索

#### 1.4 统一日志
**问题**: `fmt.Printf` 和 `zap.Logger` 混用
**方案**: 全局替换 `fmt.Printf` 为 `zap` 调用

### 第二阶段：前端体验优化

#### 2.1 虚拟滚动 / 无限加载
**问题**: PhotoGrid 一次性渲染所有照片
**方案**: IntersectionObserver + 分页加载 + 图片懒加载

#### 2.2 搜索体验增强
- 搜索时显示 loading skeleton
- 搜索历史与热门推荐
- 键盘快捷键 (/ 聚焦搜索)

### 第三阶段：代码质量

#### 3.1 统一 similarityLabels 加载
scanner.go 的 `loadSimilarityLabels()` 应使用 `config.Get().LabelSpacePath`

#### 3.2 API 响应标准化
统一返回格式 `{code, message, data}`

#### 3.3 增加测试覆盖
当前仅 377 行测试代码 (3.2% 覆盖率)，关键路径需补测试

---

## 📋 执行清单

我将按以下顺序实施**第一阶段**的改进：

- [ ] **1.1** Graceful shutdown — `cmd/server/main.go`
- [ ] **1.2** Milvus fallback — 内存向量搜索作为备选
- [ ] **1.3** ML 服务容错 — 健康检查 + 重试 + 降级
- [ ] **1.4** 统一日志 — 替换 fmt.Printf → zap

预计工作量：~3-4 小时

---

*计划由影子 🦞 生成，准备开始实施*
