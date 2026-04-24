package face

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Face 人脸信息
type Face struct {
	ID         string    `json:"id"`
	PhotoID    string    `json:"photo_id"`
	FaceIndex  int       `json:"face_index"`
	PersonID   *string   `json:"person_id,omitempty"`
	PersonName *string   `json:"person_name,omitempty"`
	Location   Location  `json:"location"`
	Thumbnail  string    `json:"thumbnail,omitempty"`
	Confidence float64   `json:"confidence"`
	CreatedAt  time.Time `json:"created_at"`
}

// Location 人脸位置
type Location struct {
	Top    int `json:"top"`
	Right  int `json:"right"`
	Bottom int `json:"bottom"`
	Left   int `json:"left"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Person 人物信息
type Person struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Avatar     string    `json:"avatar,omitempty"`
	FaceCount  int       `json:"face_count"`
	PhotoCount int       `json:"photo_count"`
	CreatedAt  time.Time `json:"created_at"`
}

// FaceService 人脸服务
type FaceService struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewFaceService 创建人脸服务
func NewFaceService(db *sql.DB, logger *zap.Logger) *FaceService {
	return &FaceService{
		db:     db,
		logger: logger,
	}
}

// DetectFaces 检测人脸
func (s *FaceService) DetectFaces(photoID string, imagePath string) ([]Face, error) {
	// TODO: 调用 Python 服务检测人脸
	// 这里先返回模拟数据
	return nil, fmt.Errorf("not implemented")
}

// GetFacesByPhoto 获取照片的人脸列表
func (s *FaceService) GetFacesByPhoto(photoID string) ([]Face, error) {
	query := `
		SELECT f.id, f.photo_id, f.face_index, f.person_id, p.name,
		       f.location, f.thumbnail, f.confidence, f.created_at
		FROM faces f
		LEFT JOIN persons p ON f.person_id = p.id
		WHERE f.photo_id = ?
		ORDER BY f.face_index
	`

	rows, err := s.db.Query(query, photoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var faces []Face
	for rows.Next() {
		var face Face
		var locationJSON string
		var personName sql.NullString

		err := rows.Scan(
			&face.ID, &face.PhotoID, &face.FaceIndex,
			&face.PersonID, &personName,
			&locationJSON, &face.Thumbnail,
			&face.Confidence, &face.CreatedAt,
		)
		if err != nil {
			s.logger.Error("scan face failed", zap.Error(err))
			continue
		}

		if personName.Valid {
			face.PersonName = &personName.String
		}

		// 解析位置 JSON
		json.Unmarshal([]byte(locationJSON), &face.Location)

		faces = append(faces, face)
	}

	return faces, nil
}

// LabelFace 标注人脸
func (s *FaceService) LabelFace(faceID string, personName string) (*Person, error) {
	// 查找或创建人物
	personID, err := s.findOrCreatePerson(personName)
	if err != nil {
		return nil, err
	}

	// 更新人脸的人物关联
	query := `UPDATE faces SET person_id = ?, updated_at = ? WHERE id = ?`
	_, err = s.db.Exec(query, personID, time.Now(), faceID)
	if err != nil {
		return nil, err
	}

	// 返回人物信息
	return s.GetPerson(personID)
}

// findOrCreatePerson 查找或创建人物
func (s *FaceService) findOrCreatePerson(name string) (string, error) {
	// 先查找
	var id string
	query := `SELECT id FROM persons WHERE name = ?`
	err := s.db.QueryRow(query, name).Scan(&id)
	if err == nil {
		return id, nil
	}

	// 创建新人物
	id = uuid.New().String()
	query = `INSERT INTO persons (id, name) VALUES (?, ?)`
	_, err = s.db.Exec(query, id, name)
	if err != nil {
		return "", err
	}

	return id, nil
}

// GetPerson 获取人物信息
func (s *FaceService) GetPerson(personID string) (*Person, error) {
	query := `
		SELECT id, name, avatar, face_count, photo_count, created_at
		FROM persons
		WHERE id = ?
	`

	var person Person
	err := s.db.QueryRow(query, personID).Scan(
		&person.ID, &person.Name, &person.Avatar,
		&person.FaceCount, &person.PhotoCount, &person.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &person, nil
}

// GetPersons 获取人物列表
func (s *FaceService) GetPersons(limit, offset int) ([]Person, int, error) {
	// 获取总数
	var total int
	countQuery := `SELECT COUNT(*) FROM persons`
	if err := s.db.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 获取列表
	query := `
		SELECT id, name, avatar, face_count, photo_count, created_at
		FROM persons
		ORDER BY face_count DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var persons []Person
	for rows.Next() {
		var person Person
		err := rows.Scan(
			&person.ID, &person.Name, &person.Avatar,
			&person.FaceCount, &person.PhotoCount, &person.CreatedAt,
		)
		if err != nil {
			continue
		}
		persons = append(persons, person)
	}

	return persons, total, nil
}

