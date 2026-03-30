"""
Quick RAG smoke test — tanpa login, tanpa input manual.
Otomatis ambil bot_id dari DB yang sudah punya chunks.

Jalankan: uv run python -m eval.test_rag
Atau dengan pertanyaan custom: uv run python -m eval.test_rag "apa itu refund?"
"""
import asyncio
import sys
import textwrap
import cohere
from google import genai
from google.genai import types
from src.config import settings
from src.processor.embedder import embed_query
from src.storage.vector_store import get_connection, hybrid_search

gemini_client = genai.Client(api_key=settings.gemini_api_key)
cohere_client = cohere.ClientV2(api_key=settings.cohere_api_key)

# ─── Warna terminal ───────────────────────────────────────────
GREEN  = "\033[92m"
YELLOW = "\033[93m"
CYAN   = "\033[96m"
RESET  = "\033[0m"
BOLD   = "\033[1m"
DIM    = "\033[2m"

def hr(char="─", n=60):
    print(char * n)

# ─── Auto-detect bot_id dari DB ───────────────────────────────
async def pick_bot_id() -> tuple[str, int]:
    """Ambil bot_id pertama yang punya chunks terbanyak."""
    conn = await get_connection()
    try:
        rows = await conn.fetch("""
            SELECT bot_id, COUNT(*) as total
            FROM file_chunks
            GROUP BY bot_id
            ORDER BY total DESC
            LIMIT 5
        """)
        if not rows:
            raise RuntimeError("Tidak ada data di tabel file_chunks. Upload dokumen dulu!")
        return str(rows[0]["bot_id"]), int(rows[0]["total"])
    finally:
        await conn.close()

async def list_bot_ids() -> list[dict]:
    conn = await get_connection()
    try:
        rows = await conn.fetch("""
            SELECT bot_id, COUNT(*) as total_chunks
            FROM file_chunks
            GROUP BY bot_id
            ORDER BY total_chunks DESC
        """)
        return [{"bot_id": str(r["bot_id"]), "chunks": int(r["total_chunks"])} for r in rows]
    finally:
        await conn.close()

# ─── RAG pipeline (sama persis dengan production) ─────────────
async def run_rag(bot_id: str, question: str) -> dict:
    query_embedding = await embed_query(question)

    chunks = await hybrid_search(
        bot_id=bot_id,
        query_embedding=query_embedding,
        query_text=question,
        top_k=settings.top_k_retrieve,
    )

    if chunks:
        rerank_response = cohere_client.rerank(
            model="rerank-v3.5",
            query=question,
            documents=[c["content"] for c in chunks],
            top_n=settings.top_k_rerank,
        )
        top_chunks = [chunks[r.index]["content"] for r in rerank_response.results]
        scores     = [round(r.relevance_score, 4) for r in rerank_response.results]
    else:
        top_chunks, scores = [], []

    context = "\n\n---\n\n".join(top_chunks) if top_chunks else "No context found."
    system_instruction = (
        "Kamu adalah asisten yang menjawab berdasarkan context yang diberikan. "
        "Jika context tidak relevan, katakan tidak tahu.\n\n"
        f"Context:\n{context}"
    )

    result = gemini_client.models.generate_content(
        model=settings.chat_model,
        contents=question,
        config=types.GenerateContentConfig(
            system_instruction=system_instruction,
            temperature=0.0,
            max_output_tokens=512,
        ),
    )

    return {
        "answer": result.text,
        "chunks": top_chunks,
        "scores": scores,
    }

# ─── Pretty print hasil ───────────────────────────────────────
def print_result(question: str, result: dict, bot_id: str):
    hr("═")
    print(f"{BOLD}❓ QUESTION{RESET}  {question}")
    print(f"{DIM}   bot_id: {bot_id}{RESET}")
    hr()

    print(f"\n{BOLD}{CYAN}💬 ANSWER{RESET}")
    for line in textwrap.wrap(result["answer"], width=70):
        print(f"   {line}")

    print(f"\n{BOLD}{YELLOW}📄 RETRIEVED CHUNKS ({len(result['chunks'])}){RESET}")
    for i, (chunk, score) in enumerate(zip(result["chunks"], result["scores"]), 1):
        preview = chunk[:120].replace("\n", " ") + ("..." if len(chunk) > 120 else "")
        print(f"\n   {BOLD}[{i}] relevance_score: {GREEN}{score}{RESET}")
        print(f"   {DIM}{preview}{RESET}")

    hr("═")

# ─── Default questions untuk smoke test ───────────────────────
DEFAULT_QUESTIONS = [
    "Apa tujuan utama dari dokumen ini?",
    "Sebutkan poin-poin penting yang ada.",
    "Apa kesimpulan atau rekomendasi yang disebutkan?",
]

async def main():
    print(f"\n{BOLD}🔍 Detecting available bots...{RESET}")
    bots = await list_bot_ids()

    if not bots:
        print("❌ Tidak ada data di file_chunks. Upload dokumen dulu!")
        sys.exit(1)

    print(f"\n{'Bot ID':<40} {'Chunks':>8}")
    hr("-", 50)
    for b in bots:
        marker = f"{GREEN}◀ (auto-selected){RESET}" if b == bots[0] else ""
        print(f"{b['bot_id']:<40} {b['chunks']:>8}  {marker}")

    bot_id = bots[0]["bot_id"]
    hr()

    if len(sys.argv) > 1:
        questions = [" ".join(sys.argv[1:])]
        print(f"\n{BOLD}Mode: Single question dari CLI{RESET}")
    else:
        questions = DEFAULT_QUESTIONS
        print(f"\n{BOLD}Mode: Smoke test ({len(questions)} default questions){RESET}")
        print(f"{DIM}Tip: uv run python -m eval.test_rag \"pertanyaan custom kamu\"{RESET}")

    for q in questions:
        print(f"\n⏳ Running RAG untuk: {CYAN}{q}{RESET}")
        result = await run_rag(bot_id, q)
        print_result(q, result, bot_id)

# run_rag() dan helper lainnya tetap di atas — bisa di-import oleh auto_test.py
if __name__ == "__main__":
    asyncio.run(main())