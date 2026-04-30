package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"photo-search-engine/internal/db"
	"photo-search-engine/internal/indexer"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// FaceHandler 人脸相关处理器
type FaceHandler struct {
	indexer  *indexer.Index
	db       *db.DB
	mlClient *MLFaceClient
	logger   *zap.Logger
}

// MLFaceClient ML服务的人脸检测客户端
type MLFaceClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewFaceHandler 创建人脸处理器
func NewFaceHandler(idx *indexer.Index, database *db.DB, mlServiceURL string, logger *zap.Logger) *FaceHandler {
	if mlServiceURL == "" {
		mlServiceURL = "http://127.0.0.1:8000"
	}
	return &FaceHandler{
		indexer: idx,
		db:      database,
		mlClient: &MLFaceClient{
			baseURL:    mlServiceURL,
			httpClient: &http.Client{Timeout: 30 * time.Second},
		},
		logger: logger,
	}
}

// FaceDetectRequest 人脸检测请求
type FaceDetectRequest struct {
	PhotoID string `json:"photo_id" binding:"required"`
}

// FaceLocation 人脸位置
type FaceLocation struct {
	Top    int `json:"top"`
	Right  int `json:"right"`
	Bottom int `json:"bottom"`
	Left   int `json:"left"`
}

// FaceInfo 人脸信息
type FaceInfo struct {
	ID           int           `json:"id"`
	PhotoID      string        `json:"photo_id"`
	FaceIndex    int           `json:"face_index"`
	PersonID     *string       `json:"person_id,omitempty"`
	PersonName   string        `json:"person_name,omitempty"`
	Location     FaceLocation  `json:"location"`
	Confidence   float64       `json:"confidence"`
	ThumbnailURL string        `json:"thumbnail_url,omitempty"`
}

// FaceDetectResponse 人脸检测响应
type FaceDetectResponse struct {
	PhotoID    string     `json:"photo_id"`
	FaceCount  int        `json:"face_count"`
	Faces      []FaceInfo `json:"faces"`
}

// HandleDetectFaces 检测照片中的人脸
func (h *FaceHandler) HandleDetectFaces(c *gin.Context) {
	var req FaceDetectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 获取照片路径
	photoPath, err := h.db.GetPhotoPathByID(req.PhotoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	// 调用 ML 服务检测人脸
	faces, err := h.mlClient.DetectFaces(photoPath)
	if err != nil {
		h.logger.Warn("Face detection failed", zap.Error(err), zap.String("photo_id", req.PhotoID))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Face detection service unavailable: " + err.Error()})
		return
	}

	// 保存缩略图
	thumbnailsDir := "./data/faces"
	os.MkdirAll(thumbnailsDir, 0755)

	var faceRecords []db.FaceRecord
	var faceInfos []FaceInfo

	for i, face := range faces {
		// 提取缩略图
		thumbnailPath := filepath.Join(thumbnailsDir, fmt.Sprintf("%s_%d.jpg", req.PhotoID, i))
		thumbnailData, err := h.mlClient.ExtractThumbnail(photoPath, face.Location)
		if err == nil {
			os.WriteFile(thumbnailPath, thumbnailData, 0644)
		}

		locationJSON, _ := json.Marshal(face.Location)
		encodingJSON, _ := json.Marshal(face.Encoding)

		faceRecords = append(faceRecords, db.FaceRecord{
			PhotoID:       req.PhotoID,
			FaceIndex:     face.Index,
			Location:      string(locationJSON),
			Encoding:      string(encodingJSON),
			Confidence:    face.Confidence,
			ThumbnailPath: thumbnailPath,
		})

		faceInfos = append(faceInfos, FaceInfo{
			PhotoID:      req.PhotoID,
			FaceIndex:    face.Index,
			Location:     face.Location,
			Confidence:   face.Confidence,
			ThumbnailURL: "/api/v1/faces/thumbnail?photo_id=" + req.PhotoID + "&index=" + strconv.Itoa(face.Index),
		})
	}

	// 保存到数据库
	if h.db != nil {
		if err := h.db.SaveFaces(req.PhotoID, faceRecords); err != nil {
			h.logger.Error("Failed to save faces", zap.Error(err))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": FaceDetectResponse{
			PhotoID:   req.PhotoID,
			FaceCount: len(faces),
			Faces:     faceInfos,
		},
	})
}

