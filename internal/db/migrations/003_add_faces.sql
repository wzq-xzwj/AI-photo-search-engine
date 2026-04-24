-- 人脸表
CREATE TABLE IF NOT EXISTS faces (
    id TEXT PRIMARY KEY,                 -- UUID
    photo_id TEXT NOT NULL,              -- 关联照片ID
    face_index INTEGER NOT NULL,         -- 照片中第几个人脸
    person_id TEXT,                      -- 关联的人物ID（标注后）
    
    -- 人脸位置 (JSON: {"top": 100, "right": 200, "bottom": 300, "left": 50})
    location TEXT NOT NULL,
    
    -- 人脸特征向量 (128维，JSON数组)
    encoding TEXT NOT NULL,
    
    -- 人脸缩略图 (base64 或文件路径)
    thumbnail TEXT,
    
    -- 检测置信度
    confidence REAL DEFAULT 0.99,
    
    -- 时间戳
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    -- 唯一约束
    UNIQUE(photo_id, face_index)
);

-- 人物表
CREATE TABLE IF NOT EXISTS persons (
    id TEXT PRIMARY KEY,                 -- UUID
    name TEXT,                           -- 人物名称（可为空，未标注时）
    avatar TEXT,                         -- 头像照片路径
    face_count INTEGER DEFAULT 0,        -- 关联人脸数量
    photo_count INTEGER DEFAULT 0,       -- 关联照片数量
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 照片表添加人脸相关字段（如果表已存在）
ALTER TABLE photos ADD COLUMN face_count INTEGER DEFAULT 0;
ALTER TABLE photos ADD COLUMN has_faces BOOLEAN DEFAULT FALSE;

-- 索引
CREATE INDEX IF NOT EXISTS idx_faces_photo_id ON faces(photo_id);
CREATE INDEX IF NOT EXISTS idx_faces_person_id ON faces(person_id);
CREATE INDEX IF NOT EXISTS idx_persons_name ON persons(name);

-- 触发器：更新人物统计
CREATE TRIGGER IF NOT EXISTS update_person_stats AFTER UPDATE OF person_id ON faces
BEGIN
    -- 更新原人物的统计
    UPDATE persons SET 
        face_count = (SELECT COUNT(*) FROM faces WHERE person_id = OLD.person_id),
        photo_count = (SELECT COUNT(DISTINCT photo_id) FROM faces WHERE person_id = OLD.person_id)
    WHERE id = OLD.person_id;
    
    -- 更新新人物的统计
    UPDATE persons SET 
        face_count = (SELECT COUNT(*) FROM faces WHERE person_id = NEW.person_id),
        photo_count = (SELECT COUNT(DISTINCT photo_id) FROM faces WHERE person_id = NEW.person_id)
    WHERE id = NEW.person_id;
END;
