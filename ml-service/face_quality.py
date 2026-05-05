"""
人脸质量评估模块
提供模糊度、姿态、遮挡等质量检测
"""

import numpy as np
import cv2
from typing import Dict, Tuple, Optional
import logging

logger = logging.getLogger(__name__)


class FaceQualityChecker:
    """人脸质量检查器"""
    
    def __init__(self):
        self.blur_threshold = 100  # 模糊度阈值
        self.pose_threshold = 30   # 姿态角度阈值(度)
        self.occlusion_threshold = 0.3  # 遮挡比例阈值
    
    def check_quality(self, image: np.ndarray, face_location: Dict) -> Dict:
        """
        综合质量检查
        
        Returns:
            {
                'blur_score': float,      # 清晰度分数 (越高越好)
                'blur_pass': bool,        # 清晰度是否通过
                'pose_score': float,      # 姿态分数
                'pose_pass': bool,        # 姿态是否通过
                'occlusion_score': float, # 遮挡分数
                'occlusion_pass': bool,   # 遮挡是否通过
                'overall_score': float,   # 综合分数 (0-1)
                'quality_pass': bool      # 是否通过所有检查
            }
        """
        top = face_location.get('top', 0)
        right = face_location.get('right', 0)
        bottom = face_location.get('bottom', 0)
        left = face_location.get('left', 0)
        
        # 裁剪人脸区域
        face_region = image[top:bottom, left:right]
        if face_region.size == 0:
            return self._failed_result()
        
        # 1. 模糊度检查
        blur_score, blur_pass = self.check_blur(face_region)
        
        # 2. 姿态检查 (简化版，基于人脸宽高比)
        pose_score, pose_pass = self.check_pose(face_region, face_location)
        
        # 3. 遮挡检查 (简化版，基于边缘完整性)
        occlusion_score, occlusion_pass = self.check_occlusion(face_region)
        
        # 综合评分
        overall_score = (blur_score * 0.4 + pose_score * 0.3 + occlusion_score * 0.3)
        quality_pass = blur_pass and pose_pass and occlusion_pass
        
        return {
            'blur_score': blur_score,
            'blur_pass': blur_pass,
            'pose_score': pose_score,
            'pose_pass': pose_pass,
            'occlusion_score': occlusion_score,
            'occlusion_pass': occlusion_pass,
            'overall_score': overall_score,
            'quality_pass': quality_pass
        }
    
    def check_blur(self, face_region: np.ndarray) -> Tuple[float, bool]:
        """
        检查人脸清晰度 (Laplacian方差)
        
        Returns:
            (score, pass)
            score: 0-1 (1表示最清晰)
            pass: 是否通过阈值
        """
        try:
            # 转为灰度图
            if face_region.ndim == 3:
                gray = cv2.cvtColor(face_region, cv2.COLOR_RGB2GRAY)
            else:
                gray = face_region
            
            # Laplacian方差
            laplacian_var = cv2.Laplacian(gray, cv2.CV_64F).var()
            
            # 归一化分数 (假设1000为非常清晰)
            score = min(1.0, laplacian_var / 1000.0)
            
            # 通过阈值判断
            pass_check = laplacian_var > self.blur_threshold
            
            return score, pass_check
            
        except Exception as e:
            logger.warning(f"Blur check failed: {e}")
            return 0.0, False
    
    def check_pose(self, face_region: np.ndarray, face_location: Dict) -> Tuple[float, bool]:
        """
        检查人脸姿态 (简化版)
        
        基于人脸宽高比和区域对称性估算
        
        Returns:
            (score, pass)
        """
        try:
            height = face_location.get('bottom', 0) - face_location.get('top', 0)
            width = face_location.get('right', 0) - face_location.get('left', 0)
            
            # 理想宽高比 ~1.2 (人脸高度略大于宽度)
            ideal_ratio = 1.2
            actual_ratio = height / max(width, 1)
            
            # 比例偏差 (0表示完美)
            ratio_deviation = abs(actual_ratio - ideal_ratio) / ideal_ratio
            
            # 分数 (1 - 偏差)
            score = max(0.0, 1.0 - ratio_deviation)
            
            # 通过检查 (偏差 < 30%)
            pass_check = ratio_deviation < 0.3
            
            return score, pass_check
            
        except Exception as e:
            logger.warning(f"Pose check failed: {e}")
            return 0.0, False
    
    def check_occlusion(self, face_region: np.ndarray) -> Tuple[float, bool]:
        """
        检查人脸遮挡 (简化版)
        
        基于边缘检测判断人脸完整性
        
        Returns:
            (score, pass)
        """
        try:
            if face_region.ndim == 3:
                gray = cv2.cvtColor(face_region, cv2.COLOR_RGB2GRAY)
            else:
                gray = face_region
            
            # 边缘检测
            edges = cv2.Canny(gray, 100, 200)
            
            # 计算边缘密度
            edge_density = np.sum(edges > 0) / edges.size
            
            # 理想边缘密度 ~0.05-0.15
            ideal_min = 0.05
            ideal_max = 0.15
            
            if edge_density < ideal_min:
                # 边缘太少，可能过度模糊或遮挡
                score = edge_density / ideal_min
            elif edge_density > ideal_max:
                # 边缘太多，可能背景复杂
                score = max(0.0, 1.0 - (edge_density - ideal_max) / 0.1)
            else:
                # 理想范围
                score = 1.0
            
            # 通过检查 (边缘密度在合理范围)
            pass_check = 0.02 < edge_density < 0.25
            
            return score, pass_check
            
        except Exception as e:
            logger.warning(f"Occlusion check failed: {e}")
            return 0.0, False
    
    def _failed_result(self) -> Dict:
        """返回失败结果"""
        return {
            'blur_score': 0.0,
            'blur_pass': False,
            'pose_score': 0.0,
            'pose_pass': False,
            'occlusion_score': 0.0,
            'occlusion_pass': False,
            'overall_score': 0.0,
            'quality_pass': False
        }


# 便捷函数
def check_face_quality(image: np.ndarray, face_location: Dict) -> Dict:
    """检查单个人脸质量"""
    checker = FaceQualityChecker()
    return checker.check_quality(image, face_location)


def batch_check_quality(image: np.ndarray, face_locations: list) -> list:
    """批量检查人脸质量"""
    checker = FaceQualityChecker()
    results = []
    for loc in face_locations:
        result = checker.check_quality(image, loc)
        results.append(result)
    return results
