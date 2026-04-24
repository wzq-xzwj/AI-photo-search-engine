# 照片搜索引擎 - 人脸识别功能计划

> 版本: 1.0
> 日期: 2026-04-24
> 项目: /Users/wuzhaoqing/Pictures/photo-search-engine

---

## 1. 功能概述

为照片搜索引擎添加人脸识别功能，实现：
- **人脸检测**: 自动检测照片中的人脸
- **人脸展示**: 前端展示检测到的人脸，支持用户标注
- **人脸搜索**: 按人物搜索照片
- **人脸分组**: 自动聚类相似人脸

---

## 2. 技术架构

### 2.1 后端架构

```
┌─────────────────────────────────────────┐
│           Go 后端引擎                    │
│  ┌─────────────┐  ┌─────────────────┐  │
│  │ 照片扫描器   │  │ 人脸检测服务     │  │
│  │ (scanner)   │  │ (face_detection)│  │
│  └──────┬──────┘  └────────┬────────┘  │
│         │                  │            │
│  ┌──────┴──────────────────┴────────┐   │
│  │         Milvus 向量数据库         │   │
│  │  (照片向量 + 人脸向量)            │   │
│  └──────────────────────────────────┘   │
│         │
│  ┌──────┴──────────────────┐
│  │      SQLite 元数据       │
│  │  (照片信息 + 人脸信息)    │
│  └─────────────────────────┘
└─────────────────────────────────────────┘
```

### 2.2 前端架构

```
┌─────────────────────────────────────────┐
│           React 前端                      │
│  ┌─────────────┐  ┌─────────────────┐  │
│  │ 照片查看器   │  │ 人脸标记面板     │  │
│  │ (PhotoView) │  │ (FacePanel)     │  │
│  └──────┬──────┘  └────────┬────────┘  │
│         │                  │            │
│  ┌──────┴──────────────────┴────────┐   │
│  │      人脸搜索界面 (FaceSearch)    │   │
│  └──────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

---

## 3. 数据模型

### 3.1 人脸表 (faces)

```sql
CREATE TABLE faces (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    photo_id TEXT NOT NULL,              -- 关联照片ID
    face_index INTEGER NOT NULL,         -- 照片中第几个人脸
    person_id TEXT,                      -- 关联的人物ID（标注后）
    
    -- 人脸位置 (JSON: {"top": 100, "right": 200, "bottom": 300, "left": 50})
    location TEXT NOT NULL,
    
    -- 人脸特征向量 (128维，JSON数组)
    encoding TEXT NOT NULL,
    
    -- 检测置信度
    confidence REAL DEFAULT 0.99,
    
    -- 时间戳
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    -- 索引
    UNIQUE(photo_id, face_index)
);

