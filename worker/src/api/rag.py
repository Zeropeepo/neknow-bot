import asyncio
import cohere
from fastapi import FastAPI, HTTPException
from fastapi.responses import StreamingResponse

from pydantic import BaseModel
from typing import List, AsyncGenerator
from openai import AsyncOpenAI

# Config
from src.config import settings
from src.processor.embedder import embed_texts
from src.storage.vector_store import hybrid_search


app = FastAPI()

openai_client = AsyncOpenAI(api_key=settings.openai_api_key)
cohere_client = cohere.Client(settings.cohere_api_key)

class HistoryMessage(BaseModel):
    role: str  
    content: str

class RAGRequest(BaseModel):
    bot_id: str
    system_prompt: str
    query: str
    history: List[HistoryMessage]

async def stream_response(request: RAGRequest) -> AsyncGenerator[str, None]:

    embeddings = await embed_texts([request.query])
    query_embedding = embeddings[0]

    chunks = await hybrid_search(
        bot_id=request.bot_id,
        query_embedding=query_embedding,
        query_text=request.query,
        top_k=settings.top_k_retrieve,
    )

    if chunks:
        rerank_response = await cohere_client.rerank(
            model="rerank-v3.5",
            query=request.query,
            documents=[chunk["content"] for chunk in chunks],
            top_n=settings.top_k_rerank,
        )
        top_chunks = [
            chunks[r.index]["content"]
            for r in rerank_response.results
        ]
    else:
        top_chunks = []


    context = "\n\n---\n\n".join(top_chunks) if top_chunks else "No relevant context found."

    messages = [
        {
            "role": "system",
            "content": f"{request.system_prompt}\n\nContext:\n{context}"
        },
    ]

    for msg in request.history:
        messages.append({
            "role": msg.role,
            "content": msg.content
        })

    messages.append({ "role": "user", "content": request.query })

    stream = await openai_client.chat.completions.create(
        model=settings.chat_model,
        messages=messages,
        stream=True,
        temperature=0.7,
        max_tokens=1024,
    )

    async for chunk in stream:

        delta = chunk.choices[0].delta.content
        if delta:
            import json
            yield f"data: {json.dumps({'delta': delta})}\n\n"

    
    yield "data: [DONE]\n\n"

@app.post("/rag")
async def rag_stream(request: RAGRequest):

    return StreamingResponse(
        stream_response(request),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
        },
    )

@app.get("/health")
async def health_check():
    return {"status": "ok"}