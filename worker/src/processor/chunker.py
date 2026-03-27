from typing import List
import tiktoken
from langchain_text_splitters import RecursiveCharacterTextSplitter
from src.config import settings


tokenizer = tiktoken.encoding_for_model("text-embedding-3-small")


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
