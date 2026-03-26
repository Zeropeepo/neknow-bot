from pydantic_settings import BaseSettings
from pydantic import Field
from typing import List


class Settings(BaseSettings):
    # Database
    db_host: str = "localhost"
    db_port: int = 5432
    db_user: str = "neknowbot"
    db_password: str = ""
    db_name: str = "neknow_bot_db"

    # RabbitMQ
    rabbitmq_url: str = "amqp://guest:guest@localhost:5672/"

    # MinIO
    minio_endpoint: str = "localhost:9000"
    minio_access_key: str = "minioadmin"
    minio_secret_key: str = ""
    minio_use_ssl: bool = False
    minio_bucket: str = "neknowbot-files"

    # OpenAI
    openai_api_key: str = ""
    embedding_model: str = "text-embedding-3-small"
    chat_model: str = "gpt-4o-mini"

    # Cohere
    cohere_api_key: str = ""

    # App
    api_port: int = 8000
    top_k_retrieve: int = 20   # Chunk
    top_k_rerank: int = 5      # 5 rerank
    chunk_size: int = 512
    chunk_overlap: int = 50

    @property
    def db_url(self) -> str:
        return (
            f"postgresql+asyncpg://{self.db_user}:{self.db_password}"
            f"@{self.db_host}:{self.db_port}/{self.db_name}"
        )

    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"
        extra = "ignore"  # ignore .env


settings = Settings()
