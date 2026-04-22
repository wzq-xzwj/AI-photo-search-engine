# Milvus 向量搜索集成方案

> 生成日期: 2026-04-23
> 项目: /Users/wuzhaoqing/Pictures/photo-search-engine

## 概述

使用 Milvus 替代 FAISS 作为向量搜索引擎，提供分布式、高性能的向量存储和检索能力。

---

## ✅ Milvus 优势

| 特性 | Milvus | FAISS | 说明 |
|------|--------|-------|------|
| **部署模式** | 独立服务/Docker | 嵌入式库 | Milvus 更适合生产环境 |
| **Go SDK** | ✅ 官方支持 | ❌ CGO 绑定 | `milvus-sdk-go/v2` 可用 |
| **持久化** | ✅ 内置 | ❌ 需手动保存 | Milvus 自动持久化 |
| **分布式** | ✅ 支持 | ❌ 单机 | Milvus 支持集群扩展 |
| **索引类型** | IVF、HNSW、FLAT 等 | 有限 | Milvus 更丰富 |
| **混合搜索** | ✅ 向量+标量 | ❌ 仅向量 | Milvus 支持过滤 |
| **管理界面** | ✅ Attu | ❌ 无 | 可视化管理和监控 |

---

## ⚠️ 注意事项

| 问题 | 说明 | 缓解方案 |
|------|------|----------|
| **额外服务** | 需要运行 Milvus 服务 | Docker Compose 一键启动 |
| **资源占用** | 最低 2GB 内存 | 开发环境用 standalone 模式 |
| **复杂度** | 比 FAISS 复杂 | 封装 SDK，隐藏细节 |
| **网络依赖** | 需要连接服务 | 本地 Docker 部署 |

---

## 🏗️ 架构设计

```
┌─────────────────────────────────────────┐
│           Go 后端引擎                    │
│  ┌─────────────┐  ┌─────────────────┐  │
│  │  API Handlers│  │  Milvus Client  │  │
│  │  (search.go) │  │  (vector_index) │  │
│  └──────┬──────┘  └────────┬────────┘  │
│         │                  │ gRPC      │
│  ┌──────┴──────────────────┴────────┐  │
│  │         SQLite (元数据)            │  │
│  └──────────────────────────────────┘  │
└─────────────────────────────────────────┘
                   │
                   │ gRPC/HTTP
                   ▼
┌─────────────────────────────────────────┐
│         Milvus Standalone               │
│  ┌─────────────┐  ┌─────────────────┐   │
│  │  Query Node  │  │  Index Node     │   │
│  │  (搜索)      │  │  (构建索引)      │   │
│  └─────────────┘  └─────────────────┘   │
│  ┌─────────────┐  ┌─────────────────┐   │
│  │  Data Node   │  │  MinIO (存储)    │   │
│  │  (数据写入)  │  │  (对象存储)      │   │
│  └─────────────┘  └─────────────────┘   │
└─────────────────────────────────────────┘
```

---

## 📦 部署方案

### 开发环境（Standalone）

```yaml
# docker-compose.yml
version: '3.5'

services:
  etcd:
    image: quay.io/coreos/etcd:v3.5.5
    environment:
      - ETCD_AUTO_COMPACTION_MODE=revision
      - ETCD_AUTO_COMPACTION_RETENTION=1000
      - ETCD_QUOTA_BACKEND_BYTES=4294967296
    volumes:
      - etcd-data:/etcd
    command: etcd -advertise-client-urls=http://127.0.0.1:2379 -listen-client-urls http://0.0.0.0:2379 --data-dir /etcd

  minio:
    image: minio/minio:RELEASE.2023-03-20T20-16-50Z
    environment:
      MINIO_ACCESS_KEY: minioadmin
      MINIO_SECRET_KEY: minioadmin
    volumes:
      - minio-data:/minio_data
    command: minio server /minio_data
    ports:
      - "9001:9001"

  milvus-standalone:
    image: milvusdb/milvus:v2.4.1
    environment:
      ETCD_ENDPOINTS: etcd:2379
      MINIO_ADDRESS: minio:9000
    ports:
      - "19530:19530"
      - "9091:9091"
    depends_on:
      - etcd
      - minio

  attu:
    image: zilliz/attu:v2.4
    environment:
      MILVUS_URL: milvus-standalone:19530
    ports:
      - "8000:3000"
    depends_on:
      - milvus-standalone

volumes:
  etcd-data:
  minio-data:
```

启动命令：
```bash
docker-compose up -d
```

---

## 🔧 代码集成

### 1. Milvus Client 封装