// HandleLabelFace 标注人脸
type LabelFaceRequest struct {
	PersonName string `json:"person_name"` // 新人物名称
	PersonID   string `json:"person_id"`   // 或选择现有人物
}

func (h *FaceHandler) HandleLabelFace(c *gin.Context) {
	faceIDStr := c.Param("id")
	faceID, err := strconv.Atoi(faceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid face ID"})
		return
	}

	var req LabelFaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	var personID string

	if req.PersonID != "" {
		// 使用现有人物
		person, err := h.db.GetPersonByID(req.PersonID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Person not found"})
			return
		}
		personID = person.ID
	} else if req.PersonName != "" {
		// 创建新人物
		personID = uuid.New().String()
		person := &db.Person{
			ID:   personID,
			Name: req.PersonName,
		}
		if err := h.db.CreatePerson(person); err != nil {
			h.logger.Error("Failed to create person", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create person"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "person_name or person_id required"})
		return
	}

	// 更新人脸标注
	if err := h.db.UpdateFaceLabel(faceID, personID); err != nil {
		h.logger.Error("Failed to label face", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to label face"})
		return
	}

	// 更新人物统计
	if err := h.db.UpdatePersonStats(personID); err != nil {
		h.logger.Warn("Failed to update person stats", zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"face_id":    faceID,
			"person_id":  personID,
			"person_name": req.PersonName,
		},
	})
}

// HandleGetPersons 获取人物列表
func (h *FaceHandler) HandleGetPersons(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	persons, total, err := h.db.GetPersons(limit, offset)
	if err != nil {
		h.logger.Error("Failed to get persons", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get persons"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"total":   total,
			"persons": persons,
		},
	})
}

// HandleSearchByPerson 按人物搜索照片
func (h *FaceHandler) HandleSearchByPerson(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	personID := c.Query("person_id")
	if personID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "person_id required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	photos, total, err := h.db.SearchPhotosByPerson(personID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to search by person", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"total":  total,
			"photos": photos,
		},
	})
}

