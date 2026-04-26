package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rwcarlsen/goexif/exif"

	"photo-search-engine/internal"
	"photo-search-engine/internal/db"
	"photo-search-engine/internal/indexer"
)

var similarityLabels = loadSimilarityLabels()

func loadSimilarityLabels() []string {
	path := "config/label_space_v1.json"
	if envPath := os.Getenv("LABEL_SPACE_PATH"); envPath != "" {
		path = envPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{
			"城市街景", "古建筑", "现代建筑", "公园园林", "海滨沙滩",
			"山川湖泊", "室内场景", "夜景灯光", "人物合影", "美食餐饮",
		}
	}
	var space struct {
		Flat []string `json:"flat"`
	}
	if err := json.Unmarshal(data, &space); err != nil {
		return []string{"城市街景", "古建筑", "室内场景", "人物合影", "美食餐饮"}
	}
	if len(space.Flat) == 0 {
		return []string{"城市街景", "古建筑", "室内场景", "人物合影", "美食餐饮"}
	}
	return space.Flat
}

// Progress 扫描进度更新
type Progress struct {
	Scanned     int    `json:"scanned"`
	TotalFiles  int    `json:"total_files"`
	Extracted   int    `json:"extracted"`
	Skipped     int    `json:"skipped"`
	Failed      int    `json:"failed"`
	CurrentFile string `json:"current_file"`
	Message     string `json:"message"`
}

