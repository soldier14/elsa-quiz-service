-- Sample quiz fixtures.
-- Import with: psql "$DATABASE_URL" -f fixtures/quizzes.sql

INSERT INTO quizzes (id, data)
VALUES
  (
    'quiz-1',
    '{
      "id": "quiz-1",
      "questions": [
        {
          "id": "q1",
          "prompt": "What is 2 + 2?",
          "options": [
            {"id": "o1", "text": "3", "correct": false},
            {"id": "o2", "text": "4", "correct": true},
            {"id": "o3", "text": "5", "correct": false}
          ],
          "points": 1
        },
        {
          "id": "q2",
          "prompt": "Choose the vowel.",
          "options": [
            {"id": "o1", "text": "b", "correct": false},
            {"id": "o2", "text": "a", "correct": true},
            {"id": "o3", "text": "t", "correct": false}
          ],
          "points": 1
        }
      ]
    }'::jsonb
  ),
  (
    'quiz-2',
    '{
      "id": "quiz-2",
      "questions": [
        {
          "id": "q1",
          "prompt": "Synonym for quick?",
          "options": [
            {"id": "o1", "text": "fast", "correct": true},
            {"id": "o2", "text": "slow", "correct": false},
            {"id": "o3", "text": "late", "correct": false}
          ],
          "points": 2
        }
      ]
    }'::jsonb
  )
ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data, updated_at = NOW();
