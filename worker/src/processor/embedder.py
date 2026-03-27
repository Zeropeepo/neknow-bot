from typing import List
from openai import AsyncOpenAI
from src.config import settings

client = AsyncOpenAI(api_key=settings.openai_api_key)

async def embed_texts(texts: List[str]) -> List[List[float]]:
    if not texts:
        return []

    all_embeddings = []
    batch_size = 100

    for i in range(0, len(texts), batch_size):
        batch = texts[i:i + batch_size]
        response = await client.embeddings.create(
            model=settings.embedding_model,
            input=batch,
        )
        embeddings = [
            item.embedding
            for item in sorted(response.data, key=lambda x: x.index)
        ]
        all_embeddings.extend(embeddings)

    return all_embeddings
