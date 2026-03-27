import io
import tempfile
import os
from unstructured.partition.auto import partition

def parse_document(content: bytes, filename: str) -> str:
    suffix = os.path.splitext(filename)[1].lower()

    with tempfile.NamedTemporaryFile(suffix=suffix, delete=False) as tmp:
        tmp.write(content)
        tmp_path = tmp.name

    try:
        elements = partition(filename=tmp_path)
        text = "\n\n".join(
            str(el) for el in elements
            if str(el).strip()
        )
        return text
    finally:
        os.unlink(tmp_path)