-- 索引
CREATE INDEX idx_faces_photo_id ON faces(photo_id);
CREATE INDEX idx_faces_person_id ON faces(person_id);
```

### 3.2 人物表 (persons)

```sql
CREATE TABLE persons (
    id TEXT PRIMARY KEY,                 -- UUID
    name TEXT NOT NULL,                  -- 人物名称
    avatar TEXT,                         -- 头像照片路径
    face_count INTEGER DEFAULT 0,        -- 关联人脸数量
    photo_count INTEGER DEFAULT 0,       -- 关联照片数量
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 3.3 照片表扩展 (photos)

```sql
-- 在现有 photos 表添加人脸相关字段
ALTER TABLE photos ADD COLUMN face_count INTEGER DEFAULT 0;
ALTER TABLE photos ADD COLUMN has_faces BOOLEAN DEFAULT FALSE;
```

---

## 4. API 设计

### 4.1 人脸检测 API

```http
POST /api/v1/faces/detect
Content-Type: application/json

{
    "photo_id": "photo_uuid"
}

Response:
{
    "code": 200,
    "data": {
        "photo_id": "photo_uuid",
        "face_count": 3,
        "faces": [
            {
                "id": "face_uuid_1",
                "index": 0,
                "location": {"top": 100, "right": 200, "bottom": 300, "left": 50},
                "confidence": 0.999,
                "person_id": null,  // 未标注
                "thumbnail": "/api/v1/faces/face_uuid_1/thumbnail"
            }
        ]
    }
}
```

### 4.2 人脸标注 API

```http
PUT /api/v1/faces/{face_id}/label
Content-Type: application/json

{
    "person_name": "张三",      // 新人物名称
    "person_id": "existing_id"  // 或选择现有人物
}

Response:
{
    "code": 200,
    "data": {
        "face_id": "face_uuid_1",
        "person_id": "person_uuid",
        "person_name": "张三"
    }
}
```

### 4.3 按人物搜索照片 API

```http
GET /api/v1/search/person?person_id=xxx&limit=20

Response:
{
    "code": 200,
    "data": {
        "total": 50,
        "photos": [
            {
                "id": "photo_uuid",
                "path": "/path/to/photo.jpg",
                "thumbnail": "...",
                "face_location": {"top": 100, "right": 200, "bottom": 300, "left": 50}
            }
        ]
    }
}
```

### 4.4 获取人物列表 API

```http
GET /api/v1/persons?limit=50

Response:
{
    "code": 200,
    "data": {
        "total": 10,
        "persons": [
            {
                "id": "person_uuid",
                "name": "张三",
                "avatar": "/path/to/avatar.jpg",
                "face_count": 15,
                "photo_count": 8
            }
        ]
    }
}
```

---

## 5. 前端设计

### 5.1 人脸标记面板 (FacePanel)

```tsx
interface FacePanelProps {
    photoId: string;
    faces: Face[];
    onLabel: (faceId: string, personName: string) => void;
}

// 功能：
// 1. 展示检测到的人脸缩略图
// 2. 支持点击选择/框选人脸
// 3. 输入框标注人物名称
// 4. 显示已标注的人物列表
```

### 5.2 照片查看器增强 (PhotoView)

```tsx
interface PhotoViewProps {
    photo: Photo;
    faces: Face[];
    showFaces: boolean;  // 是否显示人脸框
}

// 功能：
// 1. 在照片上绘制人脸框
// 2. 人脸框悬停显示标注按钮
// 3. 点击人脸框打开标注对话框
```

### 5.3 人脸搜索界面 (FaceSearch)

```tsx
interface FaceSearchProps {
    persons: Person[];
    onSelectPerson: (personId: string) => void;
}

// 功能：
// 1. 展示人物头像网格
// 2. 点击人物查看相关照片
// 3. 支持按人物名称搜索
```

---

## 6. 实施计划

### 阶段 1: 后端基础 (2天)

| 任务 | 文件 | 说明 |
|------|------|------|
| 安装依赖 | `requirements.txt` | face_recognition, dlib |
| 数据库迁移 | `internal/db/migrations` | 创建 faces, persons 表 |
| 人脸检测服务 | `internal/face/detection.go` | 调用 Python 服务 |
| Python 检测服务 | `ml-service/face_detection.py` | 人脸检测实现 |
| API 处理器 | `internal/api/handlers/faces.go` | REST API |

### 阶段 2: 前端界面 (2天)

| 任务 | 文件 | 说明 |
|------|------|------|
| 人脸标记组件 | `web/src/components/FacePanel.tsx` | 人脸展示和标注 |
| 照片查看器增强 | `web/src/components/PhotoView.tsx` | 人脸框绘制 |
| 人物搜索页面 | `web/src/pages/FaceSearch.tsx` | 按人物搜索 |
| API 客户端 | `web/src/services/faces.ts` | 前端 API 调用 |

### 阶段 3: 集成测试 (1天)

| 任务 | 说明 |
|------|------|
| 端到端测试 | 扫描 → 检测 → 标注 → 搜索 |
| 性能测试 | 大批量照片处理 |
| 边界测试 | 无人脸、多人脸、模糊人脸 |

---

## 7. 关键技术点

### 7.1 人脸检测

```python
# ml-service/face_detection.py
import face_recognition
import numpy as np
from PIL import Image
import io

class FaceDetector:
    def __init__(self):
        self.model = "hog"  # 或 "cnn" (GPU加速)
    
    def detect(self, image_path):
        """检测人脸并返回位置和特征"""
        image = face_recognition.load_image_file(image_path)
        
        # 检测人脸位置
        locations = face_recognition.face_locations(image, model=self.model)
        
        # 提取人脸特征 (128维向量)
        encodings = face_recognition.face_encodings(image, locations)
        
        faces = []
        for i, (location, encoding) in enumerate(zip(locations, encodings)):
            top, right, bottom, left = location
            faces.append({
                "index": i,
                "location": {
                    "top": top,
                    "right": right,
                    "bottom": bottom,
                    "left": left
                },
                "encoding": encoding.tolist(),  # numpy数组转列表
                "confidence": 0.99  # face_recognition默认高置信度
            })
        
        return faces
    
    def extract_thumbnail(self, image_path, location, size=150):
        """提取人脸缩略图"""
        image = Image.open(image_path)
        top, right, bottom, left = location
        
        # 扩展人脸区域 (包含更多上下文)
        margin = int((bottom - top) * 0.3)
        top = max(0, top - margin)
        bottom = min(image.height, bottom + margin)
        left = max(0, left - margin)
        right = min(image.width, right + margin)
        
        face_image = image.crop((left, top, right, bottom))
        face_image = face_image.resize((size, size), Image.Resampling.LANCZOS)
        
        # 转为 base64 或保存文件
        buffer = io.BytesIO()
        face_image.save(buffer, format="JPEG")
        return buffer.getvalue()
```

### 7.2 人脸相似度计算

```python
def compare_faces(known_encoding, unknown_encoding, tolerance=0.6):
    """
    比较两个人脸的相似度
    
    Args:
        known_encoding: 已知人脸特征
        unknown_encoding: 未知人脸特征
        tolerance: 阈值，越小越严格 (默认0.6)
    
    Returns:
        (is_match, distance)
    """
    distance = face_recognition.face_distance([known_encoding], [unknown_encoding])[0]
    is_match = distance <= tolerance
    return is_match, distance
```

### 7.3 人脸聚类 (自动分组)

```python
from sklearn.cluster import DBSCAN
import numpy as np

def cluster_faces(face_encodings, eps=0.5, min_samples=2):
    """
    使用 DBSCAN 聚类相似人脸
    
    Args:
        face_encodings: 人脸特征向量列表
        eps: 邻域半径
        min_samples: 最小样本数
    
    Returns:
        labels: 聚类标签 (-1 表示噪声/未分类)
    """
    # 转换为 numpy 数组
    X = np.array(face_encodings)
    
    # DBSCAN 聚类
    clustering = DBSCAN(eps=eps, min_samples=min_samples, metric="euclidean")
    labels = clustering.fit_predict(X)
    
    return labels
```

---

## 8. 界面设计

### 8.1 人脸标记面板

```
┌─────────────────────────────────────┐
│  人脸标记                            │
├─────────────────────────────────────┤
│                                     │
│  ┌─────┐ ┌─────┐ ┌─────┐          │
│  │ 😊  │ │ 😊  │ │ 😊  │          │
│  │     │ │     │ │     │          │
│  └─────┘ └─────┘ └─────┘          │
│  人脸1   人脸2   人脸3            │
│  [标注]  [标注]  [标注]            │
│                                     │
│  已标注人物:                        │
│  • 张三 (8张照片)                   │
│  • 李四 (5张照片)                   │
│                                     │
└─────────────────────────────────────┘
```

### 8.2 照片查看器 (带人脸框)

```
┌─────────────────────────────────────┐
│                                     │
│    ┌─────────┐                      │
│    │  😊     │  ← 人脸框 + 标注按钮 │
│    │  张三   │                      │
│    └─────────┘                      │
│                                     │
│         [照片内容]                   │
│                                     │
│    ┌─────────┐                      │
│    │  😊     │  ← 未标注人脸        │
│    │ [标注]  │                      │
│    └─────────┘                      │
│                                     │
└─────────────────────────────────────┘
```

---

## 9. 性能考虑

### 9.1 检测性能

| 场景 | 时间 | 优化方案 |
|------|------|----------|
| 单张照片 (1人脸) | ~0.5s | 异步处理 |
| 单张照片 (多人脸) | ~1s | GPU加速 |
| 批量处理 (100张) | ~2min | 并行处理 |

### 9.2 存储优化

- 人脸缩略图: 150x150 JPEG (~10KB)
- 人脸特征向量: 128维 float32 (~512B)
- 不存储原始人脸截图，只存位置和缩略图

---

## 10. 错误处理

| 错误场景 | 处理方案 |
|----------|----------|
| 无人脸 | 标记 `has_faces=false` |
| 模糊人脸 | 低置信度，提示用户 |
| 检测失败 | 记录日志，跳过该照片 |
| 重复标注 | 更新现有人物关联 |

---

## 11. 验收标准

- [ ] 自动检测照片中的人脸 (>95% 准确率)
- [ ] 前端展示人脸缩略图和位置框
- [ ] 用户可标注人物名称
- [ ] 支持按人物搜索照片
- [ ] 人脸聚类自动分组
- [ ] 处理速度 < 1s/照片

---

*计划完成，等待主人确认后实施*
