package zeri

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateJPGThumbnail(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "source.png")
	destPath := filepath.Join(dir, "thumb.jpg")

	src := image.NewRGBA(image.Rect(0, 0, 320, 180))
	for y := 0; y < 180; y++ {
		for x := 0; x < 320; x++ {
			src.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 180, A: 255})
		}
	}
	srcFile, err := os.Create(sourcePath)
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	if err := png.Encode(srcFile, src); err != nil {
		_ = srcFile.Close()
		t.Fatalf("encode source: %v", err)
	}
	if err := srcFile.Close(); err != nil {
		t.Fatalf("close source: %v", err)
	}

	if err := CreateJPGThumbnail(sourcePath, destPath, 128); err != nil {
		t.Fatalf("CreateJPGThumbnail() error = %v", err)
	}

	thumbFile, err := os.Open(destPath)
	if err != nil {
		t.Fatalf("open thumb: %v", err)
	}
	defer thumbFile.Close()
	cfg, err := jpeg.DecodeConfig(thumbFile)
	if err != nil {
		t.Fatalf("decode thumb: %v", err)
	}
	if cfg.Width != 85 || cfg.Height != 128 {
		t.Fatalf("thumbnail size = %dx%d, want 85x128", cfg.Width, cfg.Height)
	}
}

func TestCreateJPGThumbnailWebP(t *testing.T) {
	testCreateJPGThumbnailFromFile(t, filepath.Join("testdata", "yellow_rose.lossless.webp"))
}

func TestCreateJPGThumbnailAVIF(t *testing.T) {
	testCreateJPGThumbnailFromFile(t, filepath.Join("testdata", "test8.avif"))
}

func testCreateJPGThumbnailFromFile(t *testing.T, sourceRelPath string) {
	t.Helper()
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, filepath.Base(sourceRelPath))
	destPath := filepath.Join(dir, "thumb.jpg")

	sourceBytes, err := os.ReadFile(sourceRelPath)
	if err != nil {
		t.Fatalf("read source fixture: %v", err)
	}
	if err := os.WriteFile(sourcePath, sourceBytes, 0o644); err != nil {
		t.Fatalf("write source fixture: %v", err)
	}

	if err := CreateJPGThumbnail(sourcePath, destPath, 128); err != nil {
		t.Fatalf("CreateJPGThumbnail() error = %v", err)
	}

	thumbFile, err := os.Open(destPath)
	if err != nil {
		t.Fatalf("open thumb: %v", err)
	}
	defer thumbFile.Close()
	cfg, err := jpeg.DecodeConfig(thumbFile)
	if err != nil {
		t.Fatalf("decode thumb: %v", err)
	}
	if cfg.Width != 85 || cfg.Height != 128 {
		t.Fatalf("thumbnail size = %dx%d, want 85x128", cfg.Width, cfg.Height)
	}
}
