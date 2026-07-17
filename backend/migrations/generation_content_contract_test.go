package migrations

import (
	"strings"
	"testing"
)

func TestGenerationContentMigrationsUseForwardSchemaAndPartialUniqueTaskID(t *testing.T) {
	tableSQL, err := FS.ReadFile("182_ai_generation_content.sql")
	if err != nil {
		t.Fatal(err)
	}
	indexSQL, err := FS.ReadFile("183_ai_generation_content_task_id_unique_notx.sql")
	if err != nil {
		t.Fatal(err)
	}
	table := string(tableSQL)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS ai_generation_content",
		"api_key_id           BIGINT REFERENCES api_keys(id) ON DELETE SET NULL",
		"user_id              BIGINT REFERENCES users(id) ON DELETE SET NULL",
		"group_id             BIGINT REFERENCES groups(id) ON DELETE SET NULL",
		"account_id           BIGINT REFERENCES accounts(id) ON DELETE SET NULL",
		"task_id              BIGINT REFERENCES video_tasks(id) ON DELETE SET NULL",
		"adoption_notes",
		"uq_ai_generation_content_apikey_request",
	} {
		if !strings.Contains(table, required) {
			t.Fatalf("182 migration missing %q", required)
		}
	}
	index := string(indexSQL)
	for _, required := range []string{"CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS", "ON ai_generation_content(task_id)", "WHERE task_id IS NOT NULL"} {
		if !strings.Contains(index, required) {
			t.Fatalf("183 migration missing %q", required)
		}
	}
}
