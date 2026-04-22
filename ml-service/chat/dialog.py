"""Dialog state management for photo search chat."""

from __future__ import annotations

import time
from dataclasses import dataclass, field
from enum import Enum

from chat.intent import Intent, IntentType, recognize_intent
from chat.executor import ActionExecutor


class DialogState(str, Enum):
    IDLE = "idle"
    SEARCHING = "searching"
    CLASSIFYING = "classifying"
    FILTERING = "filtering"
    SUGGESTING = "suggesting"


@dataclass
class Turn:
    role: str  # "user" | "assistant"
    content: str
    timestamp: float = field(default_factory=time.time)


@dataclass
class DialogSession:
    session_id: str
    state: DialogState = DialogState.IDLE
    history: list[Turn] = field(default_factory=list)
    context: dict = field(default_factory=dict)  # last results, filters, etc.
    created_at: float = field(default_factory=time.time)


class DialogManager:
    """Manages multi-turn dialog sessions."""

    def __init__(self, executor: ActionExecutor):
        self.executor = executor
        self.sessions: dict[str, DialogSession] = {}

    def get_or_create_session(self, session_id: str) -> DialogSession:
        if session_id not in self.sessions:
            self.sessions[session_id] = DialogSession(session_id=session_id)
        return self.sessions[session_id]

    def process(self, session_id: str, user_message: str) -> dict:
        """Process a user message and return the assistant response."""
        session = self.get_or_create_session(session_id)

        # Record user turn
        session.history.append(Turn(role="user", content=user_message))

        # Recognize intent
        intent = recognize_intent(user_message)

        # Execute action based on intent
        result = self.executor.execute(intent, session.context)

        # Update session state
        session.state = result.get("next_state", DialogState.IDLE)
        if result.get("context_update"):
            session.context.update(result["context_update"])

        # Build response
        response_text = result.get("response", "抱歉，我不太理解你的意思。你可以试试「搜索日落照片」或「分类这张图片」。")
        session.history.append(Turn(role="assistant", content=response_text))

        return {
            "session_id": session_id,
            "intent": intent.type.value,
            "confidence": intent.confidence,
            "response": response_text,
            "data": result.get("data"),
            "state": session.state.value,
        }

    def clear_session(self, session_id: str) -> bool:
        if session_id in self.sessions:
            del self.sessions[session_id]
            return True
        return False
