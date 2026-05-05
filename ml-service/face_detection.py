"""
人脸检测服务
提供人脸检测、特征提取、相似度计算等功能
"""

try:
    import face_recognition
    _FACE_RECOGNITION_AVAILABLE = True
except BaseException:
    _FACE_RECOGNITION_AVAILABLE = False

import numpy as np
from PIL import Image
import io
import base64
from typing import List, Dict, Tuple, Optional
import logging

# 导入质量检查模块
try:
    from face_quality import FaceQualityChecker, check_face_quality
    _QUALITY_CHECK_AVAILABLE = True
except ImportError:
    _QUALITY_CHECK_AVAILABLE = False

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
        self._available = None
        logger.info(f"FaceDetector initialized with model: {model}")
    
    @property
    def available(self) -> bool:
        """检测 face_recognition 库是否可用"""
        if self._available is not None:
            return self._available
        self._available = _FACE_RECOGNITION_AVAILABLE
        return self._available
    
    def detect(self, image_path: str) -> List[Dict]:
        """
        检测照片中的人脸 (detect_faces 的别名)
        
        Args:
            image_path: 照片路径
            
        Returns:
            人脸列表
        """
        return self.detect_faces(image_path)
    
    def detect_from_bytes(self, data: bytes) -> List[Dict]:
        """
        从字节数据检测人脸
        
        Args:
            data: 图片字节数据
            
        Returns:
            人脸列表，包含位置、特征向量等信息
        """
        try:
            # 从字节加载图片
            pil_image = Image.open(io.BytesIO(data))
            # 转为 numpy 数组 (RGB)
            image = np.array(pil_image.convert("RGB"))
            
            # 检测人脸位置
            face_locations = face_recognition.face_locations(image, model=self.model)
            
            if not face_locations:
                logger.info(f"No faces detected in byte data")
                return []
            
            # 提取人脸特征
            face_encodings = face_recognition.face_encodings(image, face_locations)
            
            faces = []
            for i, (location, encoding) in enumerate(zip(face_locations, face_encodings)):
                top, right, bottom, left = location
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
                    "encoding": encoding.tolist(),
                    "confidence": confidence
                })
            
            # 过滤低质量检测
            faces = self.filter_faces(faces, min_confidence=0.5, min_size=80)
            
            logger.info(f"Detected {len(faces)} faces from byte data")
            return faces
            
        except Exception as e:
            logger.error(f"Face detection from bytes failed: {e}")
            return []
    
    def detect_faces(self, image_path: str) -> List[Dict]:
        """
        检测照片中的人脸 (增强版，带质量检查)
        
        Args:
            image_path: 照片路径
            
        Returns:
            人脸列表，包含位置、特征向量、质量分数等信息
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
                
                # 基础人脸信息
                face_location = {
                    "top": top,
                    "right": right,
                    "bottom": bottom,
                    "left": left,
                    "width": right - left,
                    "height": bottom - top
                }
                
                # 计算人脸置信度 (基于检测质量)
                confidence = self._calculate_confidence(image, location)
                
                # 第二阶段优化: 质量检查
                quality_result = None
                if _QUALITY_CHECK_AVAILABLE:
                    try:
                        quality_result = check_face_quality(image, face_location)
                        # 质量检查未通过，降低置信度
                        if not quality_result['quality_pass']:
                            confidence *= 0.5  # 惩罚
                            logger.debug(f"Face {i} failed quality check: {quality_result}")
                    except Exception as e:
                        logger.warning(f"Quality check failed for face {i}: {e}")
                
                face_info = {
                    "index": i,
                    "location": face_location,
                    "encoding": encoding.tolist(),  # numpy数组转列表
                    "confidence": confidence,
                    "quality": quality_result  # 新增质量信息
                }
                
                faces.append(face_info)
            
            # 过滤低质量检测 (第二阶段: 增加质量过滤)
            faces = self.filter_faces(faces, min_confidence=0.5, min_size=80)
            
            # 额外过滤质量检查未通过的
            if _QUALITY_CHECK_AVAILABLE:
                faces = [f for f in faces if f.get('quality') is None or f['quality'].get('quality_pass', True)]
            
            logger.info(f"Detected {len(faces)} faces in {image_path}")
            return faces
            
        except Exception as e:
            logger.error(f"Face detection failed for {image_path}: {e}")
            return []
    
    def extract_thumbnail_bytes(self, image_path: str, location: Dict, size: int = 150) -> bytes:
        """
        提取人脸缩略图，返回 JPEG 字节
        
        Args:
            image_path: 照片路径
            location: 人脸位置 {top, right, bottom, left}
            size: 缩略图尺寸
            
        Returns:
            JPEG 格式的缩略图字节数据
        """
        try:
            image = Image.open(image_path)
            top = location["top"]
            right = location["right"]
            bottom = location["bottom"]
            left = location["left"]
            
            # 扩展人脸区域
            margin = int((bottom - top) * 0.3)
            top = max(0, top - margin)
            bottom = min(image.height, bottom + margin)
            left = max(0, left - margin)
            right = min(image.width, right + margin)
            
            # 裁剪并缩放
            face_image = image.crop((left, top, right, bottom))
            face_image = face_image.resize((size, size), Image.Resampling.LANCZOS)
            
            # 转为 JPEG 字节
            buffer = io.BytesIO()
            face_image.save(buffer, format="JPEG", quality=85)
            return buffer.getvalue()
            
        except Exception as e:
            logger.error(f"Thumbnail extraction failed: {e}")
            return b""
    
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
        
        基于人脸大小、宽高比、清晰度、人脸占比等因素
        """
        top, right, bottom, left = location
        face_width = right - left
        face_height = bottom - top
        
        # 人脸占图片比例
        image_height, image_width = image.shape[:2]
        face_area = face_width * face_height
        image_area = image_width * image_height
        face_ratio = face_area / image_area
        
        # 基础置信度
        confidence = 0.5
        
        # 人脸大小检查
        min_pixels = 100
        if face_width < min_pixels or face_height < min_pixels:
            confidence -= 0.3
        elif face_width < 80 or face_height < 80:
            confidence -= 0.15
        else:
            confidence += 0.05
        
        # 宽高比检查
        aspect_ratio = face_height / max(face_width, 1)
        if aspect_ratio < 0.8 or aspect_ratio > 1.5:
            confidence -= 0.25
        elif aspect_ratio < 0.9 or aspect_ratio > 1.3:
            confidence -= 0.1
        else:
            confidence += 0.05
        
        # 人脸占图片比例检查
        if face_ratio < 0.005:
            confidence -= 0.25
        elif face_ratio < 0.01:
            confidence -= 0.1
        elif face_ratio > 0.8:
            confidence -= 0.25
        elif face_ratio > 0.6:
            confidence -= 0.1
        elif 0.02 < face_ratio < 0.3:
            confidence += 0.05
        
        # 清晰度估算 (基于边缘检测)
        face_region = image[top:bottom, left:right]
        if face_region.size > 0:
            gray = np.mean(face_region, axis=2) if face_region.ndim == 3 else face_region
            grad_x = np.abs(np.diff(gray, axis=1))
            grad_y = np.abs(np.diff(gray, axis=0))
            sharpness = (np.mean(grad_x) + np.mean(grad_y)) / 2
            
            # 模糊惩罚
            if sharpness < 10:  # 很模糊
                confidence -= 0.2
            elif sharpness < 20:  # 有点模糊
                confidence -= 0.05
            else:  # 清晰
                confidence += 0.1
        
        # 确保在合理范围
        return max(0.1, min(0.99, confidence))
    
    def filter_faces(self, faces: List[Dict], min_confidence: float = 0.5, 
                     min_size: int = 80, max_ratio_deviation: float = 0.3) -> List[Dict]:
        """
        过滤低质量的人脸检测
        
        Args:
            faces: 原始检测到的人脸列表
            min_confidence: 最小置信度阈值 (默认0.5)
            min_size: 最小人脸像素尺寸 (默认80)
            max_ratio_deviation: 最大宽高比偏差
            
        Returns:
            过滤后的人脸列表
        """
        filtered = []
        for face in faces:
            loc = face["location"]
            width = loc.get("width", loc.get("right", 0) - loc.get("left", 0))
            height = loc.get("height", loc.get("bottom", 0) - loc.get("top", 0))
            
            # 检查置信度 (严格)
            if face.get("confidence", 0) < min_confidence:
                logger.debug(f"过滤低置信度人脸: {face.get('confidence', 0):.2f} < {min_confidence}")
                continue
            
            # 检查最小尺寸 (严格)
            if width < min_size or height < min_size:
                logger.debug(f"过滤小人脸: {width}x{height} < {min_size}")
                continue
            
            # 检查宽高比 (严格: 0.8 ~ 1.5)
            aspect_ratio = height / max(width, 1)
            if aspect_ratio < 0.8 or aspect_ratio > 1.5:
                logger.debug(f"过滤异常比例人脸: 宽高比={aspect_ratio:.2f}")
                continue
            
            filtered.append(face)
        
        logger.info(f"人脸过滤: {len(faces)} -> {len(filtered)} (过滤 {len(faces)-len(filtered)} 个)")
        return filtered


# 全局检测器实例
_detector = None
detector: FaceDetector = None  # 由 get_detector 初始化

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


# 在模块加载时初始化全局 detector
detector = get_detector()
