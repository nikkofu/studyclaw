from fastapi import APIRouter
from pydantic import BaseModel
import sys
import os

# Ensure the current directory is in the path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
try:
    from services.llm_parser import parse_parent_input
    from services.weekly_analyst import generate_weekly_report
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

class WeeklyReportRequest(BaseModel):
    days_data: list

@router.post("/internal/analyze/weekly")
async def analyze_weekly_stats(request: WeeklyReportRequest):
    """
    Called by Golang API Server to generate a weekly report based on markdown logs.
    """
    insights = generate_weekly_report(request.days_data)
    return insights
