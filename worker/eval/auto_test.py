"""
Full RAG evaluation dengan metrik RAGAS standar industri.
Menggunakan run_rag() dari test_rag.py — pipeline identik dengan production.

Jalankan:
  uv run python -m eval.auto_test                          # auto-detect bot_id
  uv run python -m eval.auto_test <bot_id>                 # bot_id spesifik
  uv run python -m eval.auto_test <bot_id> <dataset.json>  # dataset custom

Metrik:
  - Faithfulness       : jawaban hanya dari context, tidak hallusinasi
  - Answer Relevancy   : jawaban relevan dengan pertanyaan
  - Context Precision  : chunk yang diambil memang relevan
  - Context Recall     : semua info penting berhasil di-retrieve
"""
import asyncio
import json
import math
import sys
import time
import warnings
from pathlib import Path

from datasets import Dataset
from langchain_google_genai import GoogleGenerativeAIEmbeddings
from langchain_groq import ChatGroq
from ragas import RunConfig, evaluate
from ragas.embeddings import LangchainEmbeddingsWrapper
from ragas.llms import LangchainLLMWrapper
from ragas.metrics import AnswerRelevancy, ContextPrecision, ContextRecall, Faithfulness

from eval.test_rag import list_bot_ids, run_rag
from src.config import settings

# ─── Warna terminal ───────────────────────────────────────────
GREEN  = "\033[92m"
RED    = "\033[91m"
YELLOW = "\033[93m"
CYAN   = "\033[96m"
RESET  = "\033[0m"
BOLD   = "\033[1m"
DIM    = "\033[2m"

def score_color(score: float) -> str:
    if score >= 0.7: return GREEN
    if score >= 0.4: return YELLOW
    return RED

def hr(char="─", n=65): print(char * n)

# ─── Groq wrapper — strip n di semua level ────────────────────
class GroqN1(ChatGroq):
    """Paksa n=1 di semua level — Groq tidak support n>1."""

    @property
    def _default_params(self) -> dict:
        params = super()._default_params
        params.pop("n", None)
        return params

    def _generate(self, messages, stop=None, run_manager=None, **kwargs):
        kwargs.pop("n", None)
        return super()._generate(messages, stop=stop, run_manager=run_manager, **kwargs)

    async def _agenerate(self, messages, stop=None, run_manager=None, **kwargs):
        kwargs.pop("n", None)
        return await super()._agenerate(messages, stop=stop, run_manager=run_manager, **kwargs)

# ─── Checkpoint ───────────────────────────────────────────────
CHECKPOINT_PATH = Path("eval/.checkpoint.json")

def save_checkpoint(data: dict, idx: int):
    CHECKPOINT_PATH.write_text(json.dumps({"data": data, "idx": idx}, ensure_ascii=False))

def load_checkpoint() -> tuple[dict, int]:
    if CHECKPOINT_PATH.exists():
        saved = json.loads(CHECKPOINT_PATH.read_text())
        idx = saved.get("idx", 0)
        print(f"{YELLOW}⚡ Resume dari pertanyaan {idx + 1} (checkpoint ditemukan){RESET}")
        return saved["data"], idx
    return {"question": [], "answer": [], "contexts": [], "ground_truth": []}, 0

