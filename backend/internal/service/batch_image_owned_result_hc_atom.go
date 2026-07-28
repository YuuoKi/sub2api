package service

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// HCAtomOwnedResultStore is provider-neutral durable storage for completed HC
// result JSONL. It deliberately has no account, credential, or feature-toggle
// dependency so completed downloads and cleanup survive account retirement.
type HCAtomOwnedResultStore struct {
	rootDir string
}

func NewHCAtomOwnedResultStore(rootDir string) *HCAtomOwnedResultStore {
	return &HCAtomOwnedResultStore{rootDir: strings.TrimSpace(rootDir)}
}

func (s *HCAtomOwnedResultStore) RootDir() string {
	if s == nil {
		return ""
	}
	return s.rootDir
}

func (s *HCAtomOwnedResultStore) OpenReferenced(job *BatchImageJob) (io.ReadCloser, bool, error) {
	if job == nil || job.ProviderOutputRef == nil || strings.TrimSpace(*job.ProviderOutputRef) == "" {
		return nil, false, nil
	}
	if !strings.HasPrefix(strings.TrimSpace(*job.ProviderOutputRef), "hc_atom_owned:") {
		return nil, false, nil
	}
	if strings.TrimSpace(*job.ProviderOutputRef) != hcAtomOwnedResultRef(job.BatchID) {
		return nil, false, hcAtomBatchError("HC_ATOM_OWNED_RESULT_REF_INVALID", "HC-ATOM owned result reference is invalid", nil)
	}
	return s.openValidated(job)
}

func (s *HCAtomOwnedResultStore) Recover(job *BatchImageJob) (io.ReadCloser, bool, error) {
	if job == nil {
		return nil, false, nil
	}
	if job.ProviderOutputRef != nil && strings.TrimSpace(*job.ProviderOutputRef) != "" {
		return s.OpenReferenced(job)
	}
	r, ok, err := s.openValidated(job)
	if err != nil || !ok {
		return r, ok, err
	}
	ref := hcAtomOwnedResultRef(job.BatchID)
	job.ProviderOutputRef = &ref
	return r, true, nil
}

func (s *HCAtomOwnedResultStore) Write(job *BatchImageJob, data []byte) error {
	if s == nil || strings.TrimSpace(s.rootDir) == "" {
		return hcAtomBatchError("HC_ATOM_OWNED_RESULT_STORE_UNAVAILABLE", "HC-ATOM owned result store is unavailable", nil)
	}
	if err := validateHCAtomOwnedResultData(job, data); err != nil {
		return err
	}
	path, err := s.path(job)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return hcAtomBatchError("HC_ATOM_OWNED_RESULT_WRITE_FAILED", "HC-ATOM owned result archival failed", nil)
	}
	if err := os.Chmod(filepath.Dir(path), 0o700); err != nil {
		return hcAtomBatchError("HC_ATOM_OWNED_RESULT_WRITE_FAILED", "HC-ATOM owned result archival failed", nil)
	}
	if _, err := os.Stat(path); err == nil {
		ref := hcAtomOwnedResultRef(job.BatchID)
		job.ProviderOutputRef = &ref
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return hcAtomBatchError("HC_ATOM_OWNED_RESULT_WRITE_FAILED", "HC-ATOM owned result archival failed", nil)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".hc-atom-*.tmp")
	if err != nil {
		return hcAtomBatchError("HC_ATOM_OWNED_RESULT_WRITE_FAILED", "HC-ATOM owned result archival failed", nil)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return hcAtomBatchError("HC_ATOM_OWNED_RESULT_WRITE_FAILED", "HC-ATOM owned result archival failed", nil)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return hcAtomBatchError("HC_ATOM_OWNED_RESULT_WRITE_FAILED", "HC-ATOM owned result archival failed", nil)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return hcAtomBatchError("HC_ATOM_OWNED_RESULT_WRITE_FAILED", "HC-ATOM owned result archival failed", nil)
	}
	if err := tmp.Close(); err != nil {
		return hcAtomBatchError("HC_ATOM_OWNED_RESULT_WRITE_FAILED", "HC-ATOM owned result archival failed", nil)
	}
	if err := os.Rename(tmpName, path); err != nil {
		if _, statErr := os.Stat(path); statErr != nil {
			return hcAtomBatchError("HC_ATOM_OWNED_RESULT_WRITE_FAILED", "HC-ATOM owned result archival failed", nil)
		}
	}
	ref := hcAtomOwnedResultRef(job.BatchID)
	job.ProviderOutputRef = &ref
	return nil
}

