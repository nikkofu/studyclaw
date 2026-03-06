from fastapi import APIRouter
from pydantic import BaseModel
from apps.agent_core.services.llm_parser import parse_parent_input
# For relative imports when running from apps/agent-core root, Python might need adjustment
# Adjust import logic here to be safer:
import sys
import os
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
try:
    from services.llm_parser import parse_parent_input
except ImportError:
    pass

router = APIRouter()

class ParseRequest(BaseModel):
    raw_text: str

@router.post("/internal/parse")
async def parse_task_text(request: ParseRequest):
    """
    Internal API for Golang Business Server to convert raw text to tasks.
    """
    structured_data = parse_parent_input(request.raw_text)
    return structured_data
