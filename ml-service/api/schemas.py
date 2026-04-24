"""Pydantic request / response models."""

from __future__ import annotations

from pydantic import BaseModel, Field
from typing import List, Optional, Dict, Any


# ---- Requests ----

class ExtractRequest(BaseModel):
    image_url: str | None = None
    # Image bytes are handled via file upload


class ClassifyRequest(BaseModel):
    top_k: int = Field(default=5, ge=1, le=20)
    labels: list[str] | None = None


class TextEmbedRequest(BaseModel):
    text: str


class ChatRequest(BaseModel):
    session_id: str = "default"
    message: str


# ---- Face Detection ----

class FaceDetectRequest(BaseModel):
    image_path: str


class FaceLocation(BaseModel):
    top: int
    right: int
    bottom: int
    left: int


class FaceInfo(BaseModel):
    index: int
    location: FaceLocation
    encoding: List[float]
    confidence: float = 0.99


class FaceDetectResponse(BaseModel):
    image_path: str
    face_count: int
    faces: List[FaceInfo]
    available: bool = True


class FaceThumbnailRequest(BaseModel):
    image_path: str
    location: FaceLocation
    size: int = Field(default=150, ge=50, le=500)


# ---- Responses ----

class ExtractResponse(BaseModel):
    embedding: list[float]
    dim: int


class ClassifyItem(BaseModel):
    label: str
    score: float


class ClassifyResponse(BaseModel):
    results: list[ClassifyItem]


class ClassifyBatchResponse(BaseModel):
    results: list[list[ClassifyItem]]


class SimilarityRequest(BaseModel):
    labels: list[str]
    top_k: int = Field(default=5, ge=1, le=50)


class SimilarityItem(BaseModel):
    label: str
    score: float


class SimilarityResponse(BaseModel):
    results: list[list[SimilarityItem]]


class TextEmbedResponse(BaseModel):
    embedding: list[float]
    dim: int


class ChatResponse(BaseModel):
    session_id: str
    intent: str
    confidence: float
    response: str
    data: list | dict | None = None
    state: str


class HealthResponse(BaseModel):
    status: str = "ok"
    model: str
    device: str
    index_size: int
    face_detection_available: bool = False
