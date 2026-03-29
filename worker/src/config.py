from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
        case_sensitive=False,
    )

    # Database
    db_host: str = "localhost"
    db_port: int = 5432
    db_user: str = "neknowbot"
    db_password: str           
    db_name: str = "neknow_bot_db"

    # RabbitMQ
    rabbitmq_url: str = "amqp://guest:guest@localhost:5672/"

    # MinIO
    minio_endpoint: str = "localhost:9000"
    minio_access_key: str      
    minio_secret_key: str      
    minio_use_ssl: bool = False
    minio_bucket: str = "neknowbot-files"

    # Gemini
    gemini_api_key: str        
    embedding_model: str = "gemini-embedding-001"
    chat_model: str = "gemini-3.0-flash"

    # Cohere
    cohere_api_key: str        

    # App
    api_port: int = 8000
    top_k_retrieve: int = 20
    top_k_rerank: int = 5
    chunk_size: int = 512
    chunk_overlap: int = 50

    @property
    def db_url(self) -> str:
        return (
            f"postgresql+asyncpg://{self.db_user}:{self.db_password}"
            f"@{self.db_host}:{self.db_port}/{self.db_name}"
        )


settings = Settings()