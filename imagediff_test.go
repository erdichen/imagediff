package main

import (
	"bytes"
	"image"
	"image/color"
	"io"
	"math"
	"os"
	"strings"
	"testing"
)

// Helper function to create a test image with solid color
func createTestImage(width, height int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestCalculateImageStats(t *testing.T) {
	tests := []struct {
		name      string
		img       image.Image
		wantMeanR float64
		wantMeanG float64
		wantMeanB float64
		wantMeanA float64
		wantStdR  float64
		wantStdG  float64
		wantStdB  float64
		wantStdA  float64
	}{
		{
			name:      "Solid White",
			img:       createTestImage(2, 2, color.RGBA{255, 255, 255, 255}),
			wantMeanR: 255.0, wantMeanG: 255.0, wantMeanB: 255.0, wantMeanA: 255.0,
			wantStdR: 0.0, wantStdG: 0.0, wantStdB: 0.0, wantStdA: 0.0,
		},
		{
			name: "Checkerboard",
			img: func() image.Image {
				img := image.NewRGBA(image.Rect(0, 0, 2, 2))
				img.Set(0, 0, color.RGBA{255, 255, 255, 255})
				img.Set(0, 1, color.RGBA{0, 0, 0, 255})
				img.Set(1, 0, color.RGBA{0, 0, 0, 255})
				img.Set(1, 1, color.RGBA{255, 255, 255, 255})
				return img
			}(),
			wantMeanR: 127.5, wantMeanG: 127.5, wantMeanB: 127.5, wantMeanA: 255.0,
			wantStdR: 127.5, wantStdG: 127.5, wantStdB: 127.5, wantStdA: 0.0,
		},
		{
			name:      "Single Pixel",
			img:       createTestImage(1, 1, color.RGBA{100, 150, 200, 255}),
			wantMeanR: 100.0, wantMeanG: 150.0, wantMeanB: 200.0, wantMeanA: 255.0,
			wantStdR: 0.0, wantStdG: 0.0, wantStdB: 0.0, wantStdA: 0.0,
		},
		{
			name:      "Transparent",
			img:       createTestImage(2, 2, color.RGBA{100, 100, 100, 0}),
			wantMeanR: 100.0, wantMeanG: 100.0, wantMeanB: 100.0, wantMeanA: 0.0,
			wantStdR: 0.0, wantStdG: 0.0, wantStdB: 0.0, wantStdA: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := calculateImageStats(tt.img)
			if !approxEqual(stats.meanR, tt.wantMeanR, 0.1) ||
				!approxEqual(stats.meanG, tt.wantMeanG, 0.1) ||
				!approxEqual(stats.meanB, tt.wantMeanB, 0.1) ||
				!approxEqual(stats.meanA, tt.wantMeanA, 0.1) {
				t.Errorf("%s: Means got R:%v G:%v B:%v A:%v, want R:%v G:%v B:%v A:%v",
					tt.name, stats.meanR, stats.meanG, stats.meanB, stats.meanA,
					tt.wantMeanR, tt.wantMeanG, tt.wantMeanB, tt.wantMeanA)
			}
			if !approxEqual(stats.stdR, tt.wantStdR, 0.1) ||
				!approxEqual(stats.stdG, tt.wantStdG, 0.1) ||
				!approxEqual(stats.stdB, tt.wantStdB, 0.1) ||
				!approxEqual(stats.stdA, tt.wantStdA, 0.1) {
				t.Errorf("%s: Stds got R:%v G:%v B:%v A:%v, want R:%v G:%v B:%v A:%v",
					tt.name, stats.stdR, stats.stdG, stats.stdB, stats.stdA,
					tt.wantStdR, tt.wantStdG, tt.wantStdB, tt.wantStdA)
			}
		})
	}
}

