"""Dependency injection for FastAPI."""

import torch
from functools import lru_cache

from services.clip_service import ClipService
from services.classifier import ImageClassifier
from services.embedding import EmbeddingService
from chat.executor import ActionExecutor
from chat.dialog import DialogManager
from app.config import get_settings


@lru_cache()
def get_clip_service() -> ClipService:
    settings = get_settings()
    # macOS 不支持 CUDA，直接使用 CPU
    device = "cpu"
    return ClipService(model_name=settings.clip_model_name, device=device)


@lru_cache()
def get_classifier() -> ImageClassifier:
    clip = get_clip_service()
    settings = get_settings()
    return ImageClassifier(clip_service=clip, labels=settings.style_labels)


@lru_cache()
def get_embedding_service() -> EmbeddingService:
    clip = get_clip_service()
    settings = get_settings()
    return EmbeddingService(
        clip_service=clip,
        dim=settings.embedding_dim,
        index_path=settings.faiss_index_path,
        meta_path=settings.faiss_meta_path,
    )


@lru_cache()
def get_executor() -> ActionExecutor:
    return ActionExecutor(
        classifier=get_classifier(),
        embedding_service=get_embedding_service(),
        clip_service=get_clip_service(),
    )


@lru_cache()
def get_dialog_manager() -> DialogManager:
    return DialogManager(executor=get_executor())
