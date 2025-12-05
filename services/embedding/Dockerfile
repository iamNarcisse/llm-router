FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application
COPY src/ ./src/

# Pre-download model during build (optional, speeds up startup)
ARG MODEL_NAME=BAAI/bge-small-en-v1.5
RUN python -c "from sentence_transformers import SentenceTransformer; SentenceTransformer('${MODEL_NAME}')"

EXPOSE 50052

CMD ["python", "-m", "src.main"]