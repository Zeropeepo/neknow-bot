import json
import cohere
from fastapi import FastAPI
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
from typing import List, AsyncGenerator
from google import genai
from google.genai import types

from src.config import settings
from src.processor.embedder import embed_query
from src.storage.vector_store import hybrid_search


app = FastAPI()

gemini_client = genai.Client(api_key=settings.gemini_api_key)
cohere_client = cohere.AsyncClientV2(api_key=settings.cohere_api_key)


class HistoryMessage(BaseModel):
    role: str
    content: str


class RAGRequest(BaseModel):
    bot_id: str
    system_prompt: str
    query: str
    history: List[HistoryMessage] = []


async def stream_response(request: RAGRequest) -> AsyncGenerator[str, None]:

    # 1. Embed query
    query_embedding = await embed_query(request.query)

    # 2. Hybrid search
    chunks = await hybrid_search(
        bot_id=request.bot_id,
        query_embedding=query_embedding,
        query_text=request.query,
        top_k=settings.top_k_retrieve,
    )

    # 3. Cohere rerank
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

    # 4. Build prompt
    context = "\n\n---\n\n".join(top_chunks) if top_chunks else "No relevant context found."

    system_instruction = f"{request.system_prompt}\n\nContext:\n{context}"

    contents = []
    for msg in request.history:
        contents.append(types.Content(
            role="user" if msg.role == "user" else "model",
            parts=[types.Part(text=msg.content)]
        ))
    contents.append(types.Content(
        role="user",
        parts=[types.Part(text=request.query)]
    ))

    # Stream dari Gemini
    async for chunk in await gemini_client.aio.models.generate_content_stream(
        model=settings.chat_model,
        contents=contents,
        config=types.GenerateContentConfig(
            system_instruction=system_instruction,
            temperature=0.7,
            max_output_tokens=1024,
        )
    ):
        if chunk.text:
            yield f"data: {json.dumps({'content': chunk.text})}\n\n"

    yield "data: [DONE]\n\n"


@app.post("/rag/stream")
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