import uvicorn
import structlog
from src.api.rag import app
from src.config import settings

logger = structlog.get_logger()


if __name__ == "__main__":
    logger.info("api_server_started", port=settings.api_port)
    uvicorn.run(
        app,
        host="0.0.0.0",
        port=settings.api_port,
        log_level="warning"
    )
