#!/usr/bin/env python3
import os
import sys
import uuid
import logging

import yaml
from sentence_transformers import SentenceTransformer
from qdrant_client import QdrantClient
from qdrant_client.models import Distance, VectorParams, PointStruct

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


def load_routes(path: str) -> dict:
    with open(path) as f:
        return yaml.safe_load(f)


def main():
    routes_path = os.getenv("ROUTES_PATH", "configs/routes.yaml")
    model_name = os.getenv("MODEL_NAME", "BAAI/bge-small-en-v1.5")
    qdrant_host = os.getenv("QDRANT_HOST", "localhost")
    qdrant_port = int(os.getenv("QDRANT_PORT", "6333"))
    collection_name = os.getenv("COLLECTION_NAME", "llm_routes")

    # Load routes
    logger.info(f"Loading routes from {routes_path}")
    config = load_routes(routes_path)
    routes = config.get("routes", [])

    if not routes:
        logger.error("No routes found in config")
        sys.exit(1)

    # Load model
    logger.info(f"Loading embedding model: {model_name}")
    model = SentenceTransformer(model_name)
    # Get actual dimension by encoding a test string
    test_vector = model.encode("test", normalize_embeddings=True)
    dimensions = len(test_vector)
    logger.info(f"Model dimensions: {dimensions}")

    # Connect to Qdrant
    logger.info(f"Connecting to Qdrant at {qdrant_host}:{qdrant_port}")
    client = QdrantClient(host=qdrant_host, port=qdrant_port)

    # Recreate collection
    logger.info(f"Creating collection: {collection_name}")
    if client.collection_exists(collection_name):
        client.delete_collection(collection_name)

    client.create_collection(
        collection_name=collection_name,
        vectors_config=VectorParams(
            size=dimensions if dimensions is not None else 0, distance=Distance.COSINE
        ),
    )

    # Generate and upsert points
    points = []
    total = 0

    for route in routes:
        name = route["name"]
        model_id = route["model"]
        provider = route.get("provider", "unknown")
        metadata = route.get("metadata", {})
        utterances = route.get("utterances", [])

        logger.info(f"Processing route '{name}' ({len(utterances)} utterances)")

        vectors = model.encode(
            utterances,
            prompt_name="query",
            normalize_embeddings=True,
            show_progress_bar=False,
        )

        for utterance, vector in zip(utterances, vectors):
            points.append(
                PointStruct(
                    id=str(uuid.uuid4()),
                    vector=vector.tolist(),
                    payload={
                        "route": name,
                        "model": model_id,
                        "provider": provider,
                        "utterance": utterance,
                        **metadata,
                    },
                )
            )
            total += 1

    logger.info(f"Upserting {len(points)} points")
    client.upsert(collection_name=collection_name, points=points)

    # Verify
    info = client.get_collection(collection_name)
    logger.info(f"Collection '{collection_name}' has {info.points_count} points")
    logger.info(f"Seeded {total} utterances across {len(routes)} routes")


if __name__ == "__main__":
    main()
