package indexer

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"

	"go.uber.org/zap"

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
	logger      *zap.Logger
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
	Embedding   []float32 `json:"embedding,omitempty"`
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
		logger:      zap.NewNop(),
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
		logger:       zap.NewNop(),
	}
}

// SetLogger 设置 logger
func (idx *Index) SetLogger(l *zap.Logger) {
	idx.logger = l
}

// Search 搜索照片
func (idx *Index) Search(query interface{}, limit int) ([]Photo, error) {
	// 如果有向量索引和 ML 客户端，使用语义搜索
	if idx.mlClient != nil && idx.vectorIndex != nil && idx.vectorIndex.Size() > 0 {
		return idx.semanticSearch(query, limit)
	}

	// 否则使用内存余弦相似度回退（从 SQLite 加载的 embedding）
	return idx.bruteForceSearchFallback(query, limit)
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
		idx.logger.Debug("命中结果缓存", zap.String("query", queryStr))
		return cached, nil
	}
	idx.mu.RUnlock()

	idx.logger.Info("语义搜索", zap.String("query", queryStr))

	// 检查缓存
	var queryEmbedding []float32
	if cached, found := idx.queryCache.Get(queryStr); found {
		queryEmbedding = cached
		idx.logger.Debug("使用缓存的查询向量", zap.String("query", queryStr))
	} else {
		// 编码查询文本
		var err error
		queryEmbedding, err = idx.mlClient.EncodeText(queryStr)
		if err != nil {
			idx.logger.Error("文本编码失败，回退到暴力搜索", zap.Error(err))
			return idx.bruteForceSearchFallback(query, limit)
		}
		// 存入缓存
		idx.queryCache.Add(queryStr, queryEmbedding)
	}

	// 在向量索引中搜索
	results := idx.vectorIndex.SearchByEmbedding(queryEmbedding, limit)

	// 过滤低相似度结果（最低 0.35 分）
	minScore := float32(0.35)
	n := 0
	for _, r := range results {
		if r.Score >= minScore {
			results[n] = r
			n++
		}
	}
	results = results[:n]

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

	idx.logger.Info("语义搜索完成", zap.Int("results", len(photos)))

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

// bruteForceSearchFallback 暴力余弦相似度搜索（Milvus 不可用时的回退）
// 从 SQLite 加载的 embedding 已填充到 idx.photos 中
func (idx *Index) bruteForceSearchFallback(query interface{}, limit int) ([]Photo, error) {
	queryStr, ok := query.(string)
	if !ok {
		return nil, fmt.Errorf("查询参数类型错误")
	}

	if idx.mlClient == nil {
		return nil, fmt.Errorf("ML 客户端不可用，无法进行向量搜索")
	}

	// 编码查询文本
	queryEmbedding, err := idx.mlClient.EncodeText(queryStr)
	if err != nil {
		return nil, fmt.Errorf("文本编码失败: %w", err)
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// 收集所有带有 embedding 的照片
	var candidates []Photo
	for _, p := range idx.photos {
		if len(p.Embedding) > 0 {
			candidates = append(candidates, p)
		}
	}

	if len(candidates) == 0 {
		idx.logger.Warn("没有照片包含 embedding，无法进行暴力搜索")
		return []Photo{}, nil
	}

	// 计算余弦相似度并排序
	type scoredPhoto struct {
		photo Photo
		score float32
	}
	scored := make([]scoredPhoto, 0, len(candidates))
	for _, p := range candidates {
		score := cosineSimilarity(queryEmbedding, p.Embedding)
		if score >= 0.35 { // 最低相似度阈值
			scored = append(scored, scoredPhoto{photo: p, score: score})
		}
	}

	// 按相似度降序排序
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// 取 top-k
	if len(scored) > limit {
		scored = scored[:limit]
	}

	results := make([]Photo, 0, len(scored))
	for _, s := range scored {
		s.photo.Score = s.score
		results = append(results, s.photo)
	}

	idx.logger.Info("暴力向量搜索完成", zap.Int("candidates", len(candidates)), zap.Int("results", len(results)))
	return results, nil
}

// cosineSimilarity 计算两个 float32 向量的余弦相似度
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
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

	idx.logger.Info("ListPhotos", zap.Int("page", page), zap.Int("limit", limit), zap.Int("total_photos", len(idx.photos)))

	start := (page - 1) * limit
	end := start + limit

	if start >= len(idx.photos) {
		idx.logger.Info("ListPhotos: start >= len, returning empty", zap.Int("start", start), zap.Int("len", len(idx.photos)))
		return []Photo{}, len(idx.photos), nil
	}
	if end > len(idx.photos) {
		end = len(idx.photos)
	}

	result := idx.photos[start:end]
	idx.logger.Info("ListPhotos: returning photos", zap.Int("count", len(result)), zap.Int("start", start), zap.Int("end", end))
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

	idx.logger.Info("UpdatePhotos called", zap.Int("paths", len(paths)))
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
	idx.logger.Info("Updated photos count", zap.Int("count", len(idx.photos)))
}

// UpdatePhotosMeta 更新带完整元数据的照片列表
func (idx *Index) UpdatePhotosMeta(photos []Photo) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.logger.Info("UpdatePhotosMeta called", zap.Int("photos", len(photos)))
	idx.photos = photos
	idx.logger.Info("Updated photos count", zap.Int("count", len(idx.photos)))
}

// MergePhotos 合并照片到现有索引（不替换已有照片，更新或追加新照片）
func (idx *Index) MergePhotos(newPhotos []Photo) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// 创建路径到照片的映射
	existingMap := make(map[string]int) // path -> index
	for i, p := range idx.photos {
		existingMap[p.Path] = i
	}

	merged := 0
	added := 0
	for _, p := range newPhotos {
		if i, ok := existingMap[p.Path]; ok {
			// 更新已有照片
			idx.photos[i] = p
			merged++
		} else {
			// 追加新照片
			idx.photos = append(idx.photos, p)
			added++
		}
	}
	idx.logger.Info("MergePhotos", zap.Int("merged", merged), zap.Int("added", added), zap.Int("total", len(idx.photos)))
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
	idx.logger.Info("BatchUpdateTagsByPaths", zap.Int("updated", updated))
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
		idx.logger.Error("saveTagsToFile marshal error", zap.Error(err))
		return
	}
	if err := os.WriteFile("photo_tags.json", data, 0644); err != nil {
		idx.logger.Error("saveTagsToFile write error", zap.Error(err))
	}
}

// LoadTagsFromFile 从 photo_tags.json 恢复标签
func (idx *Index) LoadTagsFromFile() {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	data, err := os.ReadFile("photo_tags.json")
	if err != nil {
		if !os.IsNotExist(err) {
			idx.logger.Error("LoadTagsFromFile read error", zap.Error(err))
		}
		return
	}
	var tagMap map[string][]string
	if err := json.Unmarshal(data, &tagMap); err != nil {
		idx.logger.Error("LoadTagsFromFile unmarshal error", zap.Error(err))
		return
	}
	updated := 0
	for i := range idx.photos {
		if tags, ok := tagMap[idx.photos[i].Path]; ok {
			idx.photos[i].Tags = tags
			updated++
		}
	}
	idx.logger.Info("LoadTagsFromFile", zap.Int("restored", updated))
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
	idx.logger.Info("RebuildTags", zap.Int("updated", updated))
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
	idx.logger.Info("RebuildTagsForEmpty", zap.Int("updated", updated))
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
