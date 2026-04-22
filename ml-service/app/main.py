"""FastAPI application entry point."""

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from api.routes import router
from app.config import get_settings

settings = get_settings()

app = FastAPI(
    title="AI Photo Search ML Service",
    description="CLIP-powered image feature extraction, classification, and conversational search",
    version="1.0.0",
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(router, prefix="/api/v1")


@app.on_event("startup")
async def startup():
    """Pre-load models on startup."""
    import logging
    logging.basicConfig(level=getattr(logging, settings.log_level.upper(), logging.INFO))
    logger = logging.getLogger(__name__)
    logger.info("Loading CLIP model: %s", settings.clip_model_name)

    # Trigger lazy loading
    from app.dependencies import get_clip_service
    get_clip_service()
    logger.info("Model loaded successfully")


if __name__ == "__main__":
    import uvicorn
    uvicorn.run("app.main:app", host=settings.host, port=settings.port, reload=True)
