"""Application configuration."""

from pydantic_settings import BaseSettings
from functools import lru_cache


class Settings(BaseSettings):
    # Model
    clip_model_name: str = "openai/clip-vit-base-patch32"
    device: str = "cuda"  # auto-detected in practice
    model_cache_dir: str = "./models"

    # FAISS
    faiss_index_path: str = "./data/faiss.index"
    faiss_meta_path: str = "./data/faiss_meta.json"
    embedding_dim: int = 512

    # Server
    host: str = "0.0.0.0"
    port: int = 8000
    log_level: str = "info"

    # Classification
    default_top_k: int = 5
    style_labels: list[str] = [
        "风景", "人像", "街拍", "美食", "宠物", "建筑", "夜景", "微距",
        "landscape", "portrait", "street photography", "food", "pet",
        "architecture", "night scene", "macro photography",
    ]

    model_config = {"env_prefix": "ML_", "env_file": ".env"}


@lru_cache()
def get_settings() -> Settings:
    return Settings()
