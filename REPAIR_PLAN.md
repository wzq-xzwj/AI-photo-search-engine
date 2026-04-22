# 照片搜索引擎 - 严重问题修复方案

> 生成日期: 2026-04-22
> 项目: /Users/wuzhaoqing/Pictures/photo-search-engine
> 基于: Opencode 代码分析 + 架构文档审查

## 执行摘要

当前代码存在 **4 个严重问题** 和 **6 个性能瓶颈**，限制了生产环境使用。本方案按优先级排序，分 3 个迭代修复。

---

## 🔴 P0: 严重问题（立即修复）

### 问题 1: 暴力向量搜索 → O(N) 复杂度

**现状**: `internal/vector_index.go` 使用纯内存 JSON + 双层循环暴力计算余弦相似度
**影响**: 1000 张照片时搜索延迟 >2s；5000 张时 >10s，不可用
**代码位置**: `internal/vector_index.go:Search()` 方法

**修复方案**:
```
选项 A: 接入 FAISS（推荐）
- 使用 faiss-go 绑定
- IVF 索引，O(logN) 复杂度
- 支持 100万+ 图片
- 工作量: ~2天

选项 B: 纯 Go 实现 HNSW
- 无 CGO 依赖
- 性能略低于 FAISS 但足够
- 工作量: ~3天

推荐: 选项 A（FAISS 成熟稳定，社区支持好）
```

---

### 问题 2: 无持久化存储 → 重启丢失索引

**现状**: 元数据仅存内存，重启后从 `vector_index.json` 恢复（但非结构化）
**影响**: 每次重启需重新扫描，15 张照片需 30 秒，1000 张需 30 分钟
**代码位置**: `internal/indexer/index.go` + `internal/api/handlers/files.go`

**修复方案**:
```
选项 A: SQLite 持久化（推荐）
- 轻量、无需额外服务
- 支持完整 SQL 查询
- 索引分片存储
- 工作量: ~2天

选项 B: BoltDB（纯 Go）
- 无 CGO
- 键值存储，查询能力弱
- 工作量: ~1.5天

推荐: 选项 A（需要复杂查询支持标签、时间过滤）
```

---

### 问题 3: 缓存无上限 → 内存泄漏

**现状**: `internal/indexer/index.go:75-159` 使用无界 map，无 TTL/LRU
**影响**: 长时间运行后 OOM；查询缓存和结果缓存无限增长
**代码位置**: `internal/indexer/index.go`

**修复方案**:
```go
// 当前问题代码
idx.queryCache[queryStr] = queryEmbedding  // 永不清除
idx.resultCache[cacheKey] = photos         // 永不清除

// 修复: 使用 github.com/hashicorp/golang-lru
import lru "github.com/hashicorp/golang-lru"

type Index struct {
    queryCache *lru.Cache  // 限制 1000 条
    resultCache *lru.Cache // 限制 500 条，TTL 5 分钟
}
```
**工作量**: ~4 小时

---

### 问题 4: 路径遍历漏洞 → 安全风险

**现状**: `HandleScan` 直接拼接用户输入路径，未验证
**影响**: 攻击者可读取任意目录（`/etc/passwd`、系统文件）
**代码位置**: `internal/api/handlers/files.go:HandleScan`

**修复方案**:
```go
func sanitizeScanPath(input string) (string, error) {
    // 1. 禁止绝对路径穿越
    // 2. 限制在白名单目录内（~/Pictures 或用户指定）
    // 3. 使用 filepath.Clean + 验证前缀
    // 4. 符号链接解析验证
}
```
**工作量**: ~2 小时

---

## 🟡 P1: 性能瓶颈（本周修复）

### 瓶颈 5: 同步文件扫描阻塞 API

**现状**: 扫描在主线程执行，前端轮询进度但后端阻塞
**影响**: 扫描期间无法搜索；大目录扫描时服务不可用
**代码位置**: `internal/scanner/scanner.go`

**修复方案**:
```
1. 扫描放入后台 goroutine
2. 使用 channel 报告进度
3. 支持取消操作（context.Cancel）
4. 扫描期间只读搜索可用（读写锁）
```
**工作量**: ~1 天

---

### 瓶颈 6: 无连接池 → HTTP 连接开销

**现状**: `internal/ml_client.go` 每次请求新建 HTTP 连接
**影响**: 高并发时连接耗尽，延迟增加
**代码位置**: `internal/ml_client.go`

**修复方案**:
```go
// 使用 http.Client 连接池
type MLClient struct {
    client *http.Client  // 复用连接
    baseURL string
}

func NewMLClient(baseURL string) *MLClient {
    return &MLClient{
        client: &http.Client{
            Timeout: 30 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 10,
                IdleConnTimeout:     90 * time.Second,
            },
        },
        baseURL: baseURL,
    }
}
```
**工作量**: ~2 小时

---

### 瓶颈 7: 前端无分页 → 大量照片卡顿

