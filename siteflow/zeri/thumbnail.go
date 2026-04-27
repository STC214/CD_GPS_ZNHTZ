package zeri

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"os"
	"path/filepath"

	_ "github.com/gen2brain/avif"
	_ "golang.org/x/image/webp"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// CreateJPGThumbnail creates a portrait JPG thumbnail with a manga-cover aspect ratio.
func CreateJPGThumbnail(sourcePath, destPath string, maxSize int) error {
	if maxSize <= 0 {
		maxSize = 256
	}
	thumbW, thumbH := thumbnailDimensions(maxSize)

	sourcePath = filepath.Clean(sourcePath)
	destPath = filepath.Clean(destPath)
	srcFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open thumbnail source %q: %w", sourcePath, err)
	}
	defer srcFile.Close()
	log.Printf("thumbnail start: source=%s dest=%s size=%d", sourcePath, destPath, maxSize)

	src, _, err := image.Decode(srcFile)
	if err != nil {
		return fmt.Errorf("decode thumbnail source %q: %w", sourcePath, err)
	}

	bounds := src.Bounds()
	if bounds.Empty() {
		return fmt.Errorf("thumbnail source %q is empty", sourcePath)
	}

	crop := coverCrop(bounds, src, thumbW, thumbH)
	canvas := image.NewRGBA(image.Rect(0, 0, thumbW, thumbH))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	scaleNearest(canvas, canvas.Bounds(), cropImage(src, crop))

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create thumbnail dir %q: %w", filepath.Dir(destPath), err)
	}
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create thumbnail %q: %w", destPath, err)
	}
	defer out.Close()

	if err := jpeg.Encode(out, canvas, &jpeg.Options{Quality: 85}); err != nil {
		return fmt.Errorf("encode thumbnail %q: %w", destPath, err)
	}
	log.Printf("thumbnail complete: %s", destPath)
	return nil
}

func thumbnailDimensions(maxSize int) (int, int) {
	thumbH := maxSize
	thumbW := (maxSize * 2) / 3
	if thumbW <= 0 {
		thumbW = 1
	}
	if thumbH <= 0 {
		thumbH = 1
	}
	return thumbW, thumbH
}

func coverCrop(bounds image.Rectangle, src image.Image, targetW, targetH int) image.Rectangle {
	if bounds.Empty() {
		return bounds
	}
	ratio := float64(targetW) / float64(targetH)
	selected := bounds
	if bounds.Dx() > bounds.Dy() {
		left, right := splitVerticalHalf(bounds)
		if contentScore(src, right) > contentScore(src, left) {
			selected = right
		} else {
			selected = left
		}
	}
	return cropToAspect(selected, ratio)
}

func splitVerticalHalf(bounds image.Rectangle) (image.Rectangle, image.Rectangle) {
	w := bounds.Dx()
	if w <= 1 {
		return bounds, bounds
	}
	half := w / 2
	if half <= 0 {
		half = 1
	}
	left := image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Min.X+half, bounds.Max.Y)
	right := image.Rect(bounds.Min.X+half, bounds.Min.Y, bounds.Max.X, bounds.Max.Y)
	return left, right
}

func contentScore(src image.Image, rect image.Rectangle) int {
	if rect.Empty() {
		return 0
	}
	stepX := rect.Dx() / 16
	stepY := rect.Dy() / 16
	if stepX < 1 {
		stepX = 1
	}
	if stepY < 1 {
		stepY = 1
	}
	score := 0
	for y := rect.Min.Y; y < rect.Max.Y; y += stepY {
		for x := rect.Min.X; x < rect.Max.X; x += stepX {
			r, g, b, a := src.At(x, y).RGBA()
			if a < 0x8000 {
				score += 3
				continue
			}
			if r < 0xf000 || g < 0xf000 || b < 0xf000 {
				score++
			}
		}
	}
	return score
}

func cropToAspect(bounds image.Rectangle, ratio float64) image.Rectangle {
	if bounds.Empty() {
		return bounds
	}
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 0 || h <= 0 || ratio <= 0 {
		return bounds
	}
	current := float64(w) / float64(h)
	if current > ratio {
		targetW := int(float64(h) * ratio)
		if targetW < 1 {
			targetW = 1
		}
		if targetW > w {
			targetW = w
		}
		left := bounds.Min.X + (w-targetW)/2
		return image.Rect(left, bounds.Min.Y, left+targetW, bounds.Max.Y)
	}
	if current < ratio {
		targetH := int(float64(w) / ratio)
		if targetH < 1 {
			targetH = 1
		}
		if targetH > h {
			targetH = h
		}
		top := bounds.Min.Y + (h-targetH)/2
		return image.Rect(bounds.Min.X, top, bounds.Max.X, top+targetH)
	}
	return bounds
}

func cropImage(src image.Image, rect image.Rectangle) image.Image {
	if rect.Empty() {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	if sub, ok := src.(interface {
		SubImage(image.Rectangle) image.Image
	}); ok {
		return sub.SubImage(rect)
	}
	dst := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(dst, dst.Bounds(), src, rect.Min, draw.Src)
	return dst
}

func scaleNearest(dst *image.RGBA, dstRect image.Rectangle, src image.Image) {
	if dstRect.Empty() {
		return
	}
	srcBounds := src.Bounds()
	dstW := dstRect.Dx()
	dstH := dstRect.Dy()
	if dstW <= 0 || dstH <= 0 {
		return
	}
	for y := 0; y < dstH; y++ {
		sy := srcBounds.Min.Y + int(float64(y)*float64(srcBounds.Dy())/float64(dstH))
		if sy >= srcBounds.Max.Y {
			sy = srcBounds.Max.Y - 1
		}
		for x := 0; x < dstW; x++ {
			sx := srcBounds.Min.X + int(float64(x)*float64(srcBounds.Dx())/float64(dstW))
			if sx >= srcBounds.Max.X {
				sx = srcBounds.Max.X - 1
			}
			dst.Set(dstRect.Min.X+x, dstRect.Min.Y+y, src.At(sx, sy))
		}
	}
}
