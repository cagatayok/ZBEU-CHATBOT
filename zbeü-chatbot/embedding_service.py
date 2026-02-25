from sentence_transformers import SentenceTransformer
from fastapi import FastAPI
from pydantic import BaseModel
import uvicorn

app = FastAPI()

# Load model (384 dimensions, fast and lightweight)
model = SentenceTransformer('all-MiniLM-L6-v2')

class EmbedRequest(BaseModel):
    text: str

@app.post("/embed")
async def create_embedding(request: EmbedRequest):
    """Generate embedding for given text"""
    embedding = model.encode(request.text)
    return {"embedding": embedding.tolist()}

@app.get("/health")
async def health():
    return {"status": "ok"}

if __name__ == "__main__":
    print("🚀 Embedding Service başlatılıyor...")
    print("📊 Model: all-MiniLM-L6-v2 (384 dimensions)")
    print("🌐 Endpoint: http://localhost:8000/embed")
    uvicorn.run(app, host="0.0.0.0", port=8000)
