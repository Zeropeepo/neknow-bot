from typing import List
from src.config import settings
from google import genai
from google.genai import types


async def embed_texts(texts: List[str]) -> List[List[float]]:
    if not texts:
        return []

    # Build client inside coroutine so async sessions are created on the active event loop.
    client = genai.Client(api_key=settings.gemini_api_key)

    all_embeddings = []
    batch_size = 100

    for i in range(0, len(texts), batch_size):
        batch = texts[i:i + batch_size]
        result = await client.aio.models.embed_content(
            model=settings.embedding_model,
            contents=batch,
            config=types.EmbedContentConfig(
                task_type="RETRIEVAL_DOCUMENT",
            ),
        )
        all_embeddings.extend([e.values for e in result.embeddings])

    return all_embeddings


async def embed_query(text: str) -> List[float]:
    # Build client inside coroutine so async sessions are created on the active event loop.
    client = genai.Client(api_key=settings.gemini_api_key)

    result = await client.aio.models.embed_content( 
        model=settings.embedding_model,
        contents=text,
        config=types.EmbedContentConfig(
            task_type="RETRIEVAL_QUERY",
        ),
    )
    return result.embeddings[0].values