package indexer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"

	"photo-search-engine/internal"
)

// Index 向量索引管理器
type Index struct {
	mu          sync.RWMutex
	photos      []Photo
	mlClient    *internal.MLClient
	vectorIndex *internal.VectorIndex
	queryCache  *lru.Cache[string, []float32] // 查询向量缓存（LRU，限制1000条）
	resultCache *lru.Cache[string, []Photo]   // 搜索结果缓存（LRU，限制500条）
}

// Photo 照片信息
type Photo struct {
	ID          string   `json:"id"`
	Path        string   `json:"path"`
	Name        string   `json:"name"`
	Tags        []string `json:"tags"`
	DateTime    string   `json:"date_time"`
	Score       float32  `json:"score,omitempty"`
	Width       int      `json:"width,omitempty"`
	Height      int      `json:"height,omitempty"`
	CameraMake  string   `json:"camera_make,omitempty"`
	CameraModel string   `json:"camera_model,omitempty"`
}

// SearchResult 搜索结果
type SearchResult struct {
	PhotoID   string
	Score     float32
	PhotoPath string
}

// Stats 统计信息
type Stats struct {
	TotalPhotos  int
	TotalTags    int
	TotalSize    int64
	VectorImages int
	TagBreakdown map[string]int
}

// New 创建新的索引
func New() *Index {
	queryCache, _ := lru.New[string, []float32](1000)
	resultCache, _ := lru.New[string, []Photo](500)
	return &Index{
		photos:      make([]Photo, 0),
		queryCache:  queryCache,
		resultCache: resultCache,
	}
}

// NewWithML 创建带 ML 客户端的索引
func NewWithML(mlClient *internal.MLClient, vectorIndex *internal.VectorIndex) *Index {
	queryCache, _ := lru.New[string, []float32](1000)
	resultCache, _ := lru.New[string, []Photo](500)
	return &Index{
		photos:       make([]Photo, 0),
		mlClient:     mlClient,
		vectorIndex:  vectorIndex,
		queryCache:   queryCache,
		resultCache:  resultCache,
	}
}

// Search 搜索照片
func (idx *Index) Search(query interface{}, limit int) ([]Photo, error) {
	// 如果有向量索引和 ML 客户端，使用语义搜索
	if idx.mlClient != nil && idx.vectorIndex != nil && idx.vectorIndex.Size() > 0 {
		return idx.semanticSearch(query, limit)
	}

	// 否则使用简单的关键词匹配
	return idx.keywordSearch(query, limit)
}

// semanticSearch 语义搜索：编码查询文本，搜索向量索引
func (idx *Index) semanticSearch(query interface{}, limit int) ([]Photo, error) {
	queryStr, ok := query.(string)
	if !ok {
		return nil, fmt.Errorf("查询参数类型错误")
	}

	cacheKey := fmt.Sprintf("%s:%d", queryStr, limit)

	// 检查结果缓存
	idx.mu.RLock()
	if cached, found := idx.resultCache.Get(cacheKey); found {
		idx.mu.RUnlock()
		fmt.Printf("命中结果缓存: %s\n", queryStr)
		return cached, nil
	}
	idx.mu.RUnlock()

	fmt.Printf("语义搜索: %s\n", queryStr)

	// 检查缓存
	var queryEmbedding []float32
	if cached, found := idx.queryCache.Get(queryStr); found {
		queryEmbedding = cached
		fmt.Printf("使用缓存的查询向量\n")
	} else {
		// 编码查询文本
		var err error
		queryEmbedding, err = idx.mlClient.EncodeText(queryStr)
		if err != nil {
			fmt.Printf("文本编码失败，回退到关键词搜索: %v\n", err)
			return idx.keywordSearch(query, limit)
		}
		// 存入缓存
		idx.queryCache.Add(queryStr, queryEmbedding)
	}

	// 在向量索引中搜索
	results := idx.vectorIndex.SearchByEmbedding(queryEmbedding, limit)

	// 将搜索结果转换为 Photo 列表
	idx.mu.RLock()
	photos := make([]Photo, 0, len(results))
	for _, r := range results {
		photo := Photo{
			Path:  r.Path,
			Name:  filepath.Base(r.Path),
			Score: r.Score,
		}
		// 尝试从内存索引中获取更多信息
		for _, p := range idx.photos {
			if p.Path == r.Path {
				photo.ID = p.ID
				photo.Tags = p.Tags
				photo.DateTime = p.DateTime
				break
			}
		}
		if photo.ID == "" {
			photo.ID = r.Path
		}
		photos = append(photos, photo)
	}
	idx.mu.RUnlock()

	fmt.Printf("语义搜索完成，找到 %d 个结果\n", len(photos))

	// 缓存结果
	idx.mu.Lock()
	idx.resultCache.Add(cacheKey, photos)
	idx.mu.Unlock()

	return photos, nil
}

