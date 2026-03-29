import asyncio
import threading
import uvicorn
import structlog
from src.api.rag import app        
from src.consumer.document_consumer import start_consumer
from src.config import settings

logger = structlog.get_logger()


def run_api():
    uvicorn.run(
        app,
        host="0.0.0.0",   
        port=settings.api_port,
        log_level="warning"
    )


async def main():
    logger.info("worker_starting",
                api_port=settings.api_port,
                embedding_model=settings.embedding_model,
                chat_model=settings.chat_model)

    api_thread = threading.Thread(target=run_api, daemon=True)
    api_thread.start()
    logger.info("api_server_started", port=settings.api_port)

    await start_consumer()


if __name__ == "__main__":
    asyncio.run(main())
