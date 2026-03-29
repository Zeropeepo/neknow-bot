import io
import pandas as pd
from pypdf import PdfReader
from docx import Document
from typing import List
import tiktoken
from langchain_text_splitters import RecursiveCharacterTextSplitter
from src.config import settings

tokenizer = tiktoken.encoding_for_model("text-embedding-3-small")

SUPPORTED_TYPES = {
    "application/pdf",
    "text/plain",
    "text/csv",
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
}

def count_tokens(text: str) -> int:
    return len(tokenizer.encode(text))

def chunk_text(text: str) -> List[str]:
    splitter = RecursiveCharacterTextSplitter(
        chunk_size=settings.chunk_size,
        chunk_overlap=settings.chunk_overlap,
        length_function=count_tokens,
        separators=["\n\n", "\n", ". ", " ", ""],
    )
    chunks = splitter.split_text(text)
    return [c for c in chunks if count_tokens(c) >= 20]

def extract_text(data: bytes, content_type: str) -> str:
    if content_type == "application/pdf":
        reader = PdfReader(io.BytesIO(data))
        return "\n\n".join(p.extract_text() or "" for p in reader.pages).strip()

    elif content_type == "text/plain":
        return data.decode("utf-8", errors="ignore").strip()

    elif content_type == "text/csv":
        df = pd.read_csv(io.BytesIO(data))
        rows = [
            ", ".join(f"{col}: {val}" for col, val in row.items())
            for _, row in df.iterrows()
        ]
        return "\n".join(rows)

    elif content_type == "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
        doc = Document(io.BytesIO(data))
        return "\n\n".join(p.text for p in doc.paragraphs if p.text.strip())

    raise ValueError(f"Unsupported content type: {content_type}")

def chunk_document(data: bytes, content_type: str) -> List[str]:
    text = extract_text(data, content_type)
    if not text:
        raise ValueError("Document is empty or could not be extracted")
    return chunk_text(text)