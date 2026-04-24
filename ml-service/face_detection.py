"""
人脸检测服务
提供人脸检测、特征提取、相似度计算等功能
"""

import face_recognition
import numpy as np
from PIL import Image
import io
import base64
from typing import List, Dict, Tuple, Optional
import logging

logger = logging.getLogger(__name__)


class FaceDetector:
    """人脸检测器"""
    
    def __init__(self, model: str = "hog"):
        """
        初始化检测器
        
        Args:
            model: 检测模型，"hog" (CPU) 或 "cnn" (GPU)
        """
        self.model = model
        logger.info(f"FaceDetector initialized with model: {model}")
    
    def detect_faces(self, image_path: str) -> List[Dict]:
        """
        检测照片中的人脸
        
        Args:
            image_path: 照片路径
            
        Returns:
            人脸列表，包含位置、特征向量等信息
        """
        try:
            # 加载图片
            image = face_recognition.load_image_file(image_path)
            
            # 检测人脸位置
            face_locations = face_recognition.face_locations(image, model=self.model)
            
            if not face_locations:
                logger.info(f"No faces detected in {image_path}")
                return []
            
            # 提取人脸特征 (128维向量)
            face_encodings = face_recognition.face_encodings(image, face_locations)
            
            faces = []
            for i, (location, encoding) in enumerate(zip(face_locations, face_encodings)):
                top, right, bottom, left = location
                
                # 计算人脸置信度 (基于检测质量)
                confidence = self._calculate_confidence(image, location)
                
                faces.append({
                    "index": i,
                    "location": {
                        "top": top,
                        "right": right,
                        "bottom": bottom,
                        "left": left,
                        "width": right - left,
                        "height": bottom - top
                    },
                    "encoding": encoding.tolist(),  # numpy数组转列表
                    "confidence": confidence
                })
            
            logger.info(f"Detected {len(faces)} faces in {image_path}")
            return faces
            
        except Exception as e:
            logger.error(f"Face detection failed for {image_path}: {e}")
            return []
    
    def extract_thumbnail(self, image_path: str, location: Dict, size: int = 150) -> str:
        """
        提取人脸缩略图
        
        Args:
            image_path: 照片路径
            location: 人脸位置 {top, right, bottom, left}
            size: 缩略图尺寸
            
        Returns:
            base64 编码的缩略图
        """
        try:
            image = Image.open(image_path)
            top = location["top"]
            right = location["right"]
            bottom = location["bottom"]
            left = location["left"]
            
            # 扩展人脸区域 (包含更多上下文)
            margin = int((bottom - top) * 0.3)
            top = max(0, top - margin)
            bottom = min(image.height, bottom + margin)
            left = max(0, left - margin)
            right = min(image.width, right + margin)
            
            # 裁剪人脸
            face_image = image.crop((left, top, right, bottom))
            face_image = face_image.resize((size, size), Image.Resampling.LANCZOS)
            
            # 转为 base64
            buffer = io.BytesIO()
            face_image.save(buffer, format="JPEG", quality=85)
            thumbnail_base64 = base64.b64encode(buffer.getvalue()).decode("utf-8")
            
            return f"data:image/jpeg;base64,{thumbnail_base64}"
            
        except Exception as e:
            logger.error(f"Thumbnail extraction failed: {e}")
            return ""
    
    def compare_faces(self, known_encoding: List[float], unknown_encoding: List[float], 
                     tolerance: float = 0.6) -> Tuple[bool, float]:
        """
        比较两个人脸的相似度
        
        Args:
            known_encoding: 已知人脸特征
            unknown_encoding: 未知人脸特征
            tolerance: 阈值，越小越严格 (默认0.6)
            
        Returns:
            (是否匹配, 距离)
        """
        try:
            # 转换为 numpy 数组
            known = np.array(known_encoding)
            unknown = np.array(unknown_encoding)
            
            # 计算欧氏距离
            distance = face_recognition.face_distance([known], [unknown])[0]
            is_match = distance <= tolerance
            
            return is_match, float(distance)
            
        except Exception as e:
            logger.error(f"Face comparison failed: {e}")
            return False, 1.0
    
    def find_similar_faces(self, target_encoding: List[float], 
                          known_faces: List[Dict], 
                          tolerance: float = 0.6) -> List[Dict]:
        """
        在已知人脸中查找相似的人脸
        
        Args:
            target_encoding: 目标人脸特征
            known_faces: 已知人脸列表 [{"id": str, "encoding": List[float]}, ...]
            tolerance: 相似度阈值
            
        Returns:
            匹配的人脸列表
        """
        matches = []
        
        for face in known_faces:
            is_match, distance = self.compare_faces(
                face["encoding"], 
                target_encoding, 
                tolerance
            )
            
            if is_match:
                matches.append({
                    "id": face.get("id"),
                    "person_id": face.get("person_id"),
                    "distance": distance,
                    "similarity": 1 - distance  # 转换为相似度
                })
        
        # 按相似度排序
        matches.sort(key=lambda x: x["similarity"], reverse=True)
        
        return matches
    
    def _calculate_confidence(self, image: np.ndarray, location: Tuple) -> float:
        """
        计算人脸检测的置信度
        
        基于人脸大小、清晰度等因素
        """
        top, right, bottom, left = location
        face_width = right - left
        face_height = bottom - top
        
        # 人脸占图片比例
        image_height, image_width = image.shape[:2]
        face_ratio = (face_width * face_height) / (image_width * image_height)
        
        # 基础置信度
        confidence = 0.99
        
        # 根据人脸大小调整
        if face_ratio < 0.01:  # 人脸太小
            confidence -= 0.1
        elif face_ratio > 0.5:  # 人脸太大（可能是误检）
            confidence -= 0.05
        
        # 确保在合理范围
        return max(0.5, min(0.99, confidence))


# 全局检测器实例
_detector = None

def get_detector(model: str = "hog") -> FaceDetector:
    """获取全局检测器实例"""
    global _detector
    if _detector is None:
        _detector = FaceDetector(model)
    return _detector


def detect_faces(image_path: str, model: str = "hog") -> List[Dict]:
    """
    便捷函数：检测人脸
    
    Args:
        image_path: 照片路径
        model: 检测模型
        
    Returns:
        人脸列表
    """
    detector = get_detector(model)
    return detector.detect_faces(image_path)


def extract_face_thumbnail(image_path: str, location: Dict, size: int = 150) -> str:
    """
    便捷函数：提取人脸缩略图
    
    Args:
        image_path: 照片路径
        location: 人脸位置
        size: 缩略图尺寸
        
    Returns:
        base64 编码的缩略图
    """
    detector = get_detector()
    return detector.extract_thumbnail(image_path, location, size)


def compare_face_encodings(encoding1: List[float], encoding2: List[float], 
                            tolerance: float = 0.6) -> Tuple[bool, float]:
    """
    便捷函数：比较两个人脸
    
    Args:
        encoding1: 人脸1特征
        encoding2: 人脸2特征
        tolerance: 阈值
        
    Returns:
        (是否匹配, 距离)
    """
    detector = get_detector()
    return detector.compare_faces(encoding1, encoding2, tolerance)
