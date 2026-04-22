"""CLIP model service for image and text feature extraction."""

from __future__ import annotations

import io
import torch
import numpy as np
from PIL import Image
from transformers import CLIPModel, CLIPProcessor


class ClipService:
    """Wraps HuggingFace CLIP for image/text encoding and similarity."""

    def __init__(self, model_name: str = "openai/clip-vit-base-patch32", device: str = None):
        if device is None:
            import torch
            device = "cuda" if torch.cuda.is_available() else "mps" if torch.backends.mps.is_available() else "cpu"
        self.device = device
        self.model = CLIPModel.from_pretrained(model_name).to(self.device).eval()
        self.processor = CLIPProcessor.from_pretrained(model_name)

    # ------------------------------------------------------------------
    # Encoding
    # ------------------------------------------------------------------

    @torch.no_grad()
    def encode_image(self, image: Image.Image | bytes) -> np.ndarray:
        """Encode a single image → (dim,) float32 numpy array (L2-normalised)."""
        if isinstance(image, (bytes, bytearray)):
            image = Image.open(io.BytesIO(image)).convert("RGB")
        inputs = self.processor(images=image, return_tensors="pt").to(self.device)
        feat = self.model.get_image_features(**inputs)
        # 兼容新版 transformers，feat 可能是 BaseModelOutput
        if hasattr(feat, 'pooler_output'):
            feat = feat.pooler_output
        elif hasattr(feat, 'last_hidden_state'):
            feat = feat.last_hidden_state.mean(dim=1)
        feat = feat / feat.norm(dim=-1, keepdim=True)
        return feat.cpu().numpy().flatten()

    @torch.no_grad()
    def encode_image_batch(self, images: list[Image.Image | bytes]) -> np.ndarray:
        """Encode a batch of images → (N, dim) float32 numpy array."""
        pil_images = []
        for img in images:
            if isinstance(img, (bytes, bytearray)):
                img = Image.open(io.BytesIO(img)).convert("RGB")
            pil_images.append(img)
        inputs = self.processor(images=pil_images, return_tensors="pt", padding=True).to(self.device)
        feats = self.model.get_image_features(**inputs)
        if hasattr(feats, 'pooler_output'):
            feats = feats.pooler_output
        elif hasattr(feats, 'last_hidden_state'):
            feats = feats.last_hidden_state.mean(dim=1)
        feats = feats / feats.norm(dim=-1, keepdim=True)
        return feats.cpu().numpy()

    @torch.no_grad()
    def encode_text(self, text: str | list[str]) -> np.ndarray:
        """Encode text → (dim,) or (N, dim) float32 numpy array (L2-normalised)."""
        if isinstance(text, str):
            text = [text]
        inputs = self.processor(text=text, return_tensors="pt", padding=True, truncation=True).to(self.device)
        feats = self.model.get_text_features(**inputs)
        if hasattr(feats, 'pooler_output'):
            feats = feats.pooler_output
        elif hasattr(feats, 'last_hidden_state'):
            feats = feats.last_hidden_state.mean(dim=1)
        feats = feats / feats.norm(dim=-1, keepdim=True)
        result = feats.cpu().numpy()
        return result.flatten() if len(text) == 1 else result

    # ------------------------------------------------------------------
    # Similarity
    # ------------------------------------------------------------------

    def similarity(self, image_feat: np.ndarray, text_feat: np.ndarray) -> float:
        """Cosine similarity between a single image and text feature vector."""
        return float(np.dot(image_feat, text_feat))

    def similarity_matrix(self, image_feats: np.ndarray, text_feats: np.ndarray) -> np.ndarray:
        """Cosine similarity matrix (N_images × N_texts)."""
        return image_feats @ text_feats.T