# ─── Collect RAG outputs ──────────────────────────────────────
async def collect_rag_outputs(bot_id: str, dataset: list[dict]) -> dict:
    data, start_idx = load_checkpoint()

    print(f"\n{BOLD}⏳ Collecting RAG outputs ({len(dataset)} questions)...{RESET}\n")

    for i, item in enumerate(dataset):
        if i < start_idx:
            continue

        question     = item["question"]
        ground_truth = item["ground_truth"]
        print(f"  [{i+1:02d}/{len(dataset):02d}] {question[:65]}...")

        result = None
        for attempt in range(5):
            try:
                result = await run_rag(bot_id, question)
                break
            except Exception as e:
                if "429" in str(e):
                    wait = 60 * (attempt + 1)
                    print(f"  {YELLOW}⏳ Rate limited, tunggu {wait}s... (attempt {attempt+1}/5){RESET}")
                    await asyncio.sleep(wait)
                else:
                    raise

        if result is None:
            print(f"  {RED}❌ Skip pertanyaan {i+1} setelah 5x retry{RESET}")
            result = {"answer": "ERROR: rate limited", "chunks": [], "scores": []}

        data["question"].append(question)
        data["answer"].append(result["answer"])
        data["contexts"].append(result["chunks"] if result["chunks"] else ["No context found."])
        data["ground_truth"].append(ground_truth)

        print(f"         → {len(result['chunks'])} chunks | rerank: {result['scores'][0] if result['scores'] else 0:.4f}")

        save_checkpoint(data, i + 1)

        if i + 1 < len(dataset):
            await asyncio.sleep(4)

    return data

# ─── Run RAGAS evaluation ─────────────────────────────────────
def run_ragas(data: dict):
    warnings.filterwarnings("ignore", category=DeprecationWarning)

    llm = LangchainLLMWrapper(GroqN1(
        model="llama-3.1-8b-instant",
        api_key=settings.groq_api_key,
        temperature=0,
    ))

    emb = LangchainEmbeddingsWrapper(GoogleGenerativeAIEmbeddings(
        model=f"models/{settings.embedding_model}",
        google_api_key=settings.gemini_api_key,
    ))

    metrics = [
        Faithfulness(llm=llm),
        AnswerRelevancy(llm=llm, embeddings=emb),
        ContextPrecision(llm=llm),
        ContextRecall(llm=llm),
    ]

    dataset = Dataset.from_dict(data)

    print(f"\n{BOLD}🔬 Running RAGAS evaluation...{RESET}")
    print(f"{DIM}  LLM judge : llama-3.1-8b-instant (Groq){RESET}")
    print(f"{DIM}  Embedding : {settings.embedding_model} (Gemini){RESET}\n")

    return evaluate(
        dataset,
        metrics=metrics,
        raise_exceptions=False,
        run_config=RunConfig(timeout=120, max_retries=5, max_wait=60, max_workers=2),
    )

# ─── Print & save results ─────────────────────────────────────
def safe_mean(df, col: str) -> float:
    if col not in df.columns:
        return 0.0
    val = df[col].mean()
    return 0.0 if math.isnan(val) else float(val)