// keywordSearch 关键词搜索（回退方案）
func (idx *Index) keywordSearch(query interface{}, limit int) ([]Photo, error) {
	queryStr, ok := query.(string)
	if !ok {
		return nil, fmt.Errorf("查询参数类型错误")
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var results []Photo
	for _, p := range idx.photos {
		if len(results) >= limit {
			break
		}
		// 简单的关键词匹配
		if contains(p.Name, queryStr) || containsAny(p.Tags, queryStr) {
			p.Score = 0.5
			results = append(results, p)
		}
	}

	return results, nil
}

// Chat AI对话
func (idx *Index) Chat(message string) (string, error) {
	if idx.mlClient == nil {
		return "ML 服务未连接", nil
	}
	return idx.mlClient.Chat(message)
}

// ListPhotos 获取照片列表
func (idx *Index) ListPhotos(page, limit int) ([]Photo, int, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	fmt.Printf("ListPhotos: page=%d, limit=%d, total_photos=%d\n", page, limit, len(idx.photos))

	start := (page - 1) * limit
	end := start + limit

	if start >= len(idx.photos) {
		fmt.Printf("ListPhotos: start=%d >= len=%d, returning empty\n", start, len(idx.photos))
		return []Photo{}, len(idx.photos), nil
	}
	if end > len(idx.photos) {
		end = len(idx.photos)
	}

	result := idx.photos[start:end]
	fmt.Printf("ListPhotos: returning %d photos (start=%d, end=%d)\n", len(result), start, end)
	return result, len(idx.photos), nil
}

// GetPhoto 获取照片详情
func (idx *Index) GetPhoto(id string) (*Photo, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	for _, p := range idx.photos {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, nil
}

// GetStatsByDir 按目录获取统计信息
func (idx *Index) GetStatsByDir(dir string) Stats {
	return idx.GetStats(dir)
}

// GetStats 获取统计信息（兼容旧接口）
func (idx *Index) GetStats(dir ...string) Stats {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	filterDir := ""
	if len(dir) > 0 {
		filterDir = dir[0]
	}

	vectorCount := 0
	if idx.vectorIndex != nil {
		vectorCount = idx.vectorIndex.Size()
	}

	totalTags := 0
	var totalSize int64
	tagBreakdown := make(map[string]int)
	photoCount := 0
	for _, p := range idx.photos {
		if filterDir != "" && !strings.HasPrefix(p.Path, filterDir) {
			continue
		}
		photoCount++
		totalTags += len(p.Tags)
		for _, t := range p.Tags {
			tagBreakdown[t]++
		}
		if info, err := os.Stat(p.Path); err == nil {
			totalSize += info.Size()
		}
	}

	return Stats{
		TotalPhotos:  photoCount,
		TotalTags:    totalTags,
		TotalSize:    totalSize,
		VectorImages: vectorCount,
		TagBreakdown: tagBreakdown,
	}
}

// UpdatePhotos 更新照片列表（兼容旧接口）
func (idx *Index) UpdatePhotos(paths []string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	fmt.Printf("UpdatePhotos called with %d paths\n", len(paths))
	photos := make([]Photo, 0, len(paths))
	for i, path := range paths {
		photos = append(photos, Photo{
			ID:       fmt.Sprintf("photo-%d", i),
			Path:     path,
			Name:     filepath.Base(path),
			Tags:     []string{"未分类"},
			DateTime: "",
		})
	}
	idx.photos = photos
	fmt.Printf("Updated photos count: %d\n", len(idx.photos))
}

// UpdatePhotosMeta 更新带完整元数据的照片列表
func (idx *Index) UpdatePhotosMeta(photos []Photo) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	fmt.Printf("UpdatePhotosMeta called with %d photos\n", len(photos))
	idx.photos = photos
	fmt.Printf("Updated photos count: %d\n", len(idx.photos))
}

// BatchUpdateTagsByPaths 按路径批量更新照片标签（用于 ML 分类结果回写）
func (idx *Index) BatchUpdateTagsByPaths(updates map[string][]string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	updated := 0
	for i := range idx.photos {
		if tags, ok := updates[idx.photos[i].Path]; ok {
			idx.photos[i].Tags = tags
			updated++
		}
	}
	fmt.Printf("BatchUpdateTagsByPaths: updated %d photos\n", updated)
	idx.saveTagsToFile()
}

func (idx *Index) saveTagsToFile() {
	tagMap := make(map[string][]string)
	for _, p := range idx.photos {
		if len(p.Tags) > 0 && p.Tags[0] != "未分类" {
			tagMap[p.Path] = p.Tags
		}
	}
	data, err := json.Marshal(tagMap)
	if err != nil {
		fmt.Printf("saveTagsToFile marshal error: %v\n", err)
		return
	}
	if err := os.WriteFile("photo_tags.json", data, 0644); err != nil {
		fmt.Printf("saveTagsToFile write error: %v\n", err)
	}
}

// LoadTagsFromFile 从 photo_tags.json 恢复标签
func (idx *Index) LoadTagsFromFile() {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	data, err := os.ReadFile("photo_tags.json")
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("LoadTagsFromFile read error: %v\n", err)
		}
		return
	}
	var tagMap map[string][]string
	if err := json.Unmarshal(data, &tagMap); err != nil {
		fmt.Printf("LoadTagsFromFile unmarshal error: %v\n", err)
		return
	}
	updated := 0
	for i := range idx.photos {
		if tags, ok := tagMap[idx.photos[i].Path]; ok {
			idx.photos[i].Tags = tags
			updated++
		}
	}
	fmt.Printf("LoadTagsFromFile: restored tags for %d photos\n", updated)
}

