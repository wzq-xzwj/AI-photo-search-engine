package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"photo-search-engine/internal/vector"
)

func main() {
	// 连接 Milvus
	milvus, err := vector.NewMilvusIndex("localhost:19530")
	if err != nil {
		fmt.Printf("❌ 连接 Milvus 失败: %v\n", err)
		os.Exit(1)
	}
	defer milvus.Close()

	// 获取当前记录数
	count, _ := milvus.GetStats()
	fmt.Printf("📊 当前 Milvus 记录数: %d\n", count)

	// 删除旧集合
	fmt.Println("🗑️  清空 Milvus 集合...")
	if err := milvus.DropCollection(); err != nil {
		fmt.Printf("❌ 删除集合失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ 旧集合已删除")

	// 重新创建（NewMilvusIndex 会自动创建）
	fmt.Println("🔨 重新创建集合...")
	milvus2, err := vector.NewMilvusIndex("localhost:19530")
	if err != nil {
		fmt.Printf("❌ 重建集合失败: %v\n", err)
		os.Exit(1)
	}
	defer milvus2.Close()

	// 从 vector_index.json 读取已有索引
	fmt.Println("📥 从本地索引重建...")
	data, err := os.ReadFile("vector_index.json")
	if err != nil {
		fmt.Printf("❌ 读取索引失败: %v\n", err)
		os.Exit(1)
	}

	var entries []struct {
		Path      string    `json:"path"`
		Embedding []float32 `json:"embedding"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		fmt.Printf("❌ 解析索引失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📋 本地索引: %d 张照片\n", len(entries))

	// 批量插入
	batchSize := 50
	total := 0
	for i := 0; i < len(entries); i += batchSize {
		end := i + batchSize
		if end > len(entries) {
			end = len(entries)
		}
		batch := entries[i:end]

		ids := make([]string, len(batch))
		paths := make([]string, len(batch))
		vectors := make([][]float32, len(batch))

		for j, e := range batch {
			ids[j] = e.Path // 用路径做 ID
			paths[j] = e.Path
			vectors[j] = e.Embedding
		}

		if err := milvus2.AddPhotosBatch(ids, paths, vectors); err != nil {
			fmt.Printf("⚠️  批量插入失败 (%d-%d): %v\n", i, end, err)
			continue
		}
		total += len(batch)
		if total%200 == 0 {
			fmt.Printf("   进度: %d/%d\n", total, len(entries))
		}
	}

	// Flush 确保数据落盘
	milvus2.Flush()

	newCount, _ := milvus2.GetStats()
	fmt.Printf("\n🎉 完成！重建 %d 条记录，Milvus 当前: %d\n", total, newCount)

	// 检查重复
	paths := make(map[string]int)
	for _, e := range entries {
		paths[e.Path]++
	}
	dupes := 0
	for _, c := range paths {
		if c > 1 {
			dupes++
		}
	}
	if dupes > 0 {
		fmt.Printf("⚠️  发现 %d 个重复路径（本地索引）\n", dupes)
	} else {
		fmt.Println("✅ 无重复")
	}

	// 清理缩略图缓存
	facesDir := "./data/faces"
	if entries, err := os.ReadDir(facesDir); err == nil {
		fmt.Printf("🗑️  清理 %d 个缩略图缓存...\n", len(entries))
		for _, e := range entries {
			os.Remove(filepath.Join(facesDir, e.Name()))
		}
	}
}
