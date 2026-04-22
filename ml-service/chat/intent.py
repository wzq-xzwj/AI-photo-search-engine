"""Intent recognition for photo search chat."""

from __future__ import annotations

import re
from enum import Enum
from dataclasses import dataclass, field


class IntentType(str, Enum):
    SEARCH = "search"          # 用文字搜索图片
    CLASSIFY = "classify"      # 分类图片
    FILTER = "filter"          # 筛选
    ACTION = "action"          # 操作（删除、收藏等）
    SUGGEST = "suggest"        # 建议/推荐
    HELP = "help"              # 帮助
    UNKNOWN = "unknown"


# Intent keyword patterns (Chinese + English)
_PATTERNS: dict[IntentType, list[str]] = {
    IntentType.SEARCH: [
        r"搜索|查找|找|搜索一下|有没有|搜|search|find|look for|show me",
        r"帮我找|帮我搜|我想看|我想找",
    ],
    IntentType.CLASSIFY: [
        r"分类|这是什么|什么类型|类别|classify|categorize|what kind",
        r"这张.*是什么|这张照片",
    ],
    IntentType.FILTER: [
        r"筛选|过滤|只要|只看|只要|filter|only|just",
        r"去掉|排除|不要|exclude",
    ],
    IntentType.ACTION: [
        r"删除|删掉|移除|收藏|喜欢|保存|delete|remove|save|favorite|like",
        r"标记|打标签|tag|label",
    ],
    IntentType.SUGGEST: [
        r"推荐|建议|有什么好|suggest|recommend|what.*good",
        r"你觉得|你认为|帮我选|帮我挑",
    ],
    IntentType.HELP: [
        r"帮助|怎么用|怎么操作|help|how to|how do",
        r"功能|你能做什么|what can you do",
    ],
}


@dataclass
class Intent:
    type: IntentType
    confidence: float
    query: str = ""
    parameters: dict = field(default_factory=dict)


def recognize_intent(text: str) -> Intent:
    """Rule-based intent recognition with keyword matching."""
    text_lower = text.strip().lower()

    best_type = IntentType.UNKNOWN
    best_score = 0.0

    for intent_type, patterns in _PATTERNS.items():
        for pat in patterns:
            if re.search(pat, text_lower):
                # Simple scoring: first match wins, longer match = higher confidence
                match = re.search(pat, text_lower)
                score = min(0.95, 0.6 + len(match.group()) * 0.03)
                if score > best_score:
                    best_score = score
                    best_type = intent_type
                    break

    # Extract search query: strip intent keywords
    query = text.strip()
    if best_type == IntentType.SEARCH:
        query = re.sub(r"^(搜索|查找|找|搜索一下|帮我找|帮我搜|我想看|我想找|search|find|look for|show me)\s*", "", query).strip()
        if not query:
            query = text.strip()

    return Intent(type=best_type, confidence=best_score, query=query)