// SearchPhotosByPerson 按人物搜索照片
func (s *FaceService) SearchPhotosByPerson(personID string, limit, offset int) ([]string, int, error) {
	// 获取总数
	var total int
	countQuery := `
		SELECT COUNT(DISTINCT photo_id) 
		FROM faces 
		WHERE person_id = ?
	`
	if err := s.db.QueryRow(countQuery, personID).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 获取照片ID列表
	query := `
		SELECT DISTINCT photo_id
		FROM faces
		WHERE person_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, personID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var photoIDs []string
	for rows.Next() {
		var photoID string
		if err := rows.Scan(&photoID); err != nil {
			continue
		}
		photoIDs = append(photoIDs, photoID)
	}

	return photoIDs, total, nil
}

// FaceHandler API处理器
type FaceHandler struct {
	service *FaceService
	logger  *zap.Logger
}

// NewFaceHandler 创建处理器
func NewFaceHandler(service *FaceService, logger *zap.Logger) *FaceHandler {
	return &FaceHandler{
		service: service,
		logger:  logger,
	}
}

// HandleDetectFaces 处理人脸检测请求
func (h *FaceHandler) HandleDetectFaces(c *gin.Context) {
	photoID := c.Param("photo_id")
	if photoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "photo_id is required"})
		return
	}

	// TODO: 获取照片路径并调用检测
	faces, err := h.service.DetectFaces(photoID, "")
	if err != nil {
		h.logger.Error("detect faces failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": faces,
	})
}

// HandleGetFaces 获取照片的人脸列表
func (h *FaceHandler) HandleGetFaces(c *gin.Context) {
	photoID := c.Param("photo_id")
	if photoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "photo_id is required"})
		return
	}

	faces, err := h.service.GetFacesByPhoto(photoID)
	if err != nil {
		h.logger.Error("get faces failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": faces,
	})
}

// HandleLabelFace 标注人脸
func (h *FaceHandler) HandleLabelFace(c *gin.Context) {
	faceID := c.Param("face_id")
	if faceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "face_id is required"})
		return
	}

	var req struct {
		PersonName string `json:"person_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	person, err := h.service.LabelFace(faceID, req.PersonName)
	if err != nil {
		h.logger.Error("label face failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": person,
	})
}

// HandleGetPersons 获取人物列表
func (h *FaceHandler) HandleGetPersons(c *gin.Context) {
	limit := 50
	offset := 0

	persons, total, err := h.service.GetPersons(limit, offset)
	if err != nil {
		h.logger.Error("get persons failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":  persons,
			"total": total,
		},
	})
}

// HandleSearchByPerson 按人物搜索照片
func (h *FaceHandler) HandleSearchByPerson(c *gin.Context) {
	personID := c.Query("person_id")
	if personID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "person_id is required"})
		return
	}

	limit := 20
	offset := 0

	photoIDs, total, err := h.service.SearchPhotosByPerson(personID, limit, offset)
	if err != nil {
		h.logger.Error("search by person failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"photo_ids": photoIDs,
			"total":     total,
		},
	})
}