func TestNormalizePixel(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		mean  float64
		std   float64
		want  float64
	}{
		{name: "Normal", value: 100, mean: 50, std: 25, want: 2.0},
		{name: "Zero Std", value: 100, mean: 50, std: 0, want: 100.0},
		{name: "Negative Std", value: 100, mean: 150, std: -25, want: 2.0}, // Negative std treated as positive magnitude
		{name: "At Mean", value: 50, mean: 50, std: 10, want: 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizePixel(tt.value, tt.mean, tt.std)
			if !approxEqual(got, tt.want, 0.001) {
				t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestComputeDiffChunk(t *testing.T) {
	tests := []struct {
		name        string
		img1        image.Image
		img2        image.Image
		chunk       Chunk
		normalized  bool
		scaleFactor float64
		diffMode    string
		wantPixel   color.RGBA
		wantC1      int64
		wantC2      int64
		wantC3      int64
	}{
		{
			name:        "Color Non-Normalized",
			img1:        createTestImage(2, 2, color.RGBA{100, 150, 200, 255}),
			img2:        createTestImage(2, 2, color.RGBA{120, 170, 220, 255}),
			chunk:       Chunk{0, 2, 0, 2},
			normalized:  false,
			scaleFactor: 1.0,
			diffMode:    "color",
			wantPixel:   color.RGBA{20, 20, 20, 255},
			wantC1:      4, wantC2: 4, wantC3: 4,
		},
		{
			name:        "BW Identical",
			img1:        createTestImage(2, 2, color.RGBA{100, 100, 100, 255}),
			img2:        createTestImage(2, 2, color.RGBA{100, 100, 100, 255}),
			chunk:       Chunk{0, 2, 0, 2},
			normalized:  false,
			scaleFactor: 1.0,
			diffMode:    "bw",
			wantPixel:   color.RGBA{0, 0, 0, 255},
			wantC1:      4, wantC2: 4, wantC3: 0,
		},
		{
			name:        "Gray Difference",
			img1:        createTestImage(2, 2, color.RGBA{100, 100, 100, 255}),
			img2:        createTestImage(2, 2, color.RGBA{150, 150, 150, 255}),
			chunk:       Chunk{0, 2, 0, 2},
			normalized:  false,
			scaleFactor: 1.0,
			diffMode:    "gray",
			wantPixel:   color.RGBA{50, 50, 50, 255}, // (50+50+50)/3 = 50
			wantC1:      4, wantC2: 4, wantC3: 4,
		},
		{
			name:        "Normalized",
			img1:        createTestImage(2, 2, color.RGBA{100, 100, 100, 255}),
			img2:        createTestImage(2, 2, color.RGBA{200, 200, 200, 255}),
			chunk:       Chunk{0, 2, 0, 2},
			normalized:  true,
			scaleFactor: 50.0,
			diffMode:    "color",
			wantPixel:   color.RGBA{255, 255, 255, 255}, // Large diff due to normalization and scale
			wantC1:      4, wantC2: 4, wantC3: 4,
		},
		{
			name:        "Small Chunk",
			img1:        createTestImage(2, 2, color.RGBA{100, 100, 100, 255}),
			img2:        createTestImage(2, 2, color.RGBA{101, 101, 101, 255}),
			chunk:       Chunk{0, 1, 0, 1},
			normalized:  false,
			scaleFactor: 1.0,
			diffMode:    "color",
			wantPixel:   color.RGBA{1, 1, 1, 255},
			wantC1:      1, wantC2: 1, wantC3: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffImg := image.NewRGBA(tt.img1.Bounds())
			stats1 := calculateImageStats(tt.img1)
			stats2 := calculateImageStats(tt.img2)

			c1, c2, c3 := computeDiffChunk(tt.img1, tt.img2, diffImg, tt.chunk, stats1, stats2,
				tt.normalized, tt.scaleFactor, tt.diffMode, false)

			if c1 != tt.wantC1 || c2 != tt.wantC2 || c3 != tt.wantC3 {
				t.Errorf("%s: Counts got c1:%d c2:%d c3:%d, want %d %d %d",
					tt.name, c1, c2, c3, tt.wantC1, tt.wantC2, tt.wantC3)
			}

			gotPixel := diffImg.At(tt.chunk.startX, tt.chunk.startY).(color.RGBA)
			if gotPixel != tt.wantPixel {
				t.Errorf("%s: Pixel got %v, want %v", tt.name, gotPixel, tt.wantPixel)
			}
		})
	}
}

func TestCreateChunks(t *testing.T) {
	tests := []struct {
		name       string
		bounds     image.Rectangle
		numChunksX int
		numChunksY int
		wantCount  int
		wantLastX  int
		wantLastY  int
	}{
		{name: "2x2", bounds: image.Rect(0, 0, 100, 100), numChunksX: 2, numChunksY: 2,
			wantCount: 4, wantLastX: 100, wantLastY: 100},
		{name: "1x1", bounds: image.Rect(0, 0, 100, 100), numChunksX: 1, numChunksY: 1,
			wantCount: 1, wantLastX: 100, wantLastY: 100},
		{name: "Odd Size", bounds: image.Rect(0, 0, 101, 103), numChunksX: 2, numChunksY: 2,
			wantCount: 4, wantLastX: 101, wantLastY: 103},
		{name: "Tiny Image", bounds: image.Rect(0, 0, 2, 2), numChunksX: 2, numChunksY: 2,
			wantCount: 4, wantLastX: 2, wantLastY: 2},
		{name: "Large Chunks", bounds: image.Rect(0, 0, 100, 100), numChunksX: 10, numChunksY: 10,
			wantCount: 100, wantLastX: 100, wantLastY: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := createChunks(tt.bounds, tt.numChunksX, tt.numChunksY)
			count := 0
			var lastChunk Chunk
			for chunk := range chunks {
				count++
				lastChunk = chunk
				if chunk.startX >= chunk.endX || chunk.startY >= chunk.endY {
					t.Errorf("%s: Invalid chunk dimensions: %v", tt.name, chunk)
				}
			}
			if count != tt.wantCount {
				t.Errorf("%s: Expected %d chunks, got %d", tt.name, tt.wantCount, count)
			}
			if lastChunk.endX != tt.wantLastX || lastChunk.endY != tt.wantLastY {
				t.Errorf("%s: Last chunk ends at (%d,%d), want (%d,%d)",
					tt.name, lastChunk.endX, lastChunk.endY, tt.wantLastX, tt.wantLastY)
			}
		})
	}
}

func TestCreateCompositeImage(t *testing.T) {
	tests := []struct {
		name     string
		img1     image.Image
		img2     image.Image
		diffImg  image.Image
		wantSize image.Rectangle
		wantPix  []color.RGBA
	}{
		{
			name:     "Basic Composite",
			img1:     createTestImage(2, 2, color.RGBA{100, 0, 0, 255}),
			img2:     createTestImage(2, 2, color.RGBA{0, 100, 0, 255}),
			diffImg:  createTestImage(2, 2, color.RGBA{0, 0, 100, 255}),
			wantSize: image.Rect(0, 0, 6, 2),
			wantPix: []color.RGBA{
				{100, 0, 0, 255}, // img1 (left)
				{0, 0, 100, 255}, // diffImg (center)
				{0, 100, 0, 255}, // img2 (right)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			composite := createCompositeImage(tt.img1, tt.img2, tt.diffImg)
			if composite.Bounds() != tt.wantSize {
				t.Errorf("%s: got bounds %v, want %v", tt.name, composite.Bounds(), tt.wantSize)
			}
			for i, pos := range []struct{ x, y int }{{0, 0}, {2, 0}, {4, 0}} {
				gotPix := composite.At(pos.x, pos.y).(color.RGBA)
				if gotPix != tt.wantPix[i] {
					t.Errorf("%s: pixel at (%d,%d) got %v, want %v", tt.name, pos.x, pos.y, gotPix, tt.wantPix[i])
				}
			}
		})
	}
}

func TestPrintUsageWithExamples(t *testing.T) {
	// Capture stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	printUsageWithExamples()
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Examples:") || !strings.Contains(output, "imagediff") {
		t.Errorf("printUsageWithExamples() output missing expected content: %s", output)
	}
}

// Helper function for approximate float comparison
func approxEqual(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}