// RebuildTags 根据照片路径重新生成标签（基于公共目录前缀）
func (idx *Index) RebuildTags() {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if len(idx.photos) == 0 {
		return
	}

	baseDir := filepath.Dir(idx.photos[0].Path)
	for _, p := range idx.photos[1:] {
		baseDir = commonPathPrefix(baseDir, filepath.Dir(p.Path))
	}

	updated := 0
	for i := range idx.photos {
		tags := extractPathTags(idx.photos[i].Path, baseDir)
		idx.photos[i].Tags = tags
		updated++
	}
	fmt.Printf("RebuildTags: updated %d photos\n", updated)
}

// UpdatePhotoDateTime 更新单张照片的日期时间
func (idx *Index) UpdatePhotoDateTime(path, dateTime string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	for i := range idx.photos {
		if idx.photos[i].Path == path {
			idx.photos[i].DateTime = dateTime
			break
		}
	}
}

// RebuildTagsForEmpty 只为没有标签的照片重新生成文件夹名标签
func (idx *Index) RebuildTagsForEmpty() {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if len(idx.photos) == 0 {
		return
	}

	baseDir := filepath.Dir(idx.photos[0].Path)
	for _, p := range idx.photos[1:] {
		baseDir = commonPathPrefix(baseDir, filepath.Dir(p.Path))
	}

	updated := 0
	for i := range idx.photos {
		if len(idx.photos[i].Tags) == 0 || (len(idx.photos[i].Tags) == 1 && idx.photos[i].Tags[0] == "未分类") {
			idx.photos[i].Tags = extractPathTags(idx.photos[i].Path, baseDir)
			updated++
		}
	}
	fmt.Printf("RebuildTagsForEmpty: updated %d photos\n", updated)
}

// ListDirs 获取所有已扫描的目录（从照片路径中提取）
func (idx *Index) ListDirs() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	dirSet := make(map[string]bool)
	for _, p := range idx.photos {
		dir := filepath.Dir(p.Path)
		dirSet[dir] = true
	}

	dirs := make([]string, 0, len(dirSet))
	for d := range dirSet {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	return dirs
}

// ListAllPhotos 返回所有照片的副本（只读安全）
func (idx *Index) ListAllPhotos() []Photo {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	result := make([]Photo, len(idx.photos))
	copy(result, idx.photos)
	return result
}

// ListPhotosByDir 按目录过滤获取照片列表
func (idx *Index) ListPhotosByDir(dir string, page, limit int) ([]Photo, int, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var filtered []Photo
	for _, p := range idx.photos {
		if strings.HasPrefix(p.Path, dir) {
			filtered = append(filtered, p)
		}
	}

	total := len(filtered)
	start := (page - 1) * limit
	end := start + limit

	if start >= total {
		return []Photo{}, total, nil
	}
	if end > total {
		end = total
	}

	return filtered[start:end], total, nil
}

// 辅助函数
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func containsAny(tags []string, query string) bool {
	for _, tag := range tags {
		if contains(tag, query) {
			return true
		}
	}
	return false
}

func extractPathTags(photoPath, baseDir string) []string {
	dir := filepath.Dir(photoPath)
	rel, err := filepath.Rel(baseDir, dir)
	if err != nil {
		return []string{"未分类"}
	}
	_ = rel
	// 只取直接父目录名作为标签（最直观）
	tag := filepath.Base(dir)
	if tag == "" || tag == "." || tag == string(filepath.Separator) {
		return []string{"未分类"}
	}
	return []string{tag}
}

func commonPathPrefix(a, b string) string {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	i := 0
	for i < minLen && a[i] == b[i] {
		i++
	}
	prefix := a[:i]
	lastSep := strings.LastIndex(prefix, string(filepath.Separator))
	if lastSep > 0 {
		prefix = prefix[:lastSep]
	}
	return prefix
}
