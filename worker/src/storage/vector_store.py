import uuid
from typing import List, Tuple
import asyncpg
from pgvector.asyncpg import register_vector
from src.config import settings


async def get_connection():
    conn = await asyncpg.connect(
        host=settings.db_host,
        port=settings.db_port,
        user=settings.db_user,
        password=settings.db_password,
        database=settings.db_name,
    )
    await register_vector(conn)
    return conn


async def insert_chunks(file_id: str, bot_id: str, chunks: List[Tuple[str, List[float]]]):
    """
    chunks: list of (text, embedding_vector)
    """
    conn = await get_connection()
    try:
        await conn.executemany(
            """
            INSERT INTO file_chunks (id, file_id, bot_id, content, embedding, chunk_index, created_at)
            VALUES ($1, $2, $3, $4, $5, $6, NOW())
            """,
            [
                (str(uuid.uuid4()), file_id, bot_id, text, embedding, idx)
                for idx, (text, embedding) in enumerate(chunks)
            ],
        )
    finally:
        await conn.close()


async def hybrid_search(
    bot_id: str,
    query_embedding: List[float],  
    query_text: str,               
    top_k: int = 20,
) -> List[dict]:
    conn = await get_connection()
    try:
        rows = await conn.fetch(
            """
            WITH dense AS (
            -- CTE pertama: rank berdasarkan vector similarity
                SELECT
                    id, content,
                    ROW_NUMBER() OVER (ORDER BY embedding <=> $1) AS rank
                    -- <=> adalah operator pgvector untuk cosine distance
                    -- Semakin kecil nilainya, semakin mirip
                    -- ROW_NUMBER() beri nomor urut 1,2,3,... dari yang paling mirip
                FROM file_chunks
                WHERE bot_id = $2
                ORDER BY embedding <=> $1
                LIMIT $3
            ),
            sparse AS (
            -- CTE kedua: rank berdasarkan keyword matching (BM25)
                SELECT
                    id, content,
                    ROW_NUMBER() OVER (
                        ORDER BY ts_rank(to_tsvector('english', content),
                                         plainto_tsquery('english', $4)) DESC
                        -- ts_rank() = BM25-like scoring di PostgreSQL
                        -- to_tsvector() = konversi teks ke searchable tokens
                        -- plainto_tsquery() = konversi query ke search query
                        -- DESC karena rank tinggi = lebih relevan (kebalikan dense)
                    ) AS rank
                FROM file_chunks
                WHERE bot_id = $2
                  AND to_tsvector('english', content) @@ plainto_tsquery('english', $4)
                  -- @@ = operator full-text search "match"
                  -- Hanya ambil chunk yang mengandung keyword dari query
                LIMIT $3
            ),
            combined AS (
            -- CTE ketiga: gabungkan dua rank dengan RRF
                SELECT
                    COALESCE(d.id, s.id) AS id,
                    -- COALESCE = ambil nilai pertama yang tidak NULL
                    -- Kalau chunk hanya ada di dense (tidak ada di sparse), pakai d.id
                    COALESCE(d.content, s.content) AS content,
                    COALESCE(1.0 / (60 + d.rank), 0) +
                    COALESCE(1.0 / (60 + s.rank), 0) AS rrf_score
                    -- Formula RRF: 1 / (k + rank) dimana k=60 adalah konstanta standar
                    -- Chunk yang rank tinggi di KEDUA metode dapat score tertinggi
                    -- Chunk yang hanya ada di salah satu tetap dapat score parsial
                    -- COALESCE(..., 0) = kalau NULL (chunk tidak ada di salah satu), score-nya 0
                FROM dense d
                FULL OUTER JOIN sparse s ON d.id = s.id
                -- FULL OUTER JOIN = ambil semua baris dari KEDUA sisi
                -- Berbeda dari INNER JOIN yang hanya ambil yang ada di keduanya
            )
            SELECT id, content, rrf_score
            FROM combined
            ORDER BY rrf_score DESC
            -- Sort dari score tertinggi
            LIMIT $3
            """,
            query_embedding,  # $1
            bot_id,           # $2
            top_k,            # $3
            query_text,       # $4
        )
        return [{"id": str(r["id"]), "content": r["content"]} for r in rows]
    finally:
        await conn.close()


async def update_file_status(file_id: str, status: str, error_msg: str = ""):
    conn = await get_connection()
    try:
        await conn.execute(
            """
            UPDATE files
            SET status = $1, error_msg = $2, updated_at = NOW()
            WHERE id = $3
            """,
            status,    # $1
            error_msg, # $2 
            file_id,   # $3
        )
    finally:
        await conn.close()
