import os
import json
from langchain_openai import ChatOpenAI
from langchain_core.prompts import PromptTemplate

# Load configurations securely
LLM_API_KEY = os.getenv("LLM_API_KEY", "")
LLM_BASE_URL = os.getenv("LLM_BASE_URL", "https://api.openai.com/v1")

# Initialize Chat Model
def get_llm():
    return ChatOpenAI(
        api_key=LLM_API_KEY,
        base_url=LLM_BASE_URL,
        model="gpt-4o", # Can be customized based on provider
        temperature=0.1,
    )

def parse_parent_input(raw_text: str) -> dict:
    """
    Parses unstructured text from parents into structured task JSON.
    """
    llm = get_llm()
    
    prompt = PromptTemplate.from_template("""
    你是一个专为儿童设计的 AI 家庭学伴引擎的后台解析核心。
    你的任务是将家长发来的【随意口语化作业记录词句】，精准提取为结构化的任务列表。

    【家长留言】
    {raw_text}

    【要求】
    1. 你只能返回 JSON 格式结果，不要输出任何额外的思考过程及Markdown包裹（不要 ```json ）。
    2. JSON 结构必须为: 
       {{
          "status": "success",
          "data": [
            {{"subject": "学科名称 (如 语文、数学、英语、兴趣)", "title": "任务的精简提炼 (例如: 听写第一单元单词, 口算三十题)", "type": "homework"}}
          ]
       }}
    3. 如果家长说的不是学习任务或是闲聊，请将 "status" 设置为 "failed"，并清空 "data"。
    """)
    
    chain = prompt | llm
    
    try:
        response = chain.invoke({"raw_text": raw_text})
        # Try parsing string to dictionary directly assuming no markdown ticks 
        # (in production, use LLM features like function calling or StructuredOutputParser)
        result_str = response.content.strip()
        if result_str.startswith("```json"):
            result_str = result_str[7:-3].strip()
            
        parsed_data = json.loads(result_str)
        return parsed_data
    except Exception as e:
        print(f"LLM Parsing error: {e}")
        return {"status": "error", "message": str(e), "data": []}