**现状**: `PhotoGrid.tsx` 一次性加载所有结果，无虚拟滚动
**影响**: >100 张照片时浏览器卡顿；>1000 张时页面崩溃
**代码位置**: `web/src/components/PhotoGrid.tsx`

**修复方案**:
```
1. 后端支持分页 API（page/page_size）
2. 前端接入 IntersectionObserver 无限滚动
3. 虚拟滚动只渲染可视区域
4. 缩略图懒加载
```
**工作量**: ~1.5 天

---

### 瓶颈 8: 重复加载配置

**现状**: `similarityLabels` 在 `scanner.go`、`photos.go` 多处重复定义
**影响**: 配置变更需改多处；不一致风险
**代码位置**: `internal/api/handlers/photos.go`、`internal/scanner/scanner.go`

**修复方案**:
```
1. 统一配置管理包（internal/config）
2. 支持热重载（文件监听）
3. 单例模式 + sync.Once
```
**工作量**: ~3 小时

---

### 瓶颈 9: 低效字符串匹配

**现状**: `search.go:483-492` 手写循环实现 `strings.Contains`
**影响**: 代码可读性差，性能无优势
**代码位置**: `internal/api/handlers/search.go`

**修复方案**:
```go
// 删除自定义函数，直接使用标准库
import "strings"

// 替换: searchString(s, substr) → strings.Contains(s, substr)
```
**工作量**: ~30 分钟

---

### 瓶颈 10: CORS 配置过宽

**现状**: 允许所有来源访问（`*`）
**影响**: CSRF 风险；生产环境不安全
**代码位置**: `internal/api/middleware/cors.go`

**修复方案**:
```go
// 生产环境限制来源
allowedOrigins := []string{
    "http://localhost:5173",  // 开发
    "http://localhost:3000",  // 生产构建
    // 用户配置的域名
}
```
**工作量**: ~1 小时

---

## 📋 实施计划

### 迭代 1: 安全 + 缓存（2 天）
| 任务 | 文件 | 工作量 |
|------|------|--------|
| 修复路径遍历漏洞 | `files.go` | 2h |
| 添加 LRU 缓存 | `index.go` | 4h |
| 限制 CORS | `cors.go` | 1h |
| 替换字符串匹配 | `search.go` | 0.5h |

### 迭代 2: 持久化 + 架构（3 天）
| 任务 | 文件 | 工作量 |
|------|------|--------|
| SQLite 集成 | 新增 `internal/db` | 2d |
| 统一配置管理 | 新增 `internal/config` | 1d |
| 扫描器异步化 | `scanner.go` | 1d |

### 迭代 3: 向量索引 + 前端（4 天）
| 任务 | 文件 | 工作量 |
|------|------|--------|
| FAISS 集成 | `vector_index.go` | 2d |
| HTTP 连接池 | `ml_client.go` | 0.5d |
| 前端分页 + 虚拟滚动 | `PhotoGrid.tsx` | 1.5d |

**总计: 9 个工作日**

---

## 🎯 验收标准

### 性能指标
- [ ] 搜索延迟 < 500ms（1000 张照片）
- [ ] 内存占用稳定（24 小时运行不 OOM）
- [ ] 重启后 < 5 秒恢复（从 SQLite 加载）
- [ ] 扫描期间搜索可用

### 安全指标
- [ ] 路径遍历漏洞修复（测试用例通过）
- [ ] CORS 限制生效
- [ ] 输入验证覆盖所有 API

### 代码质量
- [ ] 配置单一来源
- [ ] 无重复代码
- [ ] 标准库优先

---

## NOT in scope（明确排除）

| 项目 | 原因 | 后续计划 |
|------|------|----------|
| gRPC 通信 | 当前 HTTP 足够，gRPC 增加复杂度 | v1.1 |
| Redis 缓存 | SQLite + LRU 已足够 | v1.2 大规模时 |
| 人脸识别 | 需要额外模型和训练 | v1.1 |
| 移动端 App | 当前 Web 优先 | v2.0 |
| 多用户支持 | 单用户工具定位 | v1.2 |
| 云端同步 | 本地优先设计 | v2.0 |

---

## What already exists（可复用）

| 组件 | 状态 | 复用方式 |
|------|------|----------|
| 向量索引 JSON 格式 | 可用 | 迁移到 SQLite 时保留解析逻辑 |
| 扫描进度 API | 可用 | 异步化时保留接口 |
| ML 服务 API | 可用 | 无需改动 |
| 前端组件结构 | 可用 | 分页时复用 PhotoCard |
| 一键启动脚本 | 可用 | 无需改动 |

---

## Failure Modes（生产风险）

| 场景 | 风险 | 缓解 |
|------|------|------|
| FAISS 索引损坏 | 搜索不可用 | 保留 JSON 备份机制 |
| SQLite 锁定 | 并发写入失败 | WAL 模式 + 重试 |
| ML 服务断开 | 无法提取特征 | 降级为文件名搜索 |
| 大目录扫描 | 内存耗尽 | 流式处理 + 分批 |

---

*方案结束 - 等待主人确认后执行*
