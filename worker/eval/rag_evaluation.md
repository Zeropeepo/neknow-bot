# RAG System Evaluation

### Dataset
- **12 factual questions** derived from the Attention Is All You Need paper
- Ground truth written manually, covering architecture, hyperparameters, experimental results, and attention mechanisms

### RAGAS Metrics

| Metric | How It Works |
|---|---|
| **Faithfulness** | LLM judge verifies whether each claim in the answer can be traced back to the retrieved context |
| **Answer Relevancy** | Embedding similarity between the original question and questions re-generated from the answer |
| **Context Precision** | Proportion of relevant chunks among all retrieved chunks |
| **Context Recall** | Proportion of ground truth information successfully found in the retrieved context |

### Evaluation Setup
- **LLM Judge:** `llama-3.1-8b-instant` via Groq API (free tier, 14,400 RPD)
- **Embedding Judge:** `gemini-embedding-001` via Google AI Studio
- **Parallelism:** `max_workers=2` to avoid rate limiting
- **Retry:** `max_retries=5`, `timeout=120s` per job

---

## Per-Question Breakdown

| # | Question | Faith | AnsRel | CtxPre | CtxRec |
|---|---|---|---|---|---|
| 1 | What architecture does the Transformer replace? | 0.500 | 0.837 | 1.000 | N/A |
| 2 | How many encoder and decoder layers? | 0.500 | 0.850 | 0.700 | N/A |
| 3 | What is the model dimension (d_model)? | 1.000 | 0.827 | 1.000 | N/A |
| 4 | How many attention heads? | 1.000 | 0.774 | 0.917 | 1.000 |
| 5 | Feed-forward inner-layer dimension (d_ff)? | N/A | 0.907 | 0.887 | 1.000 |
| 6 | BLEU score on WMT 2014 English-to-German? | N/A | 0.877 | 0.950 | N/A |
| 7 | BLEU score on WMT 2014 English-to-French? | 1.000 | 0.866 | 1.000 | N/A |
| 8 | What optimizer was used? | 0.667 | 0.000 | 0.000 | N/A |
| 9 | What dropout rate was applied? | 1.000 | 0.850 | 1.000 | 1.000 |
| 10 | How does the Transformer encode token positions? | N/A | 0.865 | 0.950 | N/A |
| 11 | What is scaled dot-product attention? | 0.133 | 0.831 | 1.000 | 0.115 |
| 12 | How many GPUs and training duration? | N/A | 0.844 | 0.950 | 1.000 |

> **Note:** N/A on ContextRecall occurs when the ground truth is too short or incompatible for recall evaluation by the judge — this is not an indicator of retrieval failure.

---

## Findings & Recommendations

### ⚠️ Q8 — Optimizer (Adam)
- **Faith: 0.667 | AnsRel: 0.000 | CtxPre: 0.000** — retriever completely failed to find relevant chunks
- **Root cause:** Optimizer information likely sits at a chunk boundary and was split across chunks
- **Recommendation:** Increase `CHUNK_OVERLAP` from 50 → 100

### ⚠️ Q11 — Scaled Dot-Product Attention
- **Faith: 0.133 | CtxRec: 0.115** — model elaborated on the mathematical formula not verbatim present in the chunk
- **Root cause:** Likely a false negative from the LLM judge — the formula \( \text{Attention}(Q,K,V) = \text{softmax}\left(\frac{QK^T}{\sqrt{d_k}}\right)V \) was flagged as ungrounded despite being factually correct
- **Recommendation:** Strengthen system prompt to prevent elaboration outside context

### Pipeline Strengths
- **Context Precision 0.863** — hybrid search + rerank is working very well, almost no noise
- **Context Recall 0.823** — chunk size and overlap are sufficient for the majority of questions
- **Answer Relevancy 0.777** — answer generation is consistently on-topic

---

## Next Tuning Steps

```env
# Priority 1 — fix Q8 (optimizer retrieval failure)
CHUNK_OVERLAP=100

# Priority 2 — fix Q11 (low faithfulness due to elaboration)
# Add to system prompt:
# "Answer ONLY based on the provided context.
#  Do not add explanations, formulas, or information outside the context."
```