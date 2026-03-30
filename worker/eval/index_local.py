"""
Index file lokal langsung ke DB, tanpa perlu Go API / RabbitMQ.
Jalankan: uv run python -m eval.index_local <path_to_pdf> <bot_id_opsional>

Contoh:
  uv run python -m eval.index_local eval/test_docs/attention.pdf
  uv run python -m eval.index_local eval/test_docs/attention.pdf my-test-bot-001
"""
import asyncio
import sys
import uuid
from pathlib import Path

from src.processor.chunker import chunk_document
from src.processor.embedder import embed_texts
from src.storage.vector_store import get_connection, insert_chunks
from pgvector.asyncpg import register_vector

MIME_MAP = {
    ".pdf":  "application/pdf",
    ".txt":  "text/plain",
    ".csv":  "text/csv",
    ".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
}

GREEN = "\033[92m"
CYAN  = "\033[96m"
RESET = "\033[0m"
BOLD  = "\033[1m"

async def clear_bot_chunks(bot_id: str):
    conn = await get_connection()
    try:
        deleted = await conn.fetchval(
            "DELETE FROM file_chunks WHERE bot_id = $1 RETURNING COUNT(*)", bot_id
        )
        return deleted or 0
    finally:
        await conn.close()

async def index_file(file_path: Path, bot_id: str):
    mime = MIME_MAP.get(file_path.suffix.lower())
    if not mime:
        print(f"❌ Format tidak didukung: {file_path.suffix}")
        sys.exit(1)

    print(f"\n{BOLD}📄 File   :{RESET} {file_path.name}")
    print(f"{BOLD}🤖 Bot ID :{RESET} {bot_id}")
    print(f"{BOLD}📦 MIME   :{RESET} {mime}\n")

    # 1. Baca file
    data = file_path.read_bytes()
    print(f"✅ File dibaca ({len(data):,} bytes)")

    # 2. Chunk
    chunks = chunk_document(data, mime)
    print(f"✅ Chunking selesai → {CYAN}{len(chunks)} chunks{RESET}")

    # 3. Embed
    print(f"⏳ Generating embeddings untuk {len(chunks)} chunks...")
    embeddings = await embed_texts(chunks)
    print(f"✅ Embeddings selesai ({len(embeddings[0])} dimensi)")

    # 4. Clear chunk lama untuk bot ini
    conn = await get_connection()
    try:
        await conn.execute("DELETE FROM file_chunks WHERE bot_id = $1", bot_id)

        # ← Tambah ini: insert dummy row ke tabel files dulu
        file_id = str(uuid.uuid4())
        await conn.execute("""
        INSERT INTO files (id, bot_id, user_id, name, size, mime_type, bucket, object_key, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'indexed', NOW(), NOW())
        ON CONFLICT (id) DO NOTHING
    """,
        file_id,
        bot_id,
        "eval-user",            # user_id dummy
        file_path.name,
        len(data),              # size = bytes
        MIME_MAP.get(file_path.suffix.lower()),
        "eval",                 # bucket dummy
        f"eval/{file_path.name}",
        )
    finally:
        await conn.close()
    
    chunk_pairs = list(zip(chunks, embeddings))
    await insert_chunks(file_id, bot_id, chunk_pairs)
    print(f"✅ {len(chunk_pairs)} chunks disimpan ke DB")
    print(f"\n{GREEN}{BOLD}🎉 Indexing selesai!{RESET}")
    print(f"   bot_id  : {CYAN}{bot_id}{RESET}")
    print(f"   file_id : {file_id}")
    print(f"   chunks  : {len(chunk_pairs)}")
    return bot_id

async def main():
    if len(sys.argv) < 2:
        print("Usage: uv run python -m eval.index_local <file_path> [bot_id]")
        sys.exit(1)

    file_path = Path(sys.argv[1])
    if not file_path.exists():
        print(f"❌ File tidak ditemukan: {file_path}")
        sys.exit(1)

    bot_id = sys.argv[2] if len(sys.argv) > 2 else f"eval-{file_path.stem[:20]}"
    await index_file(file_path, bot_id)

if __name__ == "__main__":
    asyncio.run(main())