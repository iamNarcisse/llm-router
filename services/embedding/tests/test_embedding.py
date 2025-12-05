# tests/test_embedding.py
from sentence_transformers import SentenceTransformer
import time


def benchmark(model_name: str, backend: str = "torch"):
    print(f"\nLoading {model_name} ({backend})...")

    if backend == "onnx":
        model = SentenceTransformer(
            model_name,
            backend="onnx",
            model_kwargs={"provider": "CPUExecutionProvider"},
        )
    else:
        model = SentenceTransformer(model_name)

    # Warm up
    for _ in range(3):
        model.encode("warmup", normalize_embeddings=True)

    # Benchmark
    times = []
    for _ in range(10):
        start = time.perf_counter()
        model.encode(
            "write a python function to sort a list", normalize_embeddings=True
        )
        times.append((time.perf_counter() - start) * 1000)

    avg = sum(times) / len(times)
    print(
        f"{model_name} ({backend}): {avg:.1f}ms avg (min: {min(times):.1f}ms, max: {max(times):.1f}ms)"
    )
    return avg


if __name__ == "__main__":
    # These models have good ONNX support
    benchmark("BAAI/bge-small-en-v1.5", "torch")
    benchmark("BAAI/bge-small-en-v1.5", "onnx")

    # benchmark("sentence-transformers/all-MiniLM-L6-v2", "torch")
    # benchmark("sentence-transformers/all-MiniLM-L6-v2", "onnx")