// HandleGetFaceThumbnail 获取人脸缩略图（从原图实时裁剪）
func (h *FaceHandler) HandleGetFaceThumbnail(c *gin.Context) {
	// 优先用 face_id
	faceIDStr := c.Query("face_id")
	var face *db.FaceRecord

	if faceIDStr != "" {
		fid, err := strconv.Atoi(faceIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid face_id"})
			return
		}
		face, err = h.db.GetFaceByID(fid)
		if err != nil {
			h.logger.Error("GetFaceByID failed", zap.Error(err), zap.Int("face_id", fid))
			c.JSON(http.StatusNotFound, gin.H{"error": "Face not found", "detail": err.Error()})
			return
		}
		if face == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Face is nil"})
			return
		}
	} else {
		// 兼容旧接口 photo_id + index
		photoID := c.Query("photo_id")
		indexStr := c.Query("index")
		if photoID == "" || indexStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "face_id or photo_id+index required"})
			return
		}
		index, _ := strconv.Atoi(indexStr)
		faces, err := h.db.GetFacesByPhotoID(photoID)
		if err != nil || len(faces) <= index {
			c.JSON(http.StatusNotFound, gin.H{"error": "Face not found"})
			return
		}
		face = &faces[index]
	}

	// 如果有预存缩略图，直接返回
	if face.ThumbnailPath != "" {
		if data, err := os.ReadFile(face.ThumbnailPath); err == nil {
			c.Data(http.StatusOK, "image/jpeg", data)
			return
		}
	}

	// 否则从原图裁剪
	// photo_id 在 faces 表中可能是文件路径或 "photo-XX" 格式 ID
	photoPath := face.PhotoID
	if _, err := os.Stat(photoPath); os.IsNotExist(err) {
		// 尝试从 photos 表查找
		var lookupErr error
		photoPath, lookupErr = h.db.GetPhotoPathByID(face.PhotoID)
		if lookupErr != nil || photoPath == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found", "photo_id": face.PhotoID})
			return
		}
	}

	// 解析人脸位置
	var loc struct {
		Top    int `json:"top"`
		Right  int `json:"right"`
		Bottom int `json:"bottom"`
		Left   int `json:"left"`
	}
	if face.Location != "" {
		json.Unmarshal([]byte(face.Location), &loc)
	}

	// 打开原图
	f, err := os.Open(photoPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo file not found"})
		return
	}
	defer f.Close()

	var img image.Image
	ext := strings.ToLower(filepath.Ext(photoPath))
	if ext == ".png" {
		img, err = png.Decode(f)
	} else {
		img, err = jpeg.Decode(f)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode image"})
		return
	}

	bounds := img.Bounds()
	imgW := bounds.Max.X
	imgH := bounds.Max.Y

	// 添加 padding（20%）
	faceW := loc.Right - loc.Left
	faceH := loc.Bottom - loc.Top
	if faceW <= 0 {
		faceW = 100
	}
	if faceH <= 0 {
		faceH = 100
	}
	padX := faceW / 5
	padY := faceH / 5

	x0 := max(0, loc.Left-padX)
	y0 := max(0, loc.Top-padY)
	x1 := min(imgW, loc.Right+padX)
	y1 := min(imgH, loc.Bottom+padY)

	// 裁剪
	cropper, ok := img.(interface {
		SubImage(r image.Rectangle) image.Image
	})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Image cropping not supported"})
		return
	}
	cropped := cropper.SubImage(image.Rect(x0, y0, x1, y1))

	// 编码为 JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, cropped, &jpeg.Options{Quality: 85}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode thumbnail"})
		return
	}

	// 缓存到磁盘
	thumbDir := "./data/faces"
	os.MkdirAll(thumbDir, 0755)
	thumbPath := filepath.Join(thumbDir, fmt.Sprintf("%d.jpg", face.ID))
	os.WriteFile(thumbPath, buf.Bytes(), 0644)

	// 更新数据库
	h.db.UpdateFaceThumbnailPath(face.ID, thumbPath)

	c.Data(http.StatusOK, "image/jpeg", buf.Bytes())
}

