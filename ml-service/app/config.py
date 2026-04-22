"""Application configuration."""

from pydantic_settings import BaseSettings
from functools import lru_cache


class Settings(BaseSettings):
    # Model
    clip_model_name: str = "OFA-Sys/chinese-clip-vit-base-patch16"
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
        # 场景
        "城市街景", "古建筑", "现代建筑", "公园园林", "海滨沙滩",
        "山川湖泊", "机场车站", "室内场景", "夜景灯光", "道路桥梁",
        "校园场景", "商业街市", "历史文化街区", "自然风光", "城市天际线",
        # 主体
        "花朵植物", "树木森林", "天空云彩", "水面倒影", "交通工具",
        "飞机航空", "动物宠物", "人物合影", "儿童", "演出舞台",
        "会议讲座", "美食餐饮", "体育运动", "园林景观", "喷泉雕塑",
        # 视觉元素
        "蓝天白云", "日出日落", "雪景冰霜", "雨雾天气", "逆光剪影",
        "微距特写", "广角视野", "对称构图", "倒影镜面", "红墙黄瓦",
        "霓虹灯光", "烟火表演", "人群聚集", "书法艺术", "国旗旗帜",
        # 特定建筑/地点
        "宫殿建筑", "庙宇祠堂", "塔楼钟楼", "城墙城门", "亭台楼阁",
        "码头港口", "摩天轮", "高楼大厦", "别墅住宅", "农田乡村",
    ]

    model_config = {"env_prefix": "ML_", "env_file": ".env"}


@lru_cache()
def get_settings() -> Settings:
    return Settings()
