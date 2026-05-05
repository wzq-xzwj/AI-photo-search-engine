"""FastAPI routes for the ML service."""

from __future__ import annotations

import io
import os

import torch
from fastapi import APIRouter, File, UploadFile, Depends, HTTPException, Form
from PIL import Image

from api.schemas import (
    ExtractResponse,
    ClassifyResponse,
    ClassifyBatchResponse,
    ClassifyItem,
    SimilarityResponse,
    SimilarityItem,
    TextEmbedRequest,
    TextEmbedResponse,
    ChatRequest,
    ChatResponse,
    HealthResponse,
    FaceDetectRequest,
    FaceDetectResponse,
    FaceInfo,
    FaceLocation,
    FaceThumbnailRequest,
)
from services.clip_service import ClipService
from services.classifier import ImageClassifier
from services.classifier_v2 import ClassifierV2
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
    
    # Check face detection availability
    face_available = False
    insightface_available = False
    try:
        from insightface_service import get_face_app
        get_face_app()
        insightface_available = True
        face_available = True
    except ImportError:
        # Fallback to face_recognition
        try:
            from face_detection import detector
            face_available = detector.available
        except ImportError:
            pass
    
    return HealthResponse(
        status="ok",
        model=settings.clip_model_name,
        device="cuda" if torch.cuda.is_available() else "cpu",
        index_size=emb.size,
        face_detection_available=face_available,
    )


# ------------------------------------------------------------------
# Face Detection
# ------------------------------------------------------------------

@router.post("/faces/detect", response_model=FaceDetectResponse)
async def detect_faces(
    body: FaceDetectRequest,
):
    """检测照片中的人脸 (InsightFace)"""
    try:
        from insightface_service import detect_faces_insightface
    except ImportError as e:
        raise HTTPException(status_code=503, detail=f"InsightFace not available: {e}")
    
    image_path = body.image_path
    if not os.path.exists(image_path):
        raise HTTPException(status_code=404, detail=f"Image not found: {image_path}")
    
    try:
        faces = detect_faces_insightface(image_path, min_det_score=0.5, min_size=60)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Face detection failed: {e}")
    
    return FaceDetectResponse(
        image_path=image_path,
        face_count=len(faces),
        faces=[
            FaceInfo(
                index=f["index"],
                location=FaceLocation(**f["location"]),
                encoding=f["encoding"],
                confidence=f["confidence"],
            )
            for f in faces
        ],
        available=True,
    )


@router.post("/faces/detect-file", response_model=FaceDetectResponse)
async def detect_faces_file(
    file: UploadFile = File(...),
):
    """从上传文件检测人脸 (InsightFace)"""
    try:
        from insightface_service import detect_faces_insightface
    except ImportError as e:
        raise HTTPException(status_code=503, detail=f"InsightFace not available: {e}")
    
    if not file.content_type or not file.content_type.startswith("image/"):
        raise HTTPException(status_code=400, detail="File must be an image")
    
    data = await file.read()
    
    # Save to temp file for InsightFace
    import tempfile
    with tempfile.NamedTemporaryFile(suffix=".jpg", delete=False) as tmp:
        tmp.write(data)
        tmp_path = tmp.name
    
    try:
        faces = detect_faces_insightface(tmp_path, min_det_score=0.5, min_size=60)
    finally:
        os.unlink(tmp_path)
    
    return FaceDetectResponse(
        image_path=file.filename or "uploaded",
        face_count=len(faces),
        faces=[
            FaceInfo(
                index=f["index"],
                location=FaceLocation(**f["location"]),
                encoding=f["encoding"],
                confidence=f["confidence"],
            )
            for f in faces
        ],
        available=True,
    )


@router.post("/faces/thumbnail")
async def extract_face_thumbnail(
    body: FaceThumbnailRequest,
):
    """提取人脸缩略图 (InsightFace)"""
    try:
        from insightface_service import extract_thumbnail
    except ImportError as e:
        raise HTTPException(status_code=503, detail=f"InsightFace not available: {e}")
    
    image_path = body.image_path
    if not os.path.exists(image_path):
        raise HTTPException(status_code=404, detail=f"Image not found: {image_path}")
    
    try:
        thumbnail_bytes = extract_thumbnail(
            image_path,
            body.location.model_dump(),
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Thumbnail extraction failed: {e}")
    
    from fastapi.responses import Response
    return Response(content=thumbnail_bytes, media_type="image/jpeg")


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


@router.post("/classify/batch", response_model=ClassifyBatchResponse)
async def classify_batch(
    files: list[UploadFile] = File(...),
    top_k: int = 5,
    classifier: ImageClassifier = Depends(get_classifier),
):
    if not files:
        raise HTTPException(status_code=400, detail="No files provided")
    data_list = []
    for file in files:
        if not file.content_type or not file.content_type.startswith("image/"):
            raise HTTPException(status_code=400, detail="All files must be images")
        data_list.append(await file.read())
    batch_results = classifier.classify_batch(data_list, top_k=top_k)
    return ClassifyBatchResponse(
        results=[
            [ClassifyItem(label=r.label, score=r.score) for r in row]
            for row in batch_results
        ]
    )


# ------------------------------------------------------------------
# Similarity (raw cosine similarity, no softmax)
# ------------------------------------------------------------------

@router.post("/similarity/batch", response_model=SimilarityResponse)
async def similarity_batch(
    files: list[UploadFile] = File(...),
    labels: str = Form(...),
    top_k: int = Form(5),
    classifier: ImageClassifier = Depends(get_classifier),
):
    import json
    if not files:
        raise HTTPException(status_code=400, detail="No files provided")
    data_list = []
    for file in files:
        if not file.content_type or not file.content_type.startswith("image/"):
            raise HTTPException(status_code=400, detail="All files must be images")
        data_list.append(await file.read())
    label_list = json.loads(labels)
    if not label_list:
        raise HTTPException(status_code=400, detail="Labels required")
    batch_results = classifier.similarity_batch(data_list, label_list, top_k=top_k)
    return SimilarityResponse(
        results=[
            [SimilarityItem(label=r.label, score=r.score) for r in row]
            for row in batch_results
        ]
    )


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
# V2 Similarity (Prompt templates + threshold filtering)
# ------------------------------------------------------------------

@router.post("/similarity/batch/v2", response_model=SimilarityResponse)
async def similarity_batch_v2(
    files: list[UploadFile] = File(...),
    labels: str = Form(...),
    top_k: int = Form(8),
    min_score: float = Form(0.18),
    clip: ClipService = Depends(get_clip_service),
):
    """V2 分类：Prompt 模板编码 + 阈值过滤，Chinese CLIP 精度大幅提升"""
    import json
    if not files:
        raise HTTPException(status_code=400, detail="No files provided")
    data_list = []
    for file in files:
        if not file.content_type or not file.content_type.startswith("image/"):
            raise HTTPException(status_code=400, detail="All files must be images")
        data_list.append(await file.read())
    label_list = json.loads(labels)
    if not label_list:
        raise HTTPException(status_code=400, detail="Labels required")

    classifier_v2 = ClassifierV2(
        clip_service=clip,
        labels=label_list,
        min_score=min_score,
        top_k=top_k,
        use_prompt_template=True,
    )
    batch_results = classifier_v2.similarity_batch(data_list, label_list, top_k=top_k, min_score=min_score)
    return SimilarityResponse(
        results=[
            [SimilarityItem(label=r.label, score=r.score) for r in row]
            for row in batch_results
        ]
    )


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
