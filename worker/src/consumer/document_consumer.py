import asyncio
import json
import structlog
import aio_pika
from src.config import settings
from src.storage.minio import download_document
from src.storage.vector_store import insert_chunks, update_file_status
from src.processor.parser import parse_document
from src.processor.chunker import chunk_text
from src.processor.embedder import embed_texts

logger = structlog.get_logger()

async def process_document(message_body: dict):
    file_id = message_body["file_id"]
    bot_id = message_body["bot_id"]
    object_key = message_body["object_key"]
    mime_type = message_body["mime_type"]
    filename = message_body["file_name"]

    log = logger.bind(file_id=file_id, bot_id=bot_id)

    try:
        await update_file_status(file_id, "indexing")
        log.info("indexing_started")

        content = await download_document(object_key)
        log.info("file_downloaded", size_bytes=len(content))

        text = parse_document(content, filename)
        if not text.strip():
            raise ValueError("document is empty after parsing")
        log.info("document_parsed", text_length=len(text))

        chunks = chunk_text(text)
        if not chunks:
            raise ValueError("no valid chunks produced")
        log.info("document_chunked", num_chunks=len(chunks))

        embeddings = await embed_texts(chunks)
        log.info("embeddings_generated", num_embeddings=len(embeddings))

        chunk_pairs = list(zip(chunks, embeddings))
        await insert_chunks(file_id, bot_id, chunk_pairs)
        log.info("chunks_inserted")

        await update_file_status(file_id, "indexed")
        log.info("indexing_completed")

    except Exception as e:
        log.error("indexing_failed", error=str(e))
        await update_file_status(file_id, "failed", str(e))
        raise


async def start_consumer():
    connection = await aio_pika.connect_robust(settings.rabbitmq_url)

    async with connection:
        channel = await connection.channel()

        await channel.set_qos(prefetch_count=1)
        queue = await channel.declare_queue("index_file", durable=True)

        log = structlog.get_logger()
        log.info("consumer_started", queue="index_file")

        async with queue.iterator() as queue_iter:
            async for message in queue_iter:
                async with message.process(requeue=True):
                    body = json.loads(message.body.decode())
                    await process_document(body)
