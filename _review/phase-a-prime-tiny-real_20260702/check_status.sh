#!/usr/bin/env bash
set -euo pipefail
echo "== video_tasks =="
docker exec sub2api-postgres-dev psql -U sub2api -d sub2api -c \
  "SELECT id, status, provider, LEFT(COALESCE(result_url,''), 80) AS result_url FROM video_tasks ORDER BY id DESC LIMIT 5;"

echo "== ai_generation_content =="
docker exec sub2api-postgres-dev psql -U sub2api -d sub2api -c \
  "SELECT id, task_id, model, LEFT(prompt_redacted, 60) AS prompt_preview FROM ai_generation_content ORDER BY created_at DESC LIMIT 5;"

echo "== capture count for latest task =="
latest_task=$(docker exec sub2api-postgres-dev psql -U sub2api -d sub2api -tAc "SELECT id FROM video_tasks ORDER BY id DESC LIMIT 1;")
echo "latest_task_id=${latest_task}"
if [[ -n "${latest_task}" ]]; then
  docker exec sub2api-postgres-dev psql -U sub2api -d sub2api -tAc \
    "SELECT COUNT(*) FROM ai_generation_content WHERE task_id='${latest_task}';"
fi
