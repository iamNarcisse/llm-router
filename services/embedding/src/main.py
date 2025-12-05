# services/embedding/src/main.py
import os
import logging
from concurrent import futures

import grpc
from grpc_reflection.v1alpha import reflection

from .servicer import EmbeddingServicer
from .pb import router_pb2, router_pb2_grpc

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


def serve():
    model_name = os.getenv("MODEL_NAME", "sentence-transformers/all-MiniLM-L6-v2")
    port = os.getenv("GRPC_PORT", "50052")
    max_workers = int(os.getenv("MAX_WORKERS", "4"))

    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=max_workers),
        options=[
            ("grpc.max_send_message_length", 50 * 1024 * 1024),
            ("grpc.max_receive_message_length", 50 * 1024 * 1024),
        ],
    )

    servicer = EmbeddingServicer(model_name)
    router_pb2_grpc.add_EmbeddingServiceServicer_to_server(servicer, server)

    # Enable reflection
    SERVICE_NAMES = (
        router_pb2.DESCRIPTOR.services_by_name["EmbeddingService"].full_name,
        reflection.SERVICE_NAME,
    )
    reflection.enable_server_reflection(SERVICE_NAMES, server)

    server.add_insecure_port(f"[::]:{port}")
    server.start()

    logger.info(f"Embedding service listening on port {port}")
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
