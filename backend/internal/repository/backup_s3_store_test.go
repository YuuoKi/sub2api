package repository

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestSpoolBackupUploadBodyUsesTempFileAndReportsSize(t *testing.T) {
	content := bytes.Repeat([]byte("backup-data-"), 1024)

	file, sizeBytes, cleanup, err := spoolBackupUploadBody(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("spool upload body: %v", err)
	}
	name := file.Name()
	t.Cleanup(cleanup)

	if sizeBytes != int64(len(content)) {
		t.Fatalf("sizeBytes=%d, want %d", sizeBytes, len(content))
	}
	if _, err := os.Stat(name); err != nil {
		t.Fatalf("expected temp file before cleanup: %v", err)
	}
	readBack, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("read spooled body: %v", err)
	}
	if !bytes.Equal(readBack, content) {
		t.Fatal("spooled body content mismatch")
	}

	cleanup()
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		t.Fatalf("expected temp file removed after cleanup, got err=%v", err)
	}
}
