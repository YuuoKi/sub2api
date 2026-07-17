package service

import (
	"errors"
	"os"
)

var errVideoAssetFilesystemUnsupported = errors.New("secure video asset filesystem is unsupported on this platform")

type videoAssetFilesystem interface {
	Supported() bool
	Create(int64) (*videoAssetTempFile, error)
	Open(int64) (*os.File, error)
}

type videoAssetTempFile struct {
	File     *os.File
	commitFn func() error
	abortFn  func()
}

func (f *videoAssetTempFile) Commit() error {
	if f == nil || f.commitFn == nil {
		return ErrVideoAssetDownload
	}
	return f.commitFn()
}

func (f *videoAssetTempFile) Abort() {
	if f != nil && f.abortFn != nil {
		f.abortFn()
	}
}

type unsupportedVideoAssetFilesystem struct{}

func (unsupportedVideoAssetFilesystem) Supported() bool { return false }
func (unsupportedVideoAssetFilesystem) Create(int64) (*videoAssetTempFile, error) {
	return nil, errVideoAssetFilesystemUnsupported
}
func (unsupportedVideoAssetFilesystem) Open(int64) (*os.File, error) {
	return nil, errVideoAssetFilesystemUnsupported
}
