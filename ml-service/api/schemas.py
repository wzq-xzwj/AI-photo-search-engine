"""Pydantic request / response models."""

from __future__ import annotations

from pydantic import BaseModel, Field


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
