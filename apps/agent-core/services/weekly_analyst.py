import os
from typing import List, Dict, Any
from pydantic import BaseModel, Field

try:
    from langchain_openai import ChatOpenAI
    from langchain_core.prompts import PromptTemplate
    from langchain_core.output_parsers import JsonOutputParser
except ImportError:
    pass

LLM_API_KEY = os.getenv("LLM_API_KEY", "")
LLM_BASE_URL = os.getenv("LLM_BASE_URL", "https://api.openai.com/v1")

class WeeklyInsights(BaseModel):
    summary: str = Field(description="A friendly, encouraging summary of the week's performance directly addressing the child.")
    strengths: List[str] = Field(description="List of top 3 strengths or tasks well done.")
    areas_for_improvement: List[str] = Field(description="Constructive feedback or areas to focus on next week.")
    psychological_insight: str = Field(description="Growth-mindset oriented observation about the child's behavior patterns.")

# Fallback fake ML runner
def generate_weekly_report_mock(days_data: List[Dict[str, Any]]) -> Dict[str, Any]:
    total_tasks = 0
    completed_tasks = 0
    for day in days_data:
        tasks = day.get('tasks', [])
        total_tasks += len(tasks)
        completed_tasks += sum(1 for t in tasks if t.get('completed', False))
    
    return {
        "summary": f"Great job this week! You tackled {total_tasks} tasks and completed {completed_tasks} of them.",
        "strengths": ["Consistent effort", "Ready to tackle new challenges"],
        "areas_for_improvement": ["Try to finish pending tasks before starting new ones"],
        "psychological_insight": "Your resilience is showing! Keep growing your brain by attempting hard things.",
        "raw_metric_total": total_tasks,
        "raw_metric_completed": completed_tasks
    }

def generate_weekly_report(days_data: List[Dict[str, Any]]) -> dict:
    # If no real key, fallback to mock to prevent blocking dev
    if not LLM_API_KEY:
        return generate_weekly_report_mock(days_data)

    try:
        llm = ChatOpenAI(
            api_key=LLM_API_KEY,
            base_url=LLM_BASE_URL,
            temperature=0.4,
            model="gpt-4o-mini",
        )
        parser = JsonOutputParser(pydantic_object=WeeklyInsights)

        prompt = PromptTemplate(
            template="""You are an encouraging and perceptive AI companion for a child.
Analyze their task completions over the past 7 days and generate a weekly report.

Here is their task data over the week (Markdown Checklist Format):
{days_data}

{format_instructions}
""",
            input_variables=["days_data"],
            partial_variables={"format_instructions": parser.get_format_instructions()}
        )

        chain = prompt | llm | parser
        return chain.invoke({"days_data": str(days_data)})
    except Exception as e:
        print(f"Error calling LLM: {e}")
        return generate_weekly_report_mock(days_data)