```go
// internal/vector/milvus.go
package vector

import (
    "context"
    "fmt"
    "time"
    
    "github.com/milvus-io/milvus-sdk-go/v2/client"
    "github.com/milvus-io/milvus-sdk-go/v2/entity"
)

const (
    CollectionName = "photos"
    Dim            = 512  // CLIP 向量维度
    MetricType     = entity.IP  // 内积相似度
)

type MilvusIndex struct {
    client client.Client
    ctx    context.Context
}

func NewMilvusIndex(addr string) (*MilvusIndex, error) {
    ctx := context.Background()
    c, err := client.NewClient(ctx, client.Config{
        Address: addr,
    })
    if err != nil {
        return nil, fmt.Errorf("connect milvus: %w", err)
    }
    
    idx := &MilvusIndex{
        client: c,
        ctx:    ctx,
    }
    
    // 确保集合存在
    if err := idx.createCollection(); err != nil {
        return nil, err
    }
    
    return idx, nil
}

func (m *MilvusIndex) createCollection() error {
    // 检查集合是否存在
    has, err := m.client.HasCollection(m.ctx, CollectionName)
    if err != nil {
        return err
    }
    if has {
        return nil
    }
    
    // 创建集合
    schema := entity.NewSchema().
        WithName(CollectionName).
        WithDescription("Photo search vectors").
        WithField(entity.NewField().
            WithName("id").
            WithDataType(entity.FieldTypeInt64).
            WithIsPrimaryKey(true).
            WithIsAutoID(true)).
        WithField(entity.NewField().
            WithName("photo_id").
            WithDataType(entity.FieldTypeVarChar).
            WithMaxLength(256)).
        WithField(entity.NewField().
            WithName("file_path").
            WithDataType(entity.FieldTypeVarChar).
            WithMaxLength(512)).
        WithField(entity.NewField().
            WithName("vector").
            WithDataType(entity.FieldTypeFloatVector).
            WithDim(Dim))
    
    if err := m.client.CreateCollection(m.ctx, schema, entity.DefaultShardNumber); err != nil {
        return fmt.Errorf("create collection: %w", err)
    }
    
    // 创建索引
    idx, err := entity.NewIndexIvfFlat(entity.IP, 128)
    if err != nil {
        return err
    }
    
    if err := m.client.CreateIndex(m.ctx, CollectionName, "vector", idx, false); err != nil {
        return fmt.Errorf("create index: %w", err)
    }
    
    // 加载集合到内存
    if err := m.client.LoadCollection(m.ctx, CollectionName, false); err != nil {
        return fmt.Errorf("load collection: %w", err)
    }
    
    return nil
}

// AddPhoto 添加照片向量
func (m *MilvusIndex) AddPhoto(photoID, filePath string, vector []float32) error {
    idColumn := entity.NewColumnInt64("id", []int64{})
    photoIDColumn := entity.NewColumnVarChar("photo_id", []string{photoID})
    pathColumn := entity.NewColumnVarChar("file_path", []string{filePath})
    vectorColumn := entity.NewColumnFloatVector("vector", Dim, [][]float32{vector})
    
    _, err := m.client.Insert(m.ctx, CollectionName, "", idColumn, photoIDColumn, pathColumn, vectorColumn)
    return err
}

// Search 向量搜索
func (m *MilvusIndex) Search(vector []float32, topK int) ([]string, []float32, error) {
    sp, _ := entity.NewIndexIvfFlatSearchParam(64)
    
    results, err := m.client.Search(
        m.ctx, CollectionName, nil, "",
        []string{"photo_id", "file_path"},
        vector,
        "vector",
        entity.IP,
        topK,
        sp,
    )
    if err != nil {
        return nil, nil, fmt.Errorf("search: %w", err)
    }
    
    var photoIDs []string
    var scores []float32
    
    for _, result := range results {
        idField := result.Fields.GetColumn("photo_id").(*entity.ColumnVarChar)
        scoreField := result.Scores
        
        for i := 0; i < result.ResultCount; i++ {
            id, _ := idField.ValueByIdx(i)
            photoIDs = append(photoIDs, id)
            scores = append(scores, scoreField[i])
        }
    }
    
    return photoIDs, scores, nil
}

// DeletePhoto 删除照片
func (m *MilvusIndex) DeletePhoto(photoID string) error {
    expr := fmt.Sprintf("photo_id == '%s'", photoID)
    return m.client.Delete(m.ctx, CollectionName, expr)
}

// Close 关闭连接
func (m *MilvusIndex) Close() error {
    return m.client.Close()
}
```

---

### 2. 替换现有 VectorIndex

