"""Zero-shot image classification using CLIP."""

from __future__ import annotations

import numpy as np
from PIL import Image
from dataclasses import dataclass

from services.clip_service import ClipService


@dataclass
class ClassificationResult:
    label: str
    score: float


class ImageClassifier:
    """Classify images against a set of text labels using CLIP similarity."""

    def __init__(self, clip_service: ClipService, labels: list[str] | None = None):
        self.clip = clip_service
        self.labels: list[str] = labels or []
        self._label_feats: np.ndarray | None = None
        if self.labels:
            self._build_label_features()

    def _build_label_features(self) -> None:
        self._label_feats = self.clip.encode_text(self.labels)

    def set_labels(self, labels: list[str]) -> None:
        """Update the label set and re-encode."""
        self.labels = labels
        self._build_label_features()

    def classify(
        self,
        image: Image.Image | bytes,
        top_k: int = 5,
        labels: list[str] | None = None,
    ) -> list[ClassificationResult]:
        """Classify an image; return top-k (label, score) pairs sorted desc."""
        if labels:
            label_feats = self.clip.encode_text(labels)
            use_labels = labels
        elif self._label_feats is not None:
            label_feats = self._label_feats
            use_labels = self.labels
        else:
            raise ValueError("No labels provided for classification")

        img_feat = self.clip.encode_image(image)
        sims = label_feats @ img_feat  # (N,)

        # Softmax for probability-like scores
        exp_sims = np.exp(sims - sims.max())
        probs = exp_sims / exp_sims.sum()

        top_indices = probs.argsort()[::-1][:top_k]
        return [ClassificationResult(label=use_labels[i], score=float(probs[i])) for i in top_indices]

    def classify_batch(
        self,
        images: list[Image.Image | bytes],
        top_k: int = 5,
        labels: list[str] | None = None,
    ) -> list[list[ClassificationResult]]:
        """Classify a batch of images."""
        if labels:
            label_feats = self.clip.encode_text(labels)
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
            exp_row = np.exp(row - row.max())
            probs = exp_row / exp_row.sum()
            top_idx = probs.argsort()[::-1][:top_k]
            results.append([ClassificationResult(label=use_labels[i], score=float(probs[i])) for i in top_idx])
        return results
