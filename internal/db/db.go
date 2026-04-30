package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"photo-search-engine/internal/indexer"
)

// DB SQLite数据库封装
type DB struct {
	conn *sql.DB
}

// New 创建并初始化数据库
func New(dbPath string) (*DB, error) {
	// 确保目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 设置连接池
	conn.SetMaxOpenConns(1) // SQLite 单写
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(time.Hour)

	db := &DB{conn: conn}
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("初始化表结构失败: %w", err)
	}

	return db, nil
}

// Close 关闭数据库连接
func (db *DB) Close() error {
	return db.conn.Close()
}

// initSchema 初始化数据库表结构
func (db *DB) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS photos (
    id TEXT PRIMARY KEY,
    path TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    date_time TEXT,
    width INTEGER,
    height INTEGER,
    camera_make TEXT,
    camera_model TEXT,
    thumbnail_path TEXT,
    face_count INTEGER DEFAULT 0,
    has_faces BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    photo_id TEXT NOT NULL,
    tag TEXT NOT NULL,
    UNIQUE(photo_id, tag),
    FOREIGN KEY (photo_id) REFERENCES photos(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tags_photo_id ON tags(photo_id);
CREATE INDEX IF NOT EXISTS idx_tags_tag ON tags(tag);

CREATE TABLE IF NOT EXISTS search_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    query TEXT NOT NULL,
    result_count INTEGER DEFAULT 0,
    filters TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_search_history_created ON search_history(created_at);

CREATE TABLE IF NOT EXISTS vector_index (
    path TEXT PRIMARY KEY,
    embedding BLOB NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS faces (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    photo_id TEXT NOT NULL,
    face_index INTEGER NOT NULL,
    person_id TEXT,
    location TEXT NOT NULL,
    encoding TEXT NOT NULL,
    confidence REAL DEFAULT 0.99,
    thumbnail_path TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(photo_id, face_index),
    FOREIGN KEY (photo_id) REFERENCES photos(id) ON DELETE CASCADE,
    FOREIGN KEY (person_id) REFERENCES persons(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_faces_photo_id ON faces(photo_id);
CREATE INDEX IF NOT EXISTS idx_faces_person_id ON faces(person_id);

CREATE TABLE IF NOT EXISTS persons (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    avatar TEXT,
    face_count INTEGER DEFAULT 0,
    photo_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`
	_, err := db.conn.Exec(schema)
	return err
}

// SavePhoto 保存或更新照片
func (db *DB) SavePhoto(photo *indexer.Photo) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 插入或更新照片
	_, err = tx.Exec(`
		INSERT INTO photos (id, path, name, date_time, width, height, camera_make, camera_model, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(path) DO UPDATE SET
			id = excluded.id,
			name = excluded.name,
			date_time = excluded.date_time,
			width = excluded.width,
			height = excluded.height,
			camera_make = excluded.camera_make,
			camera_model = excluded.camera_model,
			updated_at = CURRENT_TIMESTAMP
	`, photo.ID, photo.Path, photo.Name, photo.DateTime, photo.Width, photo.Height, photo.CameraMake, photo.CameraModel)
	if err != nil {
		return err
	}

	// 删除旧标签
	_, err = tx.Exec(`DELETE FROM tags WHERE photo_id = ?`, photo.ID)
	if err != nil {
		return err
	}

	// 插入新标签
	for _, tag := range photo.Tags {
		_, err = tx.Exec(`INSERT INTO tags (photo_id, tag) VALUES (?, ?)`, photo.ID, tag)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SavePhotos 批量保存照片
func (db *DB) SavePhotos(photos []indexer.Photo) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmtPhoto, err := tx.Prepare(`
		INSERT INTO photos (id, path, name, date_time, width, height, camera_make, camera_model, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(path) DO UPDATE SET
			id = excluded.id,
			name = excluded.name,
			date_time = excluded.date_time,
			width = excluded.width,
			height = excluded.height,
			camera_make = excluded.camera_make,
			camera_model = excluded.camera_model,
			updated_at = CURRENT_TIMESTAMP
	`)
	if err != nil {
		return err
	}
	defer stmtPhoto.Close()

	stmtDelTags, err := tx.Prepare(`DELETE FROM tags WHERE photo_id = ?`)
	if err != nil {
		return err
	}
	defer stmtDelTags.Close()

	stmtTag, err := tx.Prepare(`INSERT INTO tags (photo_id, tag) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	defer stmtTag.Close()

	for _, photo := range photos {
		_, err = stmtPhoto.Exec(photo.ID, photo.Path, photo.Name, photo.DateTime, photo.Width, photo.Height, photo.CameraMake, photo.CameraModel)
		if err != nil {
			return err
		}

		_, err = stmtDelTags.Exec(photo.ID)
		if err != nil {
			return err
		}

		for _, tag := range photo.Tags {
			_, err = stmtTag.Exec(photo.ID, tag)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// LoadPhotos 加载所有照片
func (db *DB) LoadPhotos() ([]indexer.Photo, error) {
	rows, err := db.conn.Query(`
		SELECT id, path, name, date_time, width, height, camera_make, camera_model
		FROM photos
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []indexer.Photo
	for rows.Next() {
		var p indexer.Photo
		err := rows.Scan(&p.ID, &p.Path, &p.Name, &p.DateTime, &p.Width, &p.Height, &p.CameraMake, &p.CameraModel)
		if err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}

	// 加载标签
	for i := range photos {
		tags, err := db.GetTagsByPhotoID(photos[i].ID)
		if err != nil {
			return nil, err
		}
		photos[i].Tags = tags
	}

	return photos, rows.Err()
}

// GetTagsByPhotoID 获取照片标签
func (db *DB) GetTagsByPhotoID(photoID string) ([]string, error) {
	rows, err := db.conn.Query(`SELECT tag FROM tags WHERE photo_id = ?`, photoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

// UpdatePhotoTags 更新照片标签
func (db *DB) UpdatePhotoTags(photoID string, tags []string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM tags WHERE photo_id = ?`, photoID)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		_, err = tx.Exec(`INSERT INTO tags (photo_id, tag) VALUES (?, ?)`, photoID, tag)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SaveSearchHistory 保存搜索历史
func (db *DB) SaveSearchHistory(query string, resultCount int, filters map[string]string) error {
	filtersJSON, _ := json.Marshal(filters)
	_, err := db.conn.Exec(
		`INSERT INTO search_history (query, result_count, filters) VALUES (?, ?, ?)`,
		query, resultCount, string(filtersJSON),
	)
	return err
}

// GetSearchHistory 获取搜索历史
func (db *DB) GetSearchHistory(limit int) ([]SearchHistoryItem, error) {
	rows, err := db.conn.Query(`
		SELECT id, query, result_count, filters, created_at
		FROM search_history
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []SearchHistoryItem
	for rows.Next() {
		var h SearchHistoryItem
		var filtersJSON string
		err := rows.Scan(&h.ID, &h.Query, &h.ResultCount, &filtersJSON, &h.CreatedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(filtersJSON), &h.Filters)
		history = append(history, h)
	}
	return history, rows.Err()
}

// SearchHistoryItem 搜索历史记录
 type SearchHistoryItem struct {
	ID          int               `json:"id"`
	Query       string            `json:"query"`
	ResultCount int               `json:"result_count"`
	Filters     map[string]string `json:"filters,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// SaveVector 保存向量
func (db *DB) SaveVector(path string, embedding []float32) error {
	data, err := json.Marshal(embedding)
	if err != nil {
		return err
	}
	_, err = db.conn.Exec(`
		INSERT INTO vector_index (path, embedding, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(path) DO UPDATE SET
			embedding = excluded.embedding,
			updated_at = CURRENT_TIMESTAMP
	`, path, data)
	return err
}

// LoadVectors 加载所有向量
func (db *DB) LoadVectors() (map[string][]float32, error) {
	rows, err := db.conn.Query(`SELECT path, embedding FROM vector_index`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	vectors := make(map[string][]float32)
	for rows.Next() {
		var path string
		var data []byte
		if err := rows.Scan(&path, &data); err != nil {
			return nil, err
		}
		var embedding []float32
		if err := json.Unmarshal(data, &embedding); err != nil {
			continue // 跳过损坏的数据
		}
		vectors[path] = embedding
	}
	return vectors, rows.Err()
}

// DeletePhoto 删除照片
func (db *DB) DeletePhoto(photoID string) error {
	_, err := db.conn.Exec(`DELETE FROM photos WHERE id = ?`, photoID)
	return err
}

// GetPhotoCount 获取照片总数
func (db *DB) GetPhotoCount() (int, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM photos`).Scan(&count)
	return count, err
}

// MigrateFromJSON 从 JSON 文件迁移数据到 SQLite
func (db *DB) MigrateFromJSON(photosPath, tagsPath string) error {
	// 迁移照片数据
	if _, err := os.Stat(photosPath); err == nil {
		data, err := os.ReadFile(photosPath)
		if err != nil {
			return fmt.Errorf("读取照片JSON失败: %w", err)
		}
		var photos []indexer.Photo
		if err := json.Unmarshal(data, &photos); err != nil {
			return fmt.Errorf("解析照片JSON失败: %w", err)
		}
		if err := db.SavePhotos(photos); err != nil {
			return fmt.Errorf("保存照片到数据库失败: %w", err)
		}
		fmt.Printf("已迁移 %d 张照片到 SQLite\n", len(photos))
	}

	// 迁移标签数据
	if _, err := os.Stat(tagsPath); err == nil {
		data, err := os.ReadFile(tagsPath)
		if err != nil {
			return fmt.Errorf("读取标签JSON失败: %w", err)
		}
		var tagMap map[string][]string
		if err := json.Unmarshal(data, &tagMap); err != nil {
			return fmt.Errorf("解析标签JSON失败: %w", err)
		}

		// 需要先加载照片以获取ID映射
		photos, err := db.LoadPhotos()
		if err != nil {
			return err
		}
		pathToID := make(map[string]string)
		for _, p := range photos {
			pathToID[p.Path] = p.ID
		}

		for path, tags := range tagMap {
			if photoID, ok := pathToID[path]; ok {
				if err := db.UpdatePhotoTags(photoID, tags); err != nil {
					fmt.Printf("迁移标签失败 %s: %v\n", path, err)
				}
			}
		}
		fmt.Printf("已迁移 %d 条标签记录到 SQLite\n", len(tagMap))
	}

	return nil
}

// ---- Face Recognition Methods ----

// FaceRecord 人脸记录
 type FaceRecord struct {
	ID           int       `json:"id"`
	PhotoID      string    `json:"photo_id"`
	FaceIndex    int       `json:"face_index"`
	PersonID     *string   `json:"person_id,omitempty"`
	Location     string    `json:"location"`
	Encoding     string    `json:"encoding"`
	Confidence   float64   `json:"confidence"`
	ThumbnailPath string `json:"thumbnail_path,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Person 人物记录
 type Person struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Avatar     string    `json:"avatar,omitempty"`
	FaceCount  int       `json:"face_count"`
	PhotoCount int       `json:"photo_count"`
	CreatedAt  time.Time `json:"created_at"`
}

// SaveFaces 保存人脸检测结果
func (db *DB) SaveFaces(photoID string, faces []FaceRecord) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 删除该照片的旧人脸记录
	_, err = tx.Exec(`DELETE FROM faces WHERE photo_id = ?`, photoID)
	if err != nil {
		return err
	}

	// 插入新人脸记录
	for _, face := range faces {
		_, err = tx.Exec(`
			INSERT INTO faces (photo_id, face_index, person_id, location, encoding, confidence, thumbnail_path, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		`, photoID, face.FaceIndex, face.PersonID, face.Location, face.Encoding, face.Confidence, face.ThumbnailPath)
		if err != nil {
			return err
		}
	}

	// 更新照片的人脸统计
	hasFaces := len(faces) > 0
	_, err = tx.Exec(`
		UPDATE photos SET face_count = ?, has_faces = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, len(faces), hasFaces, photoID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetTotalFacesCount 获取人脸总数
func (db *DB) GetTotalFacesCount() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM faces").Scan(&count)
	return count, err
}

// GetTotalPersonsCount 获取人物总数（有 person_id 的不同数量）
func (db *DB) GetTotalPersonsCount() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(DISTINCT person_id) FROM faces WHERE person_id IS NOT NULL").Scan(&count)
	return count, err
}

// GetFacesByPhotoID 获取照片的人脸列表
func (db *DB) GetFacesByPhotoID(photoID string) ([]FaceRecord, error) {
	rows, err := db.conn.Query(`
		SELECT id, photo_id, face_index, person_id, location, encoding, confidence, thumbnail_path, created_at, updated_at
		FROM faces WHERE photo_id = ? ORDER BY face_index
	`, photoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var faces []FaceRecord
	for rows.Next() {
		var f FaceRecord
		var personID sql.NullString
		var thumbPath sql.NullString
		err := rows.Scan(&f.ID, &f.PhotoID, &f.FaceIndex, &personID, &f.Location, &f.Encoding, &f.Confidence, &thumbPath, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if personID.Valid {
			f.PersonID = &personID.String
		}
		if thumbPath.Valid {
			f.ThumbnailPath = thumbPath.String
		}
		faces = append(faces, f)
	}
	return faces, rows.Err()
}

// GetFacesByPersonID 获取人物的所有人脸
func (db *DB) GetFacesByPersonID(personID string) ([]FaceRecord, error) {
	rows, err := db.conn.Query(`
		SELECT id, photo_id, face_index, person_id, location, encoding, confidence, thumbnail_path, created_at, updated_at
		FROM faces WHERE person_id = ? ORDER BY id
	`, personID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var faces []FaceRecord
	for rows.Next() {
		var f FaceRecord
		var pid sql.NullString
		var thumbPath sql.NullString
		err := rows.Scan(&f.ID, &f.PhotoID, &f.FaceIndex, &pid, &f.Location, &f.Encoding, &f.Confidence, &thumbPath, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if pid.Valid {
			f.PersonID = &pid.String
		}
		if thumbPath.Valid {
			f.ThumbnailPath = thumbPath.String
		}
		faces = append(faces, f)
	}
	return faces, rows.Err()
}

// UpdateFaceThumbnailPath 更新人脸缩略图路径
func (db *DB) UpdateFaceThumbnailPath(faceID int, path string) error {
	_, err := db.conn.Exec(`
		UPDATE faces SET thumbnail_path = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, path, faceID)
	return err
}
func (db *DB) GetFaceByID(faceID int) (*FaceRecord, error) {
	var f FaceRecord
	var personID sql.NullString
	var thumbPath sql.NullString
	err := db.conn.QueryRow(`
		SELECT id, photo_id, face_index, person_id, location, encoding, confidence, thumbnail_path, created_at, updated_at
		FROM faces WHERE id = ?
	`, faceID).Scan(&f.ID, &f.PhotoID, &f.FaceIndex, &personID, &f.Location, &f.Encoding, &f.Confidence, &thumbPath, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if personID.Valid {
		f.PersonID = &personID.String
	}
	if thumbPath.Valid {
		f.ThumbnailPath = thumbPath.String
	}
	return &f, nil
}

// UpdateFaceLabel 更新人脸标注
func (db *DB) UpdateFaceLabel(faceID int, personID string) error {
	_, err := db.conn.Exec(`
		UPDATE faces SET person_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, personID, faceID)
	return err
}

// CreatePerson 创建人物
func (db *DB) CreatePerson(person *Person) error {
	_, err := db.conn.Exec(`
		INSERT INTO persons (id, name, avatar, face_count, photo_count, created_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, person.ID, person.Name, person.Avatar, person.FaceCount, person.PhotoCount)
	return err
}

// GetPersonByID 获取人物详情
func (db *DB) GetPersonByID(personID string) (*Person, error) {
	var p Person
	err := db.conn.QueryRow(`
		SELECT id, name, avatar, face_count, photo_count, created_at
		FROM persons WHERE id = ?
	`, personID).Scan(&p.ID, &p.Name, &p.Avatar, &p.FaceCount, &p.PhotoCount, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetPersons 获取人物列表
func (db *DB) GetPersons(limit, offset int) ([]Person, int, error) {
	var total int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM persons`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := db.conn.Query(`
		SELECT id, name, avatar, face_count, photo_count, created_at
		FROM persons ORDER BY name LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var persons []Person
	for rows.Next() {
		var p Person
		err := rows.Scan(&p.ID, &p.Name, &p.Avatar, &p.FaceCount, &p.PhotoCount, &p.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		persons = append(persons, p)
	}
	return persons, total, rows.Err()
}

// UpdatePersonStats 更新人物统计
func (db *DB) UpdatePersonStats(personID string) error {
	_, err := db.conn.Exec(`
		UPDATE persons SET
			face_count = (SELECT COUNT(*) FROM faces WHERE person_id = ?),
			photo_count = (SELECT COUNT(DISTINCT photo_id) FROM faces WHERE person_id = ?)
		WHERE id = ?
	`, personID, personID, personID)
	return err
}

// UpdatePersonName 更新人物名称
func (db *DB) UpdatePersonName(personID, name string) error {
	_, err := db.conn.Exec(`
		UPDATE persons SET name = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, name, personID)
	return err
}
func (db *DB) SearchPhotosByPerson(personID string, limit, offset int) ([]indexer.Photo, int, error) {
	var total int
	err := db.conn.QueryRow(`
		SELECT COUNT(DISTINCT photo_id) FROM faces WHERE person_id = ?
	`, personID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := db.conn.Query(`
		SELECT DISTINCT p.id, p.path, COALESCE(p.name, ''), COALESCE(p.date_time, ''), COALESCE(p.width, 0), COALESCE(p.height, 0), COALESCE(p.camera_make, ''), COALESCE(p.camera_model, '')
		FROM photos p
		JOIN faces f ON (p.id = f.photo_id OR p.path = f.photo_id)
		WHERE f.person_id = ?
		ORDER BY p.date_time DESC
		LIMIT ? OFFSET ?
	`, personID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var photos []indexer.Photo
	for rows.Next() {
		var p indexer.Photo
		err := rows.Scan(&p.ID, &p.Path, &p.Name, &p.DateTime, &p.Width, &p.Height, &p.CameraMake, &p.CameraModel)
		if err != nil {
			return nil, 0, err
		}
		photos = append(photos, p)
	}

	// 加载标签
	for i := range photos {
		tags, err := db.GetTagsByPhotoID(photos[i].ID)
		if err == nil {
			photos[i].Tags = tags
		}
	}

	return photos, total, rows.Err()
}

// GetClusters 获取聚类后的人物列表
func (db *DB) GetClusters() (*sql.Rows, error) {
	return db.conn.Query(`
		SELECT 
			person_id,
			COUNT(*) as face_count,
			COUNT(DISTINCT photo_id) as photo_count,
			(
				SELECT id FROM faces f2
				WHERE f2.person_id = f1.person_id
				ORDER BY (
					CAST(json_extract(f2.location, '$.right') AS INT) - CAST(json_extract(f2.location, '$.left') AS INT)
				) * (
					CAST(json_extract(f2.location, '$.bottom') AS INT) - CAST(json_extract(f2.location, '$.top') AS INT)
				) DESC
				LIMIT 1
			) as sample_face_id
		FROM faces f1
		WHERE person_id IS NOT NULL
		GROUP BY person_id
		ORDER BY face_count DESC
	`)
}

// GetPhotoPathByID 通过ID获取照片路径
func (db *DB) GetPhotoPathByID(photoID string) (string, error) {
	var path string
	err := db.conn.QueryRow(`SELECT path FROM photos WHERE id = ?`, photoID).Scan(&path)
	return path, err
}
