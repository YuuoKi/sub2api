//go:build !linux && !windows

package service

func newPlatformVideoAssetFilesystem(string) videoAssetFilesystem {
	return unsupportedVideoAssetFilesystem{}
}