// HandleGetFaceStats 获取人脸统计
func (h *FaceHandler) HandleGetFaceStats(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	totalFaces, err := h.db.GetTotalFacesCount()
	if err != nil {
		h.logger.Error("Failed to get faces count", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	totalPersons, err := h.db.GetTotalPersonsCount()
	if err != nil {
		h.logger.Error("Failed to get persons count", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"total_faces":   totalFaces,
			"total_persons": totalPersons,
		},
	})
}

// HandleGetClusters 获取聚类后的人物列表
func (h *FaceHandler) HandleGetClusters(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	// 查询聚类结果
	rows, err := h.db.GetClusters()
	if err != nil {
		h.logger.Error("Failed to get clusters", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get clusters"})
		return
	}
	defer rows.Close()

	type ClusterInfo struct {
		PersonID   string `json:"person_id"`
		FaceCount  int    `json:"face_count"`
		PhotoCount int    `json:"photo_count"`
		SampleFace *struct {
			ID         int    `json:"id"`
			PhotoID    string `json:"photo_id"`
			FaceIndex  int    `json:"face_index"`
			ThumbnailURL string `json:"thumbnail_url"`
		} `json:"sample_face,omitempty"`
		Name string `json:"name"`
	}

	// 先收集所有行数据
	type clusterRow struct {
		PersonID     string
		FaceCount    int
		PhotoCount   int
		SampleFaceID int
	}
	var rows2 []clusterRow
	for rows.Next() {
		var r clusterRow
		if err := rows.Scan(&r.PersonID, &r.FaceCount, &r.PhotoCount, &r.SampleFaceID); err != nil {
			continue
		}
		rows2 = append(rows2, r)
	}
	rows.Close()

	// 再查每行的样本人脸和人物名称
	var clusters []ClusterInfo
	for _, r := range rows2 {
		cluster := ClusterInfo{
			PersonID:   r.PersonID,
			FaceCount:  r.FaceCount,
			PhotoCount: r.PhotoCount,
			Name:       "未知人物",
		}

		face, err := h.db.GetFaceByID(r.SampleFaceID)
		if err == nil && face != nil {
			cluster.SampleFace = &struct {
				ID           int    `json:"id"`
				PhotoID      string `json:"photo_id"`
				FaceIndex    int    `json:"face_index"`
				ThumbnailURL string `json:"thumbnail_url"`
			}{
				ID:         face.ID,
				PhotoID:    face.PhotoID,
				FaceIndex:  face.FaceIndex,
				ThumbnailURL: "/api/v1/faces/thumbnail?photo_id=" + face.PhotoID + "&index=" + strconv.Itoa(face.FaceIndex),
			}
		}

		person, _ := h.db.GetPersonByID(r.PersonID)
		if person != nil && person.Name != "" {
			cluster.Name = person.Name
		}

		clusters = append(clusters, cluster)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"clusters": clusters,
			"total":    len(clusters),
		},
	})
}

// HandleLabelCluster 标注整个人物聚类
func (h *FaceHandler) HandleLabelCluster(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	var req struct {
		PersonID   string `json:"person_id" binding:"required"`
		PersonName string `json:"person_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// 创建或更新人物
	person, err := h.db.GetPersonByID(req.PersonID)
	if err != nil || person == nil {
		// 创建新人物
		person = &db.Person{
			ID:   req.PersonID,
			Name: req.PersonName,
		}
		if err := h.db.CreatePerson(person); err != nil {
			h.logger.Error("Failed to create person", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create person"})
			return
		}
	} else {
		// 更新名称
		if err := h.db.UpdatePersonName(req.PersonID, req.PersonName); err != nil {
			h.logger.Error("Failed to update person name", zap.Error(err))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"person_id":   req.PersonID,
			"person_name": req.PersonName,
		},
	})
}

// MLFaceInfo ML服务返回的人脸信息
type MLFaceInfo struct {
	Index      int          `json:"index"`
	Location   FaceLocation `json:"location"`
	Encoding   []float64    `json:"encoding"`
	Confidence float64      `json:"confidence"`
}

// DetectFaces 调用 ML 服务检测人脸
func (c *MLFaceClient) DetectFaces(imagePath string) ([]MLFaceInfo, error) {
	reqBody, _ := json.Marshal(map[string]string{"image_path": imagePath})
	url := fmt.Sprintf("%s/api/v1/faces/detect", c.baseURL)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ML service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ML service error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ImagePath  string       `json:"image_path"`
		FaceCount  int          `json:"face_count"`
		Faces      []MLFaceInfo `json:"faces"`
		Available  bool         `json:"available"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result.Faces, nil
}

// ExtractThumbnail 调用 ML 服务提取缩略图
func (c *MLFaceClient) ExtractThumbnail(imagePath string, location FaceLocation) ([]byte, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"image_path": imagePath,
		"location":   location,
		"size":       150,
	})
	url := fmt.Sprintf("%s/api/v1/faces/thumbnail", c.baseURL)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("thumbnail extraction failed: %s", string(body))
	}

	return io.ReadAll(resp.Body)
}
