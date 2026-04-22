"""Vector embedding storage and retrieval with FAISS."""

from __future__ import annotations

import json
import os
import uuid
from pathlib import Path

import faiss
import numpy as np
from PIL import Image

from services.clip_service import ClipService


class EmbeddingService:
    """Manage CLIP embeddings backed by a FAISS index."""

    def __init__(
        self,
        clip_service: ClipService,
        dim: int = 512,
        index_path: str = "./data/faiss.index",
        meta_path: str = "./data/faiss_meta.json",
    ):
        self.clip = clip_service
        self.dim = dim
        self.index_path = index_path
        self.meta_path = meta_path

        # Metadata: id → {id, filename, tags, ...}
        self.meta: dict[str, dict] = {}

        self._ensure_dirs()
        self._load_index()

    # ------------------------------------------------------------------
    # Index lifecycle
    # ------------------------------------------------------------------

    def _ensure_dirs(self) -> None:
        Path(self.index_path).parent.mkdir(parents=True, exist_ok=True)

    def _load_index(self) -> None:
        if os.path.exists(self.index_path):
            self.index = faiss.read_index(self.index_path)
        else:
            self.index = faiss.IndexFlatIP(self.dim)  # inner-product (cosine after L2 norm)

        if os.path.exists(self.meta_path):
            with open(self.meta_path, "r") as f:
                self.meta = json.load(f)

    def save(self) -> None:
        faiss.write_index(self.index, self.index_path)
        with open(self.meta_path, "w") as f:
            json.dump(self.meta, f, ensure_ascii=False, indent=2)

    # ------------------------------------------------------------------
    # Image vectors
    # ------------------------------------------------------------------

    def embed_image(self, image: Image.Image | bytes) -> np.ndarray:
        """Return CLIP image embedding (dim,)."""
        return self.clip.encode_image(image)

    def embed_image_batch(self, images: list[Image.Image | bytes]) -> np.ndarray:
        return self.clip.encode_image_batch(images)

    # ------------------------------------------------------------------
    # Text vectors
    # ------------------------------------------------------------------

    def embed_text(self, text: str) -> np.ndarray:
        """Return CLIP text embedding (dim,)."""
        return self.clip.encode_text(text)

    # ------------------------------------------------------------------
    # Index operations
    # ------------------------------------------------------------------

    def add(self, embedding: np.ndarray, metadata: dict | None = None) -> str:
        """Add one embedding to the index. Returns item id."""
        item_id = str(uuid.uuid4())
        vec = embedding.reshape(1, -1).astype("float32")
        self.index.add(vec)
        meta_entry = {"index_pos": self.index.ntotal - 1, **(metadata or {}), "id": item_id}
        self.meta[item_id] = meta_entry
        self.save()
        return item_id

    def add_batch(self, embeddings: np.ndarray, metadatas: list[dict] | None = None) -> list[str]:
        """Add a batch of embeddings. Returns list of item ids."""
        vecs = embeddings.astype("float32")
        start = self.index.ntotal
        self.index.add(vecs)
        ids = []
        for i in range(vecs.shape[0]):
            item_id = str(uuid.uuid4())
            meta_entry = {"index_pos": start + i, **((metadatas or [{}])[i] if metadatas else {}), "id": item_id}
            self.meta[item_id] = meta_entry
            ids.append(item_id)
        self.save()
        return ids

    def search(self, query: np.ndarray, top_k: int = 10) -> list[dict]:
        """Search the index; returns list of {id, score, ...metadata}."""
        if self.index.ntotal == 0:
            return []
        vec = query.reshape(1, -1).astype("float32")
        scores, indices = self.index.search(vec, min(top_k, self.index.ntotal))
        # Build reverse map: index_pos → id
        pos_to_id = {v["index_pos"]: k for k, v in self.meta.items()}
        results = []
        for score, idx in zip(scores[0], indices[0]):
            if idx < 0:
                continue
            item_id = pos_to_id.get(idx, str(idx))
            entry = self.meta.get(item_id, {})
            results.append({"id": item_id, "score": float(score), **entry})
        return results

    def remove(self, item_id: str) -> bool:
        """Remove item metadata (FAISS doesn't support true delete; mark removed)."""
        if item_id in self.meta:
            del self.meta[item_id]
            self.save()
            return True
        return False

    @property
    def size(self) -> int:
        return self.index.ntotal
