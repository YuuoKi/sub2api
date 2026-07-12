package handler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanPageImageRelativePath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{name: "single filename", in: "logo.png", want: "logo.png", ok: true},
		{name: "nested path", in: "images/logo.png", want: filepath.Join("images", "logo.png"), ok: true},
		{name: "dot prefix", in: "./logo.png", want: "logo.png", ok: true},
		{name: "url escaped slash", in: "images%2Flogo.png", want: filepath.Join("images", "logo.png"), ok: true},
		{name: "parent traversal", in: "../secret.png", ok: false},
		{name: "encoded parent traversal", in: "%2e%2e/secret.png", ok: false},
		{name: "backslash traversal", in: `images\secret.png`, ok: false},
		{name: "absolute path", in: "/etc/passwd", ok: false},
		{name: "encoded absolute path", in: "%2fetc/passwd", ok: false},
		{name: "encoded nul byte", in: "logo.png%00", ok: false},
		{name: "invalid escape", in: "logo.png%zz", ok: false},
		{name: "empty path", in: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := cleanPageImageRelativePath(tt.in)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("path = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolvePageImagePath(t *testing.T) {
	root := t.TempDir()
	pagesDir := filepath.Join(root, "pages")
	base := filepath.Join(pagesDir, "guide")
	if err := os.MkdirAll(filepath.Join(base, "images"), 0755); err != nil {
		t.Fatalf("create images dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "logo.png"), []byte("fake"), 0644); err != nil {
		t.Fatalf("create direct image: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "images", "logo.png"), []byte("fake"), 0644); err != nil {
		t.Fatalf("create image: %v", err)
	}

	got, ok := resolvePageImagePath(pagesDir, base, "logo.png")
	if !ok {
		realPagesDir, pagesErr := filepath.EvalSymlinks(pagesDir)
		realBase, baseErr := filepath.EvalSymlinks(base)
		realTarget, targetErr := filepath.EvalSymlinks(filepath.Join(base, "logo.png"))
		t.Fatalf("expected direct image path to be accepted (pages=%q err=%v, base=%q err=%v, target=%q err=%v)", realPagesDir, pagesErr, realBase, baseErr, realTarget, targetErr)
	}
	want := filepath.Join(base, "logo.png")
	assertSameFile(t, got, want)

	got, ok = resolvePageImagePath(pagesDir, base, "images/logo.png")
	if !ok {
		t.Fatal("expected nested image path to be accepted")
	}
	want = filepath.Join(base, "images", "logo.png")
	assertSameFile(t, got, want)

	if got, ok := resolvePageImagePath(pagesDir, base, "../guide.md"); ok {
		t.Fatalf("expected traversal to be rejected, got %q", got)
	}
}

func assertSameFile(t *testing.T, gotPath, wantPath string) {
	t.Helper()
	gotInfo, err := os.Stat(gotPath)
	if err != nil {
		t.Fatalf("stat resolved path %q: %v", gotPath, err)
	}
	wantInfo, err := os.Stat(wantPath)
	if err != nil {
		t.Fatalf("stat expected path %q: %v", wantPath, err)
	}
	if !os.SameFile(gotInfo, wantInfo) {
		t.Fatalf("resolved path %q and expected path %q do not identify the same file", gotPath, wantPath)
	}
}

func TestResolvePageImagePathWithoutSymlinks(t *testing.T) {
	root := t.TempDir()
	pagesDir := filepath.Join(root, "pages")
	base := filepath.Join(pagesDir, "guide")
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(base, 0755); err != nil {
		t.Fatalf("create page dir: %v", err)
	}
	if err := os.MkdirAll(outside, 0755); err != nil {
		t.Fatalf("create outside dir: %v", err)
	}
	direct := filepath.Join(base, "logo.png")
	if err := os.WriteFile(direct, []byte("fake"), 0644); err != nil {
		t.Fatalf("create direct image: %v", err)
	}

	t.Run("accepts ordinary file", func(t *testing.T) {
		got, ok := resolvePageImagePathWithoutSymlinks(pagesDir, direct)
		if !ok || got != filepath.Clean(direct) {
			t.Fatalf("got (%q, %v), want (%q, true)", got, ok, filepath.Clean(direct))
		}
	})

	t.Run("rejects missing component", func(t *testing.T) {
		if got, ok := resolvePageImagePathWithoutSymlinks(pagesDir, filepath.Join(base, "missing.png")); ok {
			t.Fatalf("expected missing path to be rejected, got %q", got)
		}
	})

	t.Run("rejects symlink component", func(t *testing.T) {
		secret := filepath.Join(outside, "secret.png")
		if err := os.WriteFile(secret, []byte("secret"), 0644); err != nil {
			t.Fatalf("create outside file: %v", err)
		}
		link := filepath.Join(base, "linked")
		if err := os.Symlink(outside, link); err != nil {
			t.Skipf("symlink/junction creation unavailable on this platform or account: %v", err)
		}
		if got, ok := resolvePageImagePathWithoutSymlinks(pagesDir, filepath.Join(link, "secret.png")); ok {
			t.Fatalf("expected symlink path to be rejected, got %q", got)
		}
	})
}

func TestResolvePageImagePathRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	pagesDir := filepath.Join(root, "pages")
	base := filepath.Join(pagesDir, "guide")
	outside := filepath.Join(root, "outside")

	if err := os.MkdirAll(base, 0755); err != nil {
		t.Fatalf("create page dir: %v", err)
	}
	if err := os.MkdirAll(outside, 0755); err != nil {
		t.Fatalf("create outside dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.png"), []byte("secret"), 0644); err != nil {
		t.Fatalf("create outside file: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(base, "images")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	if got, ok := resolvePageImagePath(pagesDir, base, "images/secret.png"); ok {
		t.Fatalf("expected symlink escape to be rejected, got %q", got)
	}
}