```go
// internal/vector_index.go (修改)
package internal

import (
    "fmt"
    "os"
    
    "photo-search-engine/internal/vector"
)

type VectorIndex struct {
    milvus *vector.MilvusIndex
    // 保留原有字段用于兼容
}

func NewVectorIndex(milvusAddr string) (*VectorIndex, error) {
    milvus, err := vector.NewMilvusIndex(milvusAddr)
    if err != nil {
        return nil, err
    }
    
    return &VectorIndex{
        milvus: milvus,
    }, nil
}

func (vi *VectorIndex) AddPhoto(photo Photo, vector []float32) error {
    return vi.milvus.AddPhoto(photo.ID, photo.FilePath, vector)
}

func (vi *VectorIndex) Search(queryVector []float32, limit int) ([]SearchResult, error) {
    photoIDs, scores, err := vi.milvus.Search(queryVector, limit)
    if err != nil {
        return nil, err
    }
    
    results := make([]SearchResult, len(photoIDs))
    for i, id := range photoIDs {
        results[i] = SearchResult{
            PhotoID: id,
            Score:   scores[i],
        }
    }
    
    return results, nil
}
```

---

### 3. 配置管理

```go
// internal/config/config.go (添加 Milvus 配置)
type Config struct {
    // ... 现有配置 ...
    
    Milvus struct {
        Enabled bool   `env:"MILVUS_ENABLED" default:"true"`
        Address string `env:"MILVUS_ADDRESS" default:"localhost:19530"`
    }
}
```

---

### 4. 启动脚本更新

```bash
# scripts/start.sh (添加 Milvus 启动)
#!/bin/bash

echo "🚀 启动照片搜索引擎..."

# 1. 启动 Milvus
echo "📦 启动 Milvus..."
cd deploy && docker-compose up -d

# 等待 Milvus 就绪
echo "⏳ 等待 Milvus 就绪..."
until curl -s http://localhost:9091/healthz > /dev/null; do
    sleep 1
done
echo "✅ Milvus 就绪"

# 2. 启动 ML Service
echo "🤖 启动 ML Service..."
cd ../ml-service && python app/main.py &
ML_PID=$!

# 3. 启动 Go 后端
echo "🔍 启动 Go 后端..."
cd .. && go run cmd/server/main.go &
GO_PID=$!

# 4. 启动前端
echo "🌐 启动前端..."
cd web && npm run dev &
WEB_PID=$!

echo "✅ 所有服务已启动"
echo "  Milvus: http://localhost:19530"
echo "  Attu: http://localhost:8000"
echo "  API: http://localhost:8080"
echo "  Web: http://localhost:5173"

# 统一关闭
trap "echo '正在关闭服务...'; kill $ML_PID $GO_PID $WEB_PID; cd deploy && docker-compose down; exit" SIGINT SIGTERM
wait
```

---

## 📊 性能对比

| 指标 | 暴力搜索 (当前) | Milvus (IVF) | Milvus (HNSW) |
|------|----------------|--------------|---------------|
| 1000 张搜索 | ~2000ms | ~50ms | ~20ms |
| 1万 张搜索 | ~20s | ~80ms | ~30ms |
| 10万 张搜索 | 不可用 | ~150ms | ~50ms |
| 内存占用 | O(N) | O(N) | O(N) |
| 索引构建 | 无 | 快 | 中等 |

---

## 🎯 实施计划

### 阶段 1: Milvus 部署（1 天）
- [ ] 创建 docker-compose.yml
- [ ] 测试 Milvus 启动和连接
- [ ] 验证 Attu 管理界面

### 阶段 2: SDK 集成（2 天）
- [ ] 封装 Milvus client
- [ ] 替换现有 VectorIndex
- [ ] 数据迁移（JSON → Milvus）
- [ ] 测试 CRUD 操作

### 阶段 3: 优化（1 天）
- [ ] 索引参数调优
- [ ] 批量插入优化
- [ ] 错误处理和重试
- [ ] 性能测试

**总计: 4 天**

---

## ✅ 验收标准

- [ ] 搜索延迟 < 100ms（1万张）
- [ ] 支持 10万+ 照片
- [ ] 重启不丢失索引
- [ ] 支持增量更新
- [ ] 可视化监控（Attu）

---

## 🚀 替代方案对比

| 方案 | 复杂度 | 性能 | 维护成本 | 推荐度 |
|------|--------|------|----------|--------|
| **Milvus** | 中 | 高 | 中 | ⭐⭐⭐⭐⭐ |
| FAISS (CGO) | 高 | 高 | 高 | ⭐⭐⭐ |
| 纯 Go HNSW | 低 | 中 | 低 | ⭐⭐⭐⭐ |
| SQLite VSS | 低 | 中 | 低 | ⭐⭐⭐ |

**推荐**: Milvus 适合生产环境，HNSW 适合简单场景。

---

*方案结束 - 等待主人确认*
