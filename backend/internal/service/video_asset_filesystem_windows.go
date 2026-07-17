//go:build windows

package service

func newPlatformVideoAssetFilesystem(string) videoAssetFilesystem {
	// Go's portable os APIs cannot atomically reject every Windows reparse-point
	// substitution across a multi-component path. Fail closed instead of
	// pretending that string canonicalization provides a secure boundary.
	return unsupportedVideoAssetFilesystem{}
}
