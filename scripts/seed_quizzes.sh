#!/usr/bin/env bash
set -euo pipefail

# Seeds sample quizzes into Postgres.
# Usage: DATABASE_URL=postgres://quiz:quizpass@localhost:5432/quizdb?sslmode=disable ./scripts/seed_quizzes.sh

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DB_URL="${DATABASE_URL:-postgres://quiz:quizpass@localhost:5432/quizdb?sslmode=disable}"

echo "Seeding quizzes into ${DB_URL}..."
psql "${DB_URL}" -f "${ROOT_DIR}/fixtures/quizzes.sql"
echo "Done."
