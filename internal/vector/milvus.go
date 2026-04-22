package vector

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

const (
	CollectionName = "photos"
	Dim            = 512 // CLIP 向量维度
)

type MilvusIndex struct {
	client client.Client
	ctx    context.Context
}

func NewMilvusIndex(addr string) (*MilvusIndex, error) {
	ctx := context.Background()
	
	// 连接 Milvus
	c, err := client.NewClient(ctx, client.Config{
		Address: addr,
	})
	if err != nil {
		return nil, fmt.Errorf("连接 Milvus 失败: %w", err)
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
		return fmt.Errorf("检查集合失败: %w", err)
	}
	if has {
		// 加载集合
		return m.client.LoadCollection(m.ctx, CollectionName, false)
	}

	// 创建集合
	schema := entity.NewSchema().
		WithName(CollectionName).
		WithDescription("照片搜索向量").
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
		return fmt.Errorf("创建集合失败: %w", err)
	}

	// 创建 IVF_FLAT 索引
	idx, err := entity.NewIndexIvfFlat(entity.IP, 128)
	if err != nil {
		return fmt.Errorf("创建索引配置失败: %w", err)
	}

	if err := m.client.CreateIndex(m.ctx, CollectionName, "vector", idx, false); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	// 加载集合到内存
	if err := m.client.LoadCollection(m.ctx, CollectionName, false); err != nil {
		return fmt.Errorf("加载集合失败: %w", err)
	}

	return nil
}

// AddPhoto 添加照片向量
func (m *MilvusIndex) AddPhoto(photoID, filePath string, vector []float32) error {
	photoIDColumn := entity.NewColumnVarChar("photo_id", []string{photoID})
	pathColumn := entity.NewColumnVarChar("file_path", []string{filePath})
	vectorColumn := entity.NewColumnFloatVector("vector", Dim, [][]float32{vector})

	_, err := m.client.Insert(m.ctx, CollectionName, "", photoIDColumn, pathColumn, vectorColumn)
	if err != nil {
		return fmt.Errorf("插入向量失败: %w", err)
	}

	return nil
}

// AddPhotosBatch 批量添加照片向量
func (m *MilvusIndex) AddPhotosBatch(photoIDs, filePaths []string, vectors [][]float32) error {
	if len(photoIDs) != len(filePaths) || len(photoIDs) != len(vectors) {
		return fmt.Errorf("参数长度不匹配")
	}

	photoIDColumn := entity.NewColumnVarChar("photo_id", photoIDs)
	pathColumn := entity.NewColumnVarChar("file_path", filePaths)
	vectorColumn := entity.NewColumnFloatVector("vector", Dim, vectors)

	_, err := m.client.Insert(m.ctx, CollectionName, "", photoIDColumn, pathColumn, vectorColumn)
	if err != nil {
		return fmt.Errorf("批量插入失败: %w", err)
	}

	return nil
}

// Search 向量搜索
func (m *MilvusIndex) Search(vector []float32, topK int) ([]string, []string, []float32, error) {
	sp, err := entity.NewIndexIvfFlatSearchParam(64)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("创建搜索参数失败: %w", err)
	}

	// 将 []float32 转换为 []entity.Vector
	floatVector := entity.FloatVector(vector)
	vectors := []entity.Vector{floatVector}

	results, err := m.client.Search(
		m.ctx,
		CollectionName,
		nil,        // partition names
		"",         // expression
		[]string{"photo_id", "file_path"},
		vectors,
		"vector",
		entity.IP,  // Inner Product
		topK,
		sp,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("搜索失败: %w", err)
	}

	var photoIDs []string
	var filePaths []string
	var scores []float32

	for _, result := range results {
		idField := result.Fields.GetColumn("photo_id").(*entity.ColumnVarChar)
		pathField := result.Fields.GetColumn("file_path").(*entity.ColumnVarChar)

		for i := 0; i < result.ResultCount; i++ {
			id, _ := idField.ValueByIdx(i)
			path, _ := pathField.ValueByIdx(i)
			photoIDs = append(photoIDs, id)
			filePaths = append(filePaths, path)
			scores = append(scores, result.Scores[i])
		}
	}

	return photoIDs, filePaths, scores, nil
}

// DeletePhoto 删除照片
func (m *MilvusIndex) DeletePhoto(photoID string) error {
	expr := fmt.Sprintf("photo_id == '%s'", photoID)
	return m.client.Delete(m.ctx, CollectionName, "", expr)
}

// GetStats 获取统计信息
func (m *MilvusIndex) GetStats() (int64, error) {
	stats, err := m.client.GetCollectionStatistics(m.ctx, CollectionName)
	if err != nil {
		return 0, err
	}

	// stats 是 map[string]string，直接获取 row_count
	rowCountStr, ok := stats["row_count"]
	if !ok {
		return 0, fmt.Errorf("无法获取行数")
	}

	var rowCount int64
	_, err = fmt.Sscanf(rowCountStr, "%d", &rowCount)
	if err != nil {
		return 0, fmt.Errorf("解析行数失败: %w", err)
	}

	return rowCount, nil
}

// DropCollection 删除集合（危险操作）
func (m *MilvusIndex) DropCollection() error {
	return m.client.DropCollection(m.ctx, CollectionName)
}

// Close 关闭连接
func (m *MilvusIndex) Close() error {
	return m.client.Close()
}

// HealthCheck 健康检查
func (m *MilvusIndex) HealthCheck() error {
	// 简单搜索测试
	testVector := make([]float32, Dim)
	_, _, _, err := m.Search(testVector, 1)
	return err
}