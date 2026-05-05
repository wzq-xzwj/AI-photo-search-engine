"""V2 分类器：Prompt 模板 + 阈值过滤，优化 Chinese CLIP 的标签质量"""

from __future__ import annotations

import numpy as np
from PIL import Image
from dataclasses import dataclass
from typing import Optional

from services.clip_service import ClipService


@dataclass
class ClassificationResult:
    label: str
    score: float


# Chinese CLIP prompt templates — CLIP 对完整句子比单词编码好得多
PROMPT_TEMPLATES = [
    "一张{label}的照片",
    "照片里是{label}",
    "这张照片展示了{label}",
    "画面中有{label}",
]


class ClassifierV2:
    """V2 分类器：用 Prompt 模板编码标签 + 阈值过滤 + 多策略得分"""

    def __init__(
        self,
        clip_service: ClipService,
        labels: Optional[list[str]] = None,
        min_score: float = 0.18,        # 余弦相似度最低阈值
        top_k: int = 8,                  # 最多取几个候选
        use_prompt_template: bool = True,
    ):
        self.clip = clip_service
        self.labels: list[str] = labels or []
        self.min_score = min_score
        self.top_k = top_k
        self.use_prompt_template = use_prompt_template
        self._label_feats: Optional[np.ndarray] = None
        if self.labels:
            self._build_label_features()

    def _make_prompt(self, label: str) -> str:
        """用模板生成描述性文本，CLIP 对这类文本编码精度更高"""
        if not self.use_prompt_template:
            return label
        # 平均多个模板的编码，更鲁棒
        return PROMPT_TEMPLATES[0].format(label=label)

    def _build_label_features(self) -> None:
        """用 prompt 模板编码所有标签"""
        prompts = [self._make_prompt(l) for l in self.labels]
        self._label_feats = self.clip.encode_text(prompts)

    def set_labels(self, labels: list[str]) -> None:
        self.labels = labels
        self._build_label_features()

    def classify(
        self,
        image: Image.Image | bytes,
        top_k: Optional[int] = None,
        min_score: Optional[float] = None,
        labels: Optional[list[str]] = None,
    ) -> list[ClassificationResult]:
        """单张图片分类，返回超过阈值的标签"""
        top_k = top_k or self.top_k
        min_score = min_score or self.min_score

        if labels:
            prompts = [self._make_prompt(l) for l in labels]
            label_feats = self.clip.encode_text(prompts)
            use_labels = labels
        elif self._label_feats is not None:
            label_feats = self._label_feats
            use_labels = self.labels
        else:
            raise ValueError("No labels provided")

        img_feat = self.clip.encode_image(image)
        sims = label_feats @ img_feat  # (N,) cosine similarity

        # 阈值过滤：只取分数 > min_score 的
        top_indices = sims.argsort()[::-1]
        results = []
        for i in top_indices:
            if float(sims[i]) < min_score:
                break
            results.append(ClassificationResult(
                label=use_labels[i],
                score=float(sims[i])
            ))
            if len(results) >= top_k:
                break

        return results

    def classify_batch(
        self,
        images: list[Image.Image | bytes],
        top_k: Optional[int] = None,
        min_score: Optional[float] = None,
        labels: Optional[list[str]] = None,
    ) -> list[list[ClassificationResult]]:
        """批量图片分类"""
        top_k = top_k or self.top_k
        min_score = min_score or self.min_score

        if labels:
            prompts = [self._make_prompt(l) for l in labels]
            label_feats = self.clip.encode_text(prompts)
            use_labels = labels
        elif self._label_feats is not None:
            label_feats = self._label_feats
            use_labels = self.labels
        else:
            raise ValueError("No labels provided")

        img_feats = self.clip.encode_image_batch(images)
        sims = img_feats @ label_feats.T  # (B, N)

        results = []
        for row in sims:
            top_indices = row.argsort()[::-1]
            img_results = []
            for i in top_indices:
                score = float(row[i])
                if score < min_score:
                    break
                img_results.append(ClassificationResult(
                    label=use_labels[i],
                    score=score
                ))
                if len(img_results) >= top_k:
                    break
            results.append(img_results)
        return results

    def similarity_batch(
        self,
        images: list[Image.Image | bytes],
        labels: list[str],
        top_k: Optional[int] = None,
        min_score: Optional[float] = None,
    ) -> list[list[ClassificationResult]]:
        """Compute raw cosine similarity with prompt templates and threshold"""
        top_k = top_k or self.top_k
        min_score = min_score or self.min_score

        prompts = [self._make_prompt(l) for l in labels]
        label_feats = self.clip.encode_text(prompts)
        img_feats = self.clip.encode_image_batch(images)
        sims = img_feats @ label_feats.T  # (B, N)

        results = []
        for row in sims:
            top_indices = row.argsort()[::-1]
            img_results = []
            for i in top_indices:
                score = float(row[i])
                if score < min_score:
                    break
                img_results.append(ClassificationResult(
                    label=labels[i],
                    score=score
                ))
                if len(img_results) >= top_k:
                    break
            results.append(img_results)
        return results
