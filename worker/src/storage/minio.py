import io
import asyncio
from datetime import timedelta
from minio import Minio
from src.config import settings

_client: Minio | None = None


def get_minio() -> Minio:
    global _client
    if _client is None:
        _client = Minio(
            endpoint=settings.minio_endpoint,
            access_key=settings.minio_access_key,
            secret_key=settings.minio_secret_key,
            secure=settings.minio_use_ssl,
        )
    return _client


async def upload_document(bot_id: str, doc_id: str, data: bytes, content_type: str) -> str:
    object_name = f"{bot_id}/documents/{doc_id}"
    await asyncio.to_thread(
        get_minio().put_object,
        settings.minio_bucket,
        object_name,
        io.BytesIO(data),
        len(data),
        content_type=content_type,
    )
    return object_name


async def download_document(object_name: str) -> bytes:
    response = await asyncio.to_thread(
        get_minio().get_object,
        settings.minio_bucket,
        object_name,
    )
    try:
        return response.read()
    finally:
        response.close()
        response.release_conn()


async def delete_document(object_name: str) -> None:
    await asyncio.to_thread(
        get_minio().remove_object,
        settings.minio_bucket,
        object_name,
    )


async def presigned_url(object_name: str, expires_hours: int = 1) -> str:
    return await asyncio.to_thread(
        get_minio().presigned_get_object,
        settings.minio_bucket,
        object_name,
        expires=timedelta(hours=expires_hours),
    )