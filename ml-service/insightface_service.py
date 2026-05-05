"""
InsightFace 人脸检测服务
替换 face_recognition，提供更准确的人脸检测、特征提取和质量评估
"""

import numpy as np
import cv2
import os
import logging
from typing import List, Dict, Optional, Tuple
from pathlib import Path

logger = logging.getLogger(__name__)

# Lazy-load InsightFace
_face_app = None


def get_face_app():
    """获取全局 InsightFace 实例（懒加载）"""
    global _face_app
    if _face_app is None:
        from insightface.app import FaceAnalysis
        _face_app = FaceAnalysis(
            name='buffalo_l',
            providers=['CPUExecutionProvider']
        )
        _face_app.prepare(ctx_id=0, det_size=(640, 640))
        logger.info("InsightFace buffalo_l loaded successfully")
    return _face_app


def detect_faces_insightface(
    image_path: str,
    min_det_score: float = 0.5,
    min_size: int = 60,
) -> List[Dict]:
    """
    使用 InsightFace 检测人脸
    
    Args:
        image_path: 照片路径
        min_det_score: 最小检测分数 (InsightFace 默认 ~0.5, 可靠人脸通常 > 0.7)
        min_size: 最小人脸尺寸（像素）
    
    Returns:
        人脸列表，每项包含: index, location, encoding, confidence, age, gender, landmarks
    """
    if not os.path.exists(image_path):
        logger.warning(f"Image not found: {image_path}")
        return []

    try:
        app = get_face_app()
        img = cv2.imread(image_path)
        if img is None:
            logger.warning(f"Failed to load image: {image_path}")
            return []

        img_rgb = cv2.cvtColor(img, cv2.COLOR_BGR2RGB)
        faces = app.get(img)

        results = []
        for i, face in enumerate(faces):
            bbox = face.bbox.astype(int)
            left, top, right, bottom = bbox[0], bbox[1], bbox[2], bbox[3]
            width = right - left
            height = bottom - top

            # 过滤
            if face.det_score < min_det_score:
                logger.debug(
                    f"Filtered low-score face in {os.path.basename(image_path)}: "
                    f"score={face.det_score:.3f} < {min_det_score}"
                )
                continue
            if width < min_size or height < min_size:
                logger.debug(
                    f"Filtered small face: {width}x{height} < {min_size}"
                )
                continue

            # InsightFace 的 embedding 是 normed embedding (512维或512维)
            encoding = face.normed_embedding
            if encoding is None:
                continue
            encoding_list = encoding.tolist()

            # 质量分数：综合检测分数 + 人脸大小 + 清晰度
            quality_score = _calculate_quality(img_rgb, face, bbox)

            face_data = {
                "index": i,
                "location": {
                    "top": int(top),
                    "right": int(right),
                    "bottom": int(bottom),
                    "left": int(left),
                    "width": int(width),
                    "height": int(height),
                },
                "encoding": encoding_list,
                "confidence": float(quality_score),  # 综合质量分数
                "det_score": float(face.det_score),   # InsightFace 原始检测分数
                "age": int(face.age) if hasattr(face, 'age') and face.age else None,
                "gender": int(face.gender) if hasattr(face, 'gender') else None,
            }
            results.append(face_data)

        logger.info(
            f"InsightFace: {len(results)} faces in {os.path.basename(image_path)}"
        )
        return results

    except Exception as e:
        logger.error(f"InsightFace detection failed for {image_path}: {e}")
        return []


def _calculate_quality(img_rgb: np.ndarray, face, bbox) -> float:
    """
    计算人脸综合质量分数 (0-1)
    
    综合考虑:
    - InsightFace 检测分数 (0-1)
    - 人脸区域清晰度
    - 人脸大小
    """
    left, top, right, bottom = bbox
    width = right - left
    height = bottom - top

    # 基础分：InsightFace 检测分数（权重 0.6）
    base_score = float(face.det_score)

    # 清晰度分（权重 0.2）
    face_region = img_rgb[top:bottom, left:right]
    if face_region.size > 0:
        gray = cv2.cvtColor(face_region, cv2.COLOR_RGB2GRAY)
        laplacian_var = cv2.Laplacian(gray, cv2.CV_64F).var()
        sharpness = min(1.0, laplacian_var / 500.0)
    else:
        sharpness = 0.0

    # 人脸大小分（权重 0.2）
    min_dim = min(width, height)
    if min_dim >= 150:
        size_score = 1.0
    elif min_dim >= 100:
        size_score = 0.8
    elif min_dim >= 80:
        size_score = 0.6
    else:
        size_score = 0.4

    # 综合分数
    quality = base_score * 0.6 + sharpness * 0.2 + size_score * 0.2
    return round(float(quality), 4)


def extract_thumbnail(
    image_path: str,
    location: Dict,
    output_path: Optional[str] = None,
    padding_ratio: float = 0.3,
) -> Optional[bytes]:
    """
    从原图裁剪人脸缩略图
    
    Args:
        image_path: 原图路径
        location: 人脸位置 {"top", "right", "bottom", "left"}
        output_path: 可选输出文件路径
        padding_ratio: 扩展比例 (默认 0.3 = 30%)
    
    Returns:
        JPEG 字节数据，或 None
    """
    try:
        img = cv2.imread(image_path)
        if img is None:
            return None

        top = location['top']
        right = location['right']
        bottom = location['bottom']
        left = location['left']

        face_w = right - left
        face_h = bottom - top
        pad_x = int(face_w * padding_ratio)
        pad_y = int(face_h * padding_ratio)

        h, w = img.shape[:2]
        x0 = max(0, left - pad_x)
        y0 = max(0, top - pad_y)
        x1 = min(w, right + pad_x)
        y1 = min(h, bottom + pad_y)

        face_crop = img[y0:y1, x0:x1]
        if face_crop.size == 0:
            return None

        success, jpeg = cv2.imencode('.jpg', face_crop, [cv2.IMWRITE_JPEG_QUALITY, 85])
        if not success:
            return None

        if output_path:
            os.makedirs(os.path.dirname(output_path), exist_ok=True)
            with open(output_path, 'wb') as f:
                f.write(jpeg.tobytes())

        return jpeg.tobytes()

    except Exception as e:
        logger.error(f"Thumbnail extraction failed: {e}")
        return None
