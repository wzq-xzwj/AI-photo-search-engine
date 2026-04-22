"""Action executor — maps intents to service calls."""

from __future__ import annotations

from chat.intent import Intent, IntentType


class ActionExecutor:
    """Execute actions based on recognised intents."""

    def __init__(self, classifier, embedding_service, clip_service):
        self.classifier = classifier
        self.embedding = embedding_service
        self.clip = clip_service

    def execute(self, intent: Intent, context: dict) -> dict:
        """Dispatch intent to handler; return {response, data, next_state, context_update}."""
        handlers = {
            IntentType.SEARCH: self._handle_search,
            IntentType.CLASSIFY: self._handle_classify,
            IntentType.FILTER: self._handle_filter,
            IntentType.ACTION: self._handle_action,
            IntentType.SUGGEST: self._handle_suggest,
            IntentType.HELP: self._handle_help,
            IntentType.UNKNOWN: self._handle_unknown,
        }
        handler = handlers.get(intent.type, self._handle_unknown)
        return handler(intent, context)

    # ------------------------------------------------------------------
    # Handlers
    # ------------------------------------------------------------------

    def _handle_search(self, intent: Intent, context: dict) -> dict:
        query = intent.query
        if not query:
            return {"response": "你想搜索什么呢？例如：搜索日落照片"}

        text_feat = self.embedding.embed_text(query)
        results = self.embedding.search(text_feat, top_k=10)

        if not results:
            return {
                "response": f"没有找到与「{query}」相关的照片。试试其他关键词？",
                "data": [],
            }

        # Format response
        lines = [f"找到 {len(results)} 张与「{query}」相关的照片："]
        for i, r in enumerate(results[:5], 1):
            name = r.get("filename", r.get("id", "?"))
            lines.append(f"  {i}. {name} (相似度: {r['score']:.2%})")

        return {
            "response": "\n".join(lines),
            "data": results,
            "next_state": "searching",
            "context_update": {"last_results": results, "last_query": query},
        }

    def _handle_classify(self, intent: Intent, context: dict) -> dict:
        image_bytes = context.get("pending_image")
        if image_bytes is None:
            return {"response": "请先上传一张图片，我来帮你分类。"}

        results = self.classifier.classify(image_bytes, top_k=3)
        labels_str = ", ".join(f"{r.label}({r.score:.1%})" for r in results)
        return {
            "response": f"这张图片最可能属于：{labels_str}",
            "data": [{"label": r.label, "score": r.score} for r in results],
        }

    def _handle_filter(self, intent: Intent, context: dict) -> dict:
        query = intent.query
        if not query:
            return {"response": "你想按什么条件筛选？例如：只要风景照"}

        last_results = context.get("last_results", [])
        if not last_results:
            return {"response": "还没有搜索结果，请先搜索照片。"}

        # Use CLIP to re-rank by filter query
        text_feat = self.embedding.embed_text(query)
        filtered = []
        for r in last_results:
            # Simple: keep items whose metadata tags match
            tags = r.get("tags", [])
            if isinstance(tags, str):
                tags = [tags]
            if any(query in t for t in tags):
                filtered.append(r)

        if not filtered:
            return {"response": f"在上次结果中没有找到符合「{query}」条件的照片。"}

        lines = [f"筛选出 {len(filtered)} 张符合「{query}」的照片："]
        for i, r in enumerate(filtered[:5], 1):
            lines.append(f"  {i}. {r.get('filename', r.get('id', '?'))}")

        return {
            "response": "\n".join(lines),
            "data": filtered,
            "context_update": {"last_results": filtered},
        }

    def _handle_action(self, intent: Intent, context: dict) -> dict:
        query = intent.query
        if any(kw in query for kw in ["删除", "delete", "remove"]):
            return {"response": "删除操作需要在前端界面中选择具体照片后执行。"}
        if any(kw in query for kw in ["收藏", "喜欢", "save", "favorite", "like"]):
            return {"response": "收藏操作需要在前端界面中选择具体照片后执行。"}
        return {"response": "支持的操作：收藏、删除、标记。请在前端选择照片后操作。"}

    def _handle_suggest(self, intent: Intent, context: dict) -> dict:
        size = self.embedding.size
        if size == 0:
            return {"response": "图库中还没有照片，请先上传一些照片。"}

        return {
            "response": (
                f"你的图库中有 {size} 张照片。\n"
                "试试这些搜索：\n"
                "  • 搜索「海边日落」\n"
                "  • 搜索「城市夜景」\n"
                "  • 搜索「美食照片」\n"
                "  • 搜索「宠物」\n"
                "也可以上传图片让 AI 自动分类哦！"
            ),
        }

    def _handle_help(self, intent: Intent, context: dict) -> dict:
        return {
            "response": (
                "AI 照片搜索引擎使用指南：\n"
                "  🔍 搜索：「搜索日落」「找找海边的照片」\n"
                "  🏷️ 分类：上传图片后说「分类这张图」\n"
                "  🔧 筛选：搜索后说「只要风景照」\n"
                "  ⭐ 收藏：「收藏这张照片」\n"
                "  💡 推荐：「推荐一些照片」\n"
                "\n支持中英文输入，AI 会理解你的意图！"
            ),
        }

    def _handle_unknown(self, intent: Intent, context: dict) -> dict:
        return {
            "response": "我不太理解你的意思。试试说「搜索」「分类」或「帮助」？",
        }