func (s *HCAtomOwnedResultStore) DeleteReferenced(job *BatchImageJob) (bool, error) {
	if job == nil || job.ProviderOutputRef == nil || strings.TrimSpace(*job.ProviderOutputRef) == "" {
		return false, nil
	}
	ref := strings.TrimSpace(*job.ProviderOutputRef)
	if !strings.HasPrefix(ref, "hc_atom_owned:") {
		return false, nil
	}
	if ref != hcAtomOwnedResultRef(job.BatchID) {
		return true, hcAtomBatchError("HC_ATOM_OWNED_RESULT_REF_INVALID", "HC-ATOM owned result reference is invalid", nil)
	}
	path, err := s.path(job)
	if err != nil {
		return true, err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return true, hcAtomBatchError("HC_ATOM_OWNED_RESULT_CLEANUP_FAILED", "HC-ATOM owned result cleanup failed", nil)
	}
	return true, nil
}

func (s *HCAtomOwnedResultStore) openValidated(job *BatchImageJob) (io.ReadCloser, bool, error) {
	path, err := s.path(job)
	if err != nil {
		return nil, false, err
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, hcAtomBatchError("HC_ATOM_OWNED_RESULT_OPEN_FAILED", "HC-ATOM owned result is unavailable", nil)
	}
	if err := validateHCAtomOwnedResultReader(job, f); err != nil {
		_ = f.Close()
		return nil, false, err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		_ = f.Close()
		return nil, false, hcAtomBatchError("HC_ATOM_OWNED_RESULT_OPEN_FAILED", "HC-ATOM owned result is unavailable", nil)
	}
	return f, true, nil
}

func (s *HCAtomOwnedResultStore) path(job *BatchImageJob) (string, error) {
	if s == nil || strings.TrimSpace(s.rootDir) == "" {
		return "", hcAtomBatchError("HC_ATOM_OWNED_RESULT_STORE_UNAVAILABLE", "HC-ATOM owned result store is unavailable", nil)
	}
	if job == nil || !hcAtomSafeOwnedResultID(job.BatchID) {
		return "", hcAtomBatchError("HC_ATOM_OWNED_RESULT_PATH_INVALID", "HC-ATOM owned result path is invalid", nil)
	}
	root, err := filepath.Abs(s.rootDir)
	if err != nil {
		return "", hcAtomBatchError("HC_ATOM_OWNED_RESULT_PATH_INVALID", "HC-ATOM owned result path is invalid", nil)
	}
	path := filepath.Join(root, "hc_atom", job.BatchID+".jsonl")
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", hcAtomBatchError("HC_ATOM_OWNED_RESULT_PATH_INVALID", "HC-ATOM owned result path is invalid", nil)
	}
	return path, nil
}

func validateHCAtomOwnedResultData(job *BatchImageJob, data []byte) error {
	if len(data) == 0 || len(data) >= batchImageJSONLMaxLineBytes {
		return hcAtomBatchError("HC_ATOM_OWNED_RESULT_INVALID", "HC-ATOM owned result is invalid", nil)
	}
	return validateHCAtomOwnedResultReader(job, bytes.NewReader(data))
}

func validateHCAtomOwnedResultReader(job *BatchImageJob, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), batchImageJSONLMaxLineBytes)
	lineCount := 0
	expectedCustomID := batchImageProviderInputRef(job)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lineCount++
		if lineCount > 1 {
			return hcAtomBatchError("HC_ATOM_OWNED_RESULT_INVALID", "HC-ATOM owned result is invalid", nil)
		}
		parsed, err := ExtractBatchImagePartsFromResultLine([]byte(line))
		if err != nil || (expectedCustomID != "" && parsed.CustomID != expectedCustomID) {
			return hcAtomBatchError("HC_ATOM_OWNED_RESULT_INVALID", "HC-ATOM owned result is invalid", nil)
		}
	}
	if err := scanner.Err(); err != nil || lineCount != 1 {
		return hcAtomBatchError("HC_ATOM_OWNED_RESULT_INVALID", "HC-ATOM owned result is invalid", nil)
	}
	return nil
}
