"""FastAPI routes for the ML service."""

from __future__ import annotations

import io

import torch
from fastapi import APIRouter, File, UploadFile, Depends, HTTPException
from PIL import Image

from api.schemas import (
    ExtractResponse,
    ClassifyRequest,
    ClassifyResponse,
    ClassifyItem,
    TextEmbedRequest,
    TextEmbedResponse,
    ChatRequest,
    ChatResponse,
    HealthResponse,
)
from services.clip_service import ClipService
from services.classifier import ImageClassifier
from services.embedding import EmbeddingService
from chat.dialog import DialogManager
from app.dependencies import (
    get_clip_service,
    get_classifier,
    get_embedding_service,
    get_dialog_manager,
)
from app.config import get_settings

router = APIRouter()


# ------------------------------------------------------------------
# Health
# ------------------------------------------------------------------

@router.get("/health", response_model=HealthResponse)
async def health(
    clip: ClipService = Depends(get_clip_service),
    emb: EmbeddingService = Depends(get_embedding_service),
):
    settings = get_settings()
    return HealthResponse(
        status="ok",
        model=settings.clip_model_name,
        device="cuda" if torch.cuda.is_available() else "cpu",
        index_size=emb.size,
    )


# ------------------------------------------------------------------
# Extract image features
# ------------------------------------------------------------------

@router.post("/extract", response_model=ExtractResponse)
async def extract_features(
    file: UploadFile = File(...),
    emb: EmbeddingService = Depends(get_embedding_service),
):
    if not file.content_type or not file.content_type.startswith("image/"):
        raise HTTPException(status_code=400, detail="File must be an image")
    data = await file.read()
    vec = emb.embed_image(data)
    return ExtractResponse(embedding=vec.tolist(), dim=len(vec))


# ------------------------------------------------------------------
# Classify image
# ------------------------------------------------------------------

@router.post("/classify", response_model=ClassifyResponse)
async def classify_image(
    file: UploadFile = File(...),
    top_k: int = 5,
    classifier: ImageClassifier = Depends(get_classifier),
):
    if not file.content_type or not file.content_type.startswith("image/"):
        raise HTTPException(status_code=400, detail="File must be an image")
    data = await file.read()
    results = classifier.classify(data, top_k=top_k)
    return ClassifyResponse(results=[ClassifyItem(label=r.label, score=r.score) for r in results])


# ------------------------------------------------------------------
# Text embedding
# ------------------------------------------------------------------

@router.post("/embed/text", response_model=TextEmbedResponse)
async def embed_text(
    body: TextEmbedRequest,
    emb: EmbeddingService = Depends(get_embedding_service),
):
    vec = emb.embed_text(body.text)
    return TextEmbedResponse(embedding=vec.tolist(), dim=len(vec))


# ------------------------------------------------------------------
# Chat
# ------------------------------------------------------------------

@router.post("/chat", response_model=ChatResponse)
async def chat(
    body: ChatRequest,
    dm: DialogManager = Depends(get_dialog_manager),
):
    result = dm.process(body.session_id, body.message)
    return ChatResponse(**result)
