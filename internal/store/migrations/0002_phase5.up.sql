CREATE EXTENSION IF NOT EXISTS vector;

ALTER TABLE memories
    ADD COLUMN IF NOT EXISTS embedding vector(16),
    ADD COLUMN IF NOT EXISTS embedding_model TEXT NOT NULL DEFAULT 'nyx-hash-16',
    ADD COLUMN IF NOT EXISTS retention_policy TEXT NOT NULL DEFAULT 'standard';

CREATE INDEX IF NOT EXISTS idx_memories_embedding_cosine
    ON memories USING hnsw (embedding vector_cosine_ops);

CREATE INDEX IF NOT EXISTS idx_memories_retention_policy
    ON memories(retention_policy);