def print_results(results, data: dict, bot_id: str, elapsed: float):
    df = results.to_pandas()

    faith = safe_mean(df, "faithfulness")
    ans_r = safe_mean(df, "answer_relevancy")
    ctx_p = safe_mean(df, "context_precision")
    ctx_r = safe_mean(df, "context_recall")

    nan_count = int(df["faithfulness"].isna().sum()) if "faithfulness" in df.columns else len(df)
    if nan_count > 0:
        print(f"\n{YELLOW}⚠️  {nan_count}/{len(df)} pertanyaan gagal dievaluasi (NaN){RESET}")

    hr("═")
    print(f"\n{BOLD}📊 RAGAS EVALUATION RESULTS{RESET}")
    print(f"   bot_id  : {CYAN}{bot_id}{RESET}")
    print(f"   dataset : {len(data['question'])} questions")
    print(f"   model   : {settings.chat_model}")
    hr()

    metrics_display = [
        ("Faithfulness",      faith, "Jawaban hanya dari context, tidak hallusinasi"),
        ("Answer Relevancy",  ans_r, "Jawaban relevan dengan pertanyaan"),
        ("Context Precision", ctx_p, "Chunk yang diambil memang relevan (tidak noise)"),
        ("Context Recall",    ctx_r, "Semua info penting berhasil di-retrieve"),
    ]

    for name, score, desc in metrics_display:
        bar   = "█" * int(score * 20) + "░" * (20 - int(score * 20))
        color = score_color(score)
        print(f"  {BOLD}{name:<22}{RESET} {color}{score:.3f}{RESET}  {bar}  {DIM}{desc}{RESET}")

    overall = (faith + ans_r + ctx_p + ctx_r) / 4
    hr()
    print(f"  {BOLD}{'Overall (avg)':<22}{RESET} {score_color(overall)}{overall:.3f}{RESET}")
    print(f"  {DIM}Total waktu evaluasi: {elapsed:.1f}s{RESET}")
    hr("═")

    print(f"\n  {GREEN}■{RESET} ≥ 0.70  Bagus")
    print(f"  {YELLOW}■{RESET} ≥ 0.40  Perlu tuning")
    print(f"  {RED}■{RESET} < 0.40  Ada masalah serius\n")

    print(f"{BOLD}📋 Per-Question Detail{RESET}")
    hr()
    print(f"  {'#':<4} {'Faith':>7} {'AnsRel':>7} {'CtxPre':>7} {'CtxRec':>7}  Question")
    hr()
    for i, row in df.iterrows():
        def fmt(v): return f"{v:.3f}" if isinstance(v, float) and not math.isnan(v) else "  N/A"
        f  = fmt(row.get("faithfulness",      float("nan")))
        ar = fmt(row.get("answer_relevancy",  float("nan")))
        cp = fmt(row.get("context_precision", float("nan")))
        cr = fmt(row.get("context_recall",    float("nan")))
        q  = data["question"][i][:50]
        print(f"  {i+1:<4} {f:>7} {ar:>7} {cp:>7} {cr:>7}  {DIM}{q}...{RESET}")
    hr()

    out_path = Path("eval/results_ragas.csv")
    df["question"] = data["question"]
    df["answer"]   = data["answer"]
    df.to_csv(out_path, index=False)
    print(f"\n💾 Detail disimpan di {CYAN}eval/results_ragas.csv{RESET}")

    print(f"\n{BOLD}💡 Tuning Hints{RESET}")
    if nan_count == len(df):
        print(f"  {RED}• Semua evaluasi gagal — cek koneksi Groq/Gemini dan coba lagi{RESET}")
        return
    if faith < 0.7:
        print(f"  {RED}• Faithfulness rendah{RESET} → kuatkan system prompt: 'Jawab HANYA dari context'")
    if ans_r < 0.7:
        print(f"  {YELLOW}• Answer Relevancy rendah{RESET} → periksa task_type embed query = RETRIEVAL_QUERY")
    if ctx_p < 0.7:
        print(f"  {YELLOW}• Context Precision rendah{RESET} → turunkan top_k_retrieve atau naikkan top_k_rerank")
    if ctx_r < 0.7:
        print(f"  {RED}• Context Recall rendah{RESET} → naikkan chunk_size atau top_k_retrieve di .env")
    if all(s >= 0.7 for s in [faith, ans_r, ctx_p, ctx_r]):
        print(f"  {GREEN}✓ Semua metrik bagus! RAG pipeline kamu sudah solid.{RESET}")

# ─── Main ─────────────────────────────────────────────────────
async def main():
    if len(sys.argv) > 1:
        bot_id = sys.argv[1]
    else:
        bots = await list_bot_ids()
        if not bots:
            print("❌ Tidak ada data di file_chunks!")
            sys.exit(1)
        bot_id = bots[0]["bot_id"]
        print(f"✅ Auto-selected bot_id: {CYAN}{bot_id}{RESET}")

    dataset_path = Path(sys.argv[2]) if len(sys.argv) > 2 else Path("eval/qa_attention.json")
    if not dataset_path.exists():
        print(f"❌ Dataset tidak ditemukan: {dataset_path}")
        sys.exit(1)

    with open(dataset_path) as f:
        dataset = json.load(f)

    start   = time.time()
    data    = await collect_rag_outputs(bot_id, dataset)
    results = run_ragas(data)
    print_results(results, data, bot_id, time.time() - start)

if __name__ == "__main__":
    asyncio.run(main())