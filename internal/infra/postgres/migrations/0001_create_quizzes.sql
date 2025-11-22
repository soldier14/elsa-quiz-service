-- Creates the quizzes table to store quiz content as JSONB.
CREATE TABLE IF NOT EXISTS quizzes (
    id TEXT PRIMARY KEY,
    data JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Helpful index if querying nested fields (optional).
-- CREATE INDEX IF NOT EXISTS idx_quizzes_data_gin ON quizzes USING GIN (data);
