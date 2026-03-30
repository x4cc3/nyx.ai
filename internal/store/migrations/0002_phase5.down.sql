DROP INDEX IF EXISTS idx_memories_retention_policy;
DROP INDEX IF EXISTS idx_memories_embedding_cosine;

ALTER TABLE memories
    DROP COLUMN IF EXISTS retention_policy,
    DROP COLUMN IF EXISTS embedding_model,
    DROP COLUMN IF EXISTS embedding;