// ScanTask 扫描任务状态
type ScanTask struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	Directory   string    `json:"directory"`
	Scanned     int       `json:"scanned"`
	TotalFiles  int       `json:"total_files"`
	Extracted   int       `json:"extracted"`
	Skipped     int       `json:"skipped"`
	Failed      int       `json:"failed"`
	CurrentFile string    `json:"current_file"`
	Message     string    `json:"message"`
	Error       string    `json:"error,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Scanner 文件扫描器
type Scanner struct {
	imageExtensions map[string]bool
	indexer         *indexer.Index
	mlClient        *internal.MLClient
	vectorIndex     *internal.VectorIndex
	db              *db.DB
	tasks           map[string]*ScanTask
	tasksMu         sync.RWMutex
	taskCounter     int
	// 扫描期间保护索引的读写锁
	scanMu sync.RWMutex
}

// New 创建新的扫描器
func New() *Scanner {
	return &Scanner{
		imageExtensions: defaultImageExtensions(),
		indexer:         indexer.New(),
		tasks:           make(map[string]*ScanTask),
	}
}

// NewWithIndexer 创建带索引器的扫描器
func NewWithIndexer(idx *indexer.Index) *Scanner {
	return &Scanner{
		imageExtensions: defaultImageExtensions(),
		indexer:         idx,
		tasks:           make(map[string]*ScanTask),
	}
}

// NewWithML 创建带 ML 客户端和向量索引的扫描器
func NewWithML(idx *indexer.Index, mlClient *internal.MLClient, vectorIndex *internal.VectorIndex) *Scanner {
	return &Scanner{
		imageExtensions: defaultImageExtensions(),
		indexer:         idx,
		mlClient:        mlClient,
		vectorIndex:     vectorIndex,
		tasks:           make(map[string]*ScanTask),
	}
}

// NewWithDB 创建带数据库的扫描器
func NewWithDB(idx *indexer.Index, mlClient *internal.MLClient, vectorIndex *internal.VectorIndex, database *db.DB) *Scanner {
	return &Scanner{
		imageExtensions: defaultImageExtensions(),
		indexer:         idx,
		mlClient:        mlClient,
		vectorIndex:     vectorIndex,
		db:              database,
		tasks:           make(map[string]*ScanTask),
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

func (s *Scanner) newTaskID() string {
	s.taskCounter++
	return fmt.Sprintf("scan-%d-%d", time.Now().Unix(), s.taskCounter)
}

func (s *Scanner) updateTask(task *ScanTask) {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()
	task.UpdatedAt = time.Now()
	s.tasks[task.ID] = task
}

// GetTask 获取扫描任务状态
func (s *Scanner) GetTask(id string) *ScanTask {
	s.tasksMu.RLock()
	defer s.tasksMu.RUnlock()
	if t, ok := s.tasks[id]; ok {
		return t
	}
	return nil
}

// StartScan 启动异步扫描并返回任务 ID
func (s *Scanner) StartScan(dir string) string {
	task := &ScanTask{
		ID:        s.newTaskID(),
		Status:    "scanning",
		Directory: dir,
		Message:   "正在扫描目录...",
		UpdatedAt: time.Now(),
	}
	s.updateTask(task)

	go s.scanAsync(task)
	return task.ID
}

// StartScanWithContext 支持取消的扫描
func (s *Scanner) StartScanWithContext(ctx context.Context, dir string) string {
	task := &ScanTask{
		ID:        s.newTaskID(),
		Status:    "scanning",
		Directory: dir,
		Message:   "正在扫描目录...",
		UpdatedAt: time.Now(),
	}
	s.updateTask(task)

	go s.scanAsyncWithContext(ctx, task)
	return task.ID
}

var skipDirs = map[string]bool{
	"venv": true, "node_modules": true, ".git": true,
	"__pycache__": true, "dist": true, "build": true,
	".venv": true, "env": true,
}

var skipNamePrefixes = []string{
	"balloon-", "test_display_", "icon-", "logo", "favicon",
}

func shouldSkipFile(name string) bool {
	for _, prefix := range skipNamePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func (s *Scanner) scanAsync(task *ScanTask) {
	s.scanAsyncWithContext(context.Background(), task)
}

func (s *Scanner) scanAsyncWithContext(ctx context.Context, task *ScanTask) {
	fmt.Printf("Scanner.StartScan called with dir: %s, task=%s\n", task.Directory, task.ID)

	// 第一阶段：收集文件（可取消）
	var photos []indexer.Photo
	err := s.collectFiles(ctx, task, &photos)
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		task.Message = "扫描失败: " + err.Error()
		s.updateTask(task)
		return
	}

	// 检查是否已取消
	select {
	case <-ctx.Done():
		task.Status = "cancelled"
		task.Message = "扫描已取消"
		s.updateTask(task)
		return
	default:
	}

	task.TotalFiles = len(photos)
	task.Message = fmt.Sprintf("发现 %d 张照片，开始索引...", len(photos))
	s.updateTask(task)

	// 第二阶段：更新索引（使用写锁保护）
	s.scanMu.Lock()
	s.indexer.UpdatePhotosMeta(photos)
	s.scanMu.Unlock()

	// 保存到数据库
	if s.db != nil {
		if err := s.db.SavePhotos(photos); err != nil {
			fmt.Printf("保存照片到数据库失败: %v\n", err)
		}
	}

	// 检查是否已取消
	select {
	case <-ctx.Done():
		task.Status = "cancelled"
		task.Message = "扫描已取消"
		s.updateTask(task)
		return
	default:
	}

	// 第三阶段：提取特征
	if s.mlClient != nil && s.vectorIndex != nil && len(photos) > 0 {
		task.Status = "extracting"
		task.Message = "正在提取图像特征..."
		s.updateTask(task)

		imagePaths := make([]string, len(photos))
		for i, p := range photos {
			imagePaths[i] = p.Path
		}
		s.extractFeaturesWithContext(ctx, task, imagePaths, task.Directory)
	}

	// 检查是否已取消
	select {
	case <-ctx.Done():
		task.Status = "cancelled"
		task.Message = "扫描已取消"
		s.updateTask(task)
		return
	default:
	}

	// 第四阶段：人脸检测
	if s.mlClient != nil && s.db != nil && len(photos) > 0 {
		s.detectFacesInPhotos(ctx, task, photos)
	}

	task.Status = "completed"
	task.Message = fmt.Sprintf("扫描完成！共处理 %d 张照片", len(photos))
	task.CurrentFile = ""
	s.updateTask(task)
}

// collectFiles 收集文件（支持取消）
func (s *Scanner) collectFiles(ctx context.Context, task *ScanTask, photos *[]indexer.Photo) error {
	return filepath.Walk(task.Directory, func(path string, info os.FileInfo, err error) error {
		// 检查取消信号
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			fmt.Printf("Walk error: %v\n", err)
			return err
		}

		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		baseName := filepath.Base(path)
		if shouldSkipFile(baseName) {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if s.imageExtensions[ext] {
			dateStr, width, height, makeStr, modelStr := extractEXIF(path)
			photo := indexer.Photo{
				ID:          fmt.Sprintf("photo-%d", len(*photos)),
				Path:        path,
				Name:        baseName,
				Tags:        extractDirTags(path, task.Directory),
				DateTime:    dateStr,
				Width:       width,
				Height:      height,
				CameraMake:  makeStr,
				CameraModel: modelStr,
			}
			if photo.DateTime == "" {
				photo.DateTime = info.ModTime().Format("2006-01-02 15:04:05")
			}
			*photos = append(*photos, photo)
			task.Scanned++
			task.CurrentFile = baseName
			s.updateTask(task)
		}
		return nil
	})
}

// extractFeaturesWithContext 带取消和进度报告的特征提取
func (s *Scanner) extractFeaturesWithContext(ctx context.Context, task *ScanTask, images []string, baseDir string) {
	pendingBatch := make([]string, 0, 10)

	for _, imgPath := range images {
		// 检查取消信号
		select {
		case <-ctx.Done():
			return
		default:
		}

		task.CurrentFile = filepath.Base(imgPath)
		s.updateTask(task)

		if s.vectorIndex.HasImage(imgPath) {
			task.Skipped++
			continue
		}

		embedding, err := s.mlClient.ExtractImageEmbedding(imgPath)
		if err != nil {
			fmt.Printf("  ✗ 提取失败: %v\n", err)
			task.Failed++
			s.updateTask(task)
			continue
		}

		// 使用写锁保护向量索引
		s.scanMu.Lock()
		s.vectorIndex.Add(imgPath, embedding)
		s.scanMu.Unlock()

		// 保存到数据库
		if s.db != nil {
			if err := s.db.SaveVector(imgPath, embedding); err != nil {
				fmt.Printf("  ⚠ 保存向量到数据库失败: %v\n", err)
			}
		}

		task.Extracted++
		s.updateTask(task)

		pendingBatch = append(pendingBatch, imgPath)
		if len(pendingBatch) >= 10 {
			s.classifyAndUpdateTags(pendingBatch, baseDir)
			pendingBatch = pendingBatch[:0]
		}

		// 每处理 10 张保存一次索引
		if task.Extracted%10 == 0 {
			s.scanMu.RLock()
			if err := s.vectorIndex.Save(); err != nil {
				fmt.Printf("  ⚠ 保存索引失败: %v\n", err)
			}
			s.scanMu.RUnlock()
		}
	}

	if len(pendingBatch) > 0 {
		s.classifyAndUpdateTags(pendingBatch, baseDir)
	}

	s.scanMu.RLock()
	if err := s.vectorIndex.Save(); err != nil {
		fmt.Printf("⚠ 保存索引失败: %v\n", err)
	}
	s.scanMu.RUnlock()

	fmt.Printf("任务 %s 特征提取完成: 提取 %d, 跳过 %d, 失败 %d\n",
		task.ID, task.Extracted, task.Skipped, task.Failed)
}

// extractFeaturesWithTask 带进度更新的特征提取（兼容旧接口）
func (s *Scanner) extractFeaturesWithTask(task *ScanTask, images []string, baseDir string) {
	s.extractFeaturesWithContext(context.Background(), task, images, baseDir)
}

// detectFacesInPhotos 批量检测人脸并保存到数据库
func (s *Scanner) detectFacesInPhotos(ctx context.Context, task *ScanTask, photos []indexer.Photo) {
	task.Status = "detecting_faces"
	task.Message = "正在检测人脸..."
	s.updateTask(task)

	detected := 0
	for i, photo := range photos {
		select {
		case <-ctx.Done():
			return
		default:
		}

		task.CurrentFile = photo.Name
		s.updateTask(task)

		// 跳过已检测过的照片
		if s.db != nil {
			existing, _ := s.db.GetFacesByPhotoID(photo.ID)
			if len(existing) > 0 {
				continue
			}
		}

		result, err := s.mlClient.DetectFaces(photo.Path)
		if err != nil {
			continue
		}

		if result.FaceCount == 0 {
			continue
		}

		// 保存人脸数据
		if s.db != nil {
			var faceRecords []db.FaceRecord
			for _, f := range result.Faces {
				encodingJSON, _ := json.Marshal(f.Encoding)
				locationJSON, _ := json.Marshal(f.Location)
				faceRecords = append(faceRecords, db.FaceRecord{
					PhotoID:       photo.ID,
					FaceIndex:     f.Index,
					PersonID:      nil,
					Location:      string(locationJSON),
					Encoding:      string(encodingJSON),
					Confidence:    f.Confidence,
					ThumbnailPath: "",
				})
			}
			if err := s.db.SaveFaces(photo.ID, faceRecords); err != nil {
				fmt.Printf("  ⚠ 保存人脸数据失败 %s: %v\n", photo.Name, err)
			} else {
				detected++
			}
		}

		// 每 10 张更新一次进度
		if (i+1)%10 == 0 {
			task.Message = fmt.Sprintf("人脸检测中... (%d/%d)", i+1, len(photos))
			s.updateTask(task)
		}
	}

	if detected > 0 {
		fmt.Printf("人脸检测完成: %d 张照片检测到人脸\n", detected)
	}
}

func (s *Scanner) classifyAndUpdateTags(batch []string, baseDir string) {
	if s.mlClient == nil {
		return
	}
	results, err := s.mlClient.SimilarityBatch(batch, similarityLabels, 3)
	if err != nil {
		fmt.Printf("  ⚠ 批量相似度计算失败: %v\n", err)
		return
	}

	updates := make(map[string][]string)
	for i, result := range results {
		path := batch[i]
		folderTag := extractDirTags(path, baseDir)
		if len(folderTag) == 0 {
			folderTag = []string{"未分类"}
		}

		mlTags := make([]string, 0, 3)
		for i, item := range result {
			if i >= 3 {
				break
			}
			mlTags = append(mlTags, item.Label)
		}

		tagSet := make(map[string]bool)
		var merged []string
		for _, t := range folderTag {
			if !tagSet[t] {
				tagSet[t] = true
				merged = append(merged, t)
			}
		}
		for _, t := range mlTags {
			if !tagSet[t] {
				tagSet[t] = true
				merged = append(merged, t)
			}
		}
		updates[path] = merged
	}

	// 使用写锁保护索引更新
	s.scanMu.Lock()
	s.indexer.BatchUpdateTagsByPaths(updates)
	s.scanMu.Unlock()

	// 保存标签到数据库
	if s.db != nil {
		for path, tags := range updates {
			// 查找照片ID
			photos := s.indexer.ListAllPhotos()
			for _, p := range photos {
				if p.Path == path {
					if err := s.db.UpdatePhotoTags(p.ID, tags); err != nil {
						fmt.Printf("保存标签到数据库失败 %s: %v\n", path, err)
					}
					break
				}
			}
		}
	}
}

// ExtractEXIFDate 仅提取照片的 EXIF 日期时间
func ExtractEXIFDate(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return ""
	}

	if tm, err := x.DateTime(); err == nil {
		return tm.Format("2006-01-02 15:04:05")
	}
	return ""
}

func extractEXIF(path string) (dateStr string, width, height int, makeStr, modelStr string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return
	}

	if tm, err := x.DateTime(); err == nil {
		dateStr = tm.Format("2006-01-02 15:04:05")
	}
	if tag, err := x.Get(exif.Make); err == nil {
		makeStr, _ = tag.StringVal()
		makeStr = strings.TrimSpace(makeStr)
	}
	if tag, err := x.Get(exif.Model); err == nil {
		modelStr, _ = tag.StringVal()
		modelStr = strings.TrimSpace(modelStr)
	}
	if tag, err := x.Get(exif.PixelXDimension); err == nil {
		if d, err := tag.Int(0); err == nil {
			width = int(d)
		}
	}
	if tag, err := x.Get(exif.PixelYDimension); err == nil {
		if d, err := tag.Int(0); err == nil {
			height = int(d)
		}
	}
	return
}

func extractDirTags(photoPath, baseDir string) []string {
	dir := filepath.Dir(photoPath)
	_, err := filepath.Rel(baseDir, dir)
	if err != nil {
		return []string{"未分类"}
	}
	// 只取直接父目录名作为标签（最直观）
	tag := filepath.Base(dir)
	if tag == "" || tag == "." || tag == string(filepath.Separator) {
		return []string{"未分类"}
	}
	return []string{tag}
}

// Scan 同步扫描（保留兼容）
func (s *Scanner) Scan(dir string) ([]string, error) {
	taskID := s.StartScan(dir)
	// 轮询等待完成
	for {
		task := s.GetTask(taskID)
		if task == nil {
			return nil, fmt.Errorf("task not found")
		}
		if task.Status == "completed" || task.Status == "failed" || task.Status == "cancelled" {
			// 从索引中重新获取照片列表
			return nil, nil // 兼容旧接口，实际调用方不依赖返回值了
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// ScanDir 扫描目录（别名）
func (s *Scanner) ScanDir(dir string) (int, error) {
	taskID := s.StartScan(dir)
	for {
		task := s.GetTask(taskID)
		if task == nil {
			return 0, fmt.Errorf("task not found")
		}
		if task.Status == "completed" {
			return task.TotalFiles, nil
		}
		if task.Status == "failed" {
			return 0, fmt.Errorf(task.Error)
		}
		if task.Status == "cancelled" {
			return 0, fmt.Errorf("scan cancelled")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// IsScanning 检查是否有扫描任务在进行中
func (s *Scanner) IsScanning() bool {
	s.tasksMu.RLock()
	defer s.tasksMu.RUnlock()
	for _, task := range s.tasks {
		if task.Status == "scanning" || task.Status == "extracting" {
			return true
		}
	}
	return false
}

// GetProgressChannel 返回进度 channel（用于实时接收进度更新）
func (s *Scanner) GetProgressChannel(taskID string, interval time.Duration) <-chan Progress {
	ch := make(chan Progress, 10)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			task := s.GetTask(taskID)
			if task == nil {
				return
			}
			ch <- Progress{
				Scanned:     task.Scanned,
				TotalFiles:  task.TotalFiles,
				Extracted:   task.Extracted,
				Skipped:     task.Skipped,
				Failed:      task.Failed,
				CurrentFile: task.CurrentFile,
				Message:     task.Message,
			}
			if task.Status == "completed" || task.Status == "failed" || task.Status == "cancelled" {
				return
			}
			<-ticker.C
		}
	}()
	return ch
}
