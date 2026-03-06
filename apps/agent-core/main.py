from fastapi import FastAPI
import uvicorn
from api.routes import router

app = FastAPI(
    title="StudyClaw Agent Core",
    description="The AI Brain for Task Parsing and Emotional Companionship",
    version="1.0.0"
)

app.include_router(router, prefix="/api/v1")

@app.get("/ping")
async def ping():
    return {"message": "Agent Core is alive"}

if __name__ == "__main__":
    import os
    port = int(os.getenv("AGENT_PORT", 8000))
    uvicorn.run("main:app", host="0.0.0.0", port=port, reload=True)
