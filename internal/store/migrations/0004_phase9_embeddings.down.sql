DROP INDEX IF EXISTS idx_memories_embedding_cosine;

ALTER TABLE memories
    ALTER COLUMN embedding TYPE vector(16) USING NULL,
    ALTER COLUMN embedding_model SET DEFAULT 'nyx-hash-16';

CREATE INDEX IF NOT EXISTS idx_memories_embedding_cosine
    ON memories USING hnsw (embedding vector_cosine_ops);
