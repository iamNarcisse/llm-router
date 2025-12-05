# services/embedding/src/servicer.py
import time
import logging
import os

from sentence_transformers import SentenceTransformer

from .pb import router_pb2, router_pb2_grpc

logger = logging.getLogger(__name__)


class EmbeddingServicer(router_pb2_grpc.EmbeddingServiceServicer):
    def __init__(self, model_name: str):
        self.model_name = model_name
        logger.info(f"Loading model: {model_name}")
        start = time.perf_counter()

        # Check if ONNX backend is requested
        backend = os.getenv("EMBEDDING_BACKEND", "torch")

        if backend == "onnx":
            logger.info("Using ONNX backend")
            self.model = SentenceTransformer(
                model_name,
                backend="onnx",
                model_kwargs={"provider": "CPUExecutionProvider"},
            )
        else:
            self.model = SentenceTransformer(model_name)

        self.dimensions = self.model.get_sentence_embedding_dimension()

        # Warm up (important — first inference is always slower)
        for _ in range(3):
            _ = self.model.encode("warmup query", normalize_embeddings=True)

        load_time = time.perf_counter() - start
        logger.info(f"Model loaded in {load_time:.2f}s, dimensions: {self.dimensions}")

    def Embed(
        self, request: router_pb2.EmbedRequest, context
    ) -> router_pb2.EmbedResponse:
        start = time.perf_counter()

        vector = self.model.encode(
            request.text, prompt_name="query", normalize_embeddings=True
        ).tolist()

        latency_ms = (time.perf_counter() - start) * 1000

        return router_pb2.EmbedResponse(
            vector=vector, dimensions=self.dimensions, latency_ms=latency_ms
        )

    def EmbedBatch(
        self, request: router_pb2.EmbedBatchRequest, context
    ) -> router_pb2.EmbedBatchResponse:
        start = time.perf_counter()

        vectors = self.model.encode(
            list(request.texts),
            prompt_name="query",
            normalize_embeddings=True,
            batch_size=32,
        )

        latency_ms = (time.perf_counter() - start) * 1000

        return router_pb2.EmbedBatchResponse(
            vectors=[router_pb2.Vector(values=v.tolist()) for v in vectors],
            latency_ms=latency_ms,
        )
