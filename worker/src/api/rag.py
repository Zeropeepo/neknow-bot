import json
import asyncio
import cohere
from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
from typing import List, AsyncGenerator
from google import genai
from google.genai import types
import structlog

from src.config import settings
from src.processor.embedder import embed_query
from src.storage.vector_store import hybrid_search
from src.consumer.document_consumer import start_consumer

logger = structlog.get_logger()


class HistoryMessage(BaseModel):
    role: str
    content: str


class RAGRequest(BaseModel):
    bot_id: str
    system_prompt: str
    query: str
    history: List[HistoryMessage] = []


MAX_HISTORY_MESSAGES = 6
MAX_HISTORY_CHARS = 1200
MAX_CHUNK_CHARS = 1200
MAX_CONTEXT_CHARS = 6000


def truncate_text(text: str, max_chars: int) -> str:
    if max_chars <= 0 or len(text) <= max_chars:
        return text
    return text[:max_chars]


@asynccontextmanager
async def lifespan(app: FastAPI):
    """
    Lifespan context manager for FastAPI.
    Starts RabbitMQ consumer on startup, gracefully stops on shutdown.
    All runs in the same event loop as FastAPI.
    """
    # Startup
    logger.info("worker_starting",
                api_port=settings.api_port,
                embedding_model=settings.embedding_model,
                chat_model=settings.chat_model)
    
    consumer_task = asyncio.create_task(start_consumer())
    logger.info("consumer_started", queue="index_file")
    
    yield
    
    # Shutdown
    consumer_task.cancel()
    try:
        await consumer_task
    except asyncio.CancelledError:
        logger.info("consumer_stopped")


app = FastAPI(lifespan=lifespan)



async def stream_response(request: RAGRequest) -> AsyncGenerator[str, None]:
    # Create async clients inside request scope to keep them bound to the active event loop.
    gemini_client = genai.Client(api_key=settings.gemini_api_key)
    cohere_client = cohere.AsyncClientV2(api_key=settings.cohere_api_key)

    log = logger.bind(
        bot_id=request.bot_id,
        query_len=len(request.query),
        history_len=len(request.history),
    )

    # Embed query
    query_embedding = await embed_query(request.query)

    # Hybrid search
    chunks = await hybrid_search(
        bot_id=request.bot_id,
        query_embedding=query_embedding,
        query_text=request.query,
        top_k=settings.top_k_retrieve,
    )

    # 3. Cohere rerank
    if chunks:
        try:
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
        except Exception as e:
            # Graceful fallback: keep serving answers using hybrid rank if rerank provider fails.
            log.warning("rerank_failed_fallback", error=str(e), chunk_count=len(chunks))
            top_chunks = [chunk["content"] for chunk in chunks[: settings.top_k_rerank]]
    else:
        top_chunks = []

    top_chunks = [truncate_text(chunk, MAX_CHUNK_CHARS) for chunk in top_chunks]

    # Build prompt
    context = "\n\n---\n\n".join(top_chunks) if top_chunks else "No relevant context found."
    context = truncate_text(context, MAX_CONTEXT_CHARS)

    system_instruction = f"{request.system_prompt}\n\nContext:\n{context}"
    log = log.bind(context_chars=len(context), system_chars=len(system_instruction))

    contents = []
    recent_history = request.history[-MAX_HISTORY_MESSAGES:]
    for msg in recent_history:
        contents.append(types.Content(
            role="user" if msg.role == "user" else "model",
            parts=[types.Part(text=truncate_text(msg.content, MAX_HISTORY_CHARS))]
        ))
    contents.append(types.Content(
        role="user",
        parts=[types.Part(text=request.query)]
    ))

    log.info("rag_prompt_built", history_used=len(recent_history), content_parts=len(contents))

    # Gemini stream
    try:
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
    except Exception as e:
        log.error("gemini_stream_failed", error=str(e))
        yield f"data: {json.dumps({'content': 'Maaf, terjadi gangguan saat memproses jawaban.'})}\n\n"
    finally:
        close_fn = getattr(cohere_client, "close", None)
        if close_fn is not None:
            try:
                maybe_awaitable = close_fn()
                if asyncio.iscoroutine(maybe_awaitable):
                    await maybe_awaitable
            except Exception:
                pass
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