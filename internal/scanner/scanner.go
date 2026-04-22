package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"photo-search-engine/internal"
	"photo-search-engine/internal/indexer"
)

// Scanner 文件扫描器
type Scanner struct {
	imageExtensions map[string]bool
	indexer         *indexer.Index
	mlClient        *internal.MLClient
	vectorIndex     *internal.VectorIndex
}

// New 创建新的扫描器
func New() *Scanner {
	return &Scanner{
		imageExtensions: defaultImageExtensions(),
		indexer:         indexer.New(),
	}
}

// NewWithIndexer 创建带索引器的扫描器
func NewWithIndexer(idx *indexer.Index) *Scanner {
	return &Scanner{
		imageExtensions: defaultImageExtensions(),
		indexer:         idx,
	}
}

// NewWithML 创建带 ML 客户端和向量索引的扫描器
func NewWithML(idx *indexer.Index, mlClient *internal.MLClient, vectorIndex *internal.VectorIndex) *Scanner {
	return &Scanner{
		imageExtensions: defaultImageExtensions(),
		indexer:         idx,
		mlClient:        mlClient,
		vectorIndex:     vectorIndex,
	}
}

func defaultImageExtensions() map[string]bool {
	return map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".bmp":  true,
		".tiff": true,
		".tif":  true,
		".webp": true,
		".heic": true,
		".heif": true,
		".raw":  true,
		".cr2":  true,
		".nef":  true,
		".arw":  true,
		".dng":  true,
	}
}

// Scan 扫描目录中的图片文件
func (s *Scanner) Scan(dir string) ([]string, error) {
	fmt.Printf("Scanner.Scan called with dir: %s\n", dir)
	var images []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Walk error: %v\n", err)
			return err
		}

		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if s.imageExtensions[ext] {
				images = append(images, path)
			}
		}

		return nil
	})

	fmt.Printf("Walk completed, found %d images, err: %v\n", len(images), err)

	// 更新索引
	if err == nil {
		fmt.Printf("Calling UpdatePhotos with %d images\n", len(images))
		s.indexer.UpdatePhotos(images)
	} else {
		fmt.Printf("Not calling UpdatePhotos because err: %v\n", err)
	}

	// 如果有 ML 客户端和向量索引，提取特征
	if s.mlClient != nil && s.vectorIndex != nil && err == nil {
		fmt.Printf("开始提取图像特征...\n")
		s.extractFeatures(images)
	}

	return images, err
}

// extractFeatures 对图片列表提取特征并存入向量索引
func (s *Scanner) extractFeatures(images []string) {
	extracted := 0
	skipped := 0
	failed := 0

	for i, imgPath := range images {
		// 跳过已提取的图片
		if s.vectorIndex.HasImage(imgPath) {
			skipped++
			continue
		}

		fmt.Printf("[%d/%d] 提取特征: %s\n", i+1, len(images), filepath.Base(imgPath))

		embedding, err := s.mlClient.ExtractImageEmbedding(imgPath)
		if err != nil {
			fmt.Printf("  ✗ 提取失败: %v\n", err)
			failed++
			continue
		}

		s.vectorIndex.Add(imgPath, embedding)
		extracted++

		// 每处理 10 张图片保存一次索引
		if extracted%10 == 0 {
			if err := s.vectorIndex.Save(); err != nil {
				fmt.Printf("  ⚠ 保存索引失败: %v\n", err)
			}
		}
	}

	// 最终保存索引
	if err := s.vectorIndex.Save(); err != nil {
		fmt.Printf("⚠ 保存索引失败: %v\n", err)
	}

	fmt.Printf("特征提取完成: 提取 %d, 跳过 %d, 失败 %d, 索引总数 %d\n",
		extracted, skipped, failed, s.vectorIndex.Size())
}

// ScanDir 扫描目录（别名）
func (s *Scanner) ScanDir(dir string) (int, error) {
	images, err := s.Scan(dir)
	return len(images), err
}
