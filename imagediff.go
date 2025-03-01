package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"iter"
	"log"
	"math"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"sync/atomic"
)

// Global flag pointer variables
var (
	leftPtr            = flag.String("left", "", "Left input image file (required)")
	rightPtr           = flag.String("right", "", "Right input image file (required)")
	outputPtr          = flag.String("output", "", "Output image file (default: temporary file)")
	waitPtr            = flag.Bool("wait", false, "Wait for image viewer to close before exiting")
	viewerPtr          = flag.String("viewer", "", "Custom image viewer command (overrides default)")
	includeInputsPtr   = flag.Bool("include-inputs", false, "Include input images in output (left and right of diff)")
	normalizedPtr      = flag.Bool("normalized", false, "Use normalized difference (adjusts for brightness/contrast)")
	scalePtr           = flag.Float64("scale", 2.0, "Scale factor for amplifying differences in non-normalized mode (default: 2.0)")
	normalizedScalePtr = flag.Float64("normalized-scale", 50.0, "Scale factor for amplifying differences in normalized mode (default: 50.0)")
	diffModePtr        = flag.String("diff-mode", "color", "Difference mode: 'bw' (black-and-white), 'gray' (grayscale), 'color' (default)")
	verbosePtr         = flag.Bool("verbose", false, "Enable verbose logging")
	gitConfigPtr       = flag.String("git-config", "", "Configure imagediff as git difftool: 'enable' or 'disable'")
)

type ImageStats struct {
	meanR, meanG, meanB, meanA float64
	stdR, stdG, stdB, stdA     float64
}

type Chunk struct {
	startX, endX int
	startY, endY int
}

func calculateImageStats(img image.Image) ImageStats {
	bounds := img.Bounds()
	var sumR, sumG, sumB, sumA float64
	count := float64(bounds.Dx() * bounds.Dy())

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			sumR += float64(r) / 257
			sumG += float64(g) / 257
			sumB += float64(b) / 257
			sumA += float64(a) / 257
		}
	}

	meanR := sumR / count
	meanG := sumG / count
	meanB := sumB / count
	meanA := sumA / count

	var sumSqDiffR, sumSqDiffG, sumSqDiffB, sumSqDiffA float64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			r8 := float64(r) / 257
			g8 := float64(g) / 257
			b8 := float64(b) / 257
			a8 := float64(a) / 257

			sumSqDiffR += (r8 - meanR) * (r8 - meanR)
			sumSqDiffG += (g8 - meanG) * (g8 - meanG)
			sumSqDiffB += (b8 - meanB) * (b8 - meanB)
			sumSqDiffA += (a8 - meanA) * (a8 - meanA)
		}
	}

	return ImageStats{
		meanR: meanR,
		meanG: meanG,
		meanB: meanB,
		meanA: meanA,
		stdR:  math.Sqrt(sumSqDiffR / count),
		stdG:  math.Sqrt(sumSqDiffG / count),
		stdB:  math.Sqrt(sumSqDiffB / count),
		stdA:  math.Sqrt(sumSqDiffA / count),
	}
}

func normalizePixel(value float64, mean float64, std float64) float64 {
	if std == 0 {
		return value
	}
	return (value - mean) / std
}

func computeDiffChunk(img1, img2 image.Image, diffImg *image.RGBA, chunk Chunk, stats1, stats2 ImageStats, normalized bool, scaleFactor float64, diffMode string, verbose bool) (int64, int64, int64) {
	if verbose {
		log.Printf("Processing chunk: startX=%d, endX=%d, startY=%d, endY=%d", chunk.startX, chunk.endX, chunk.startY, chunk.endY)
	}

	var count1, count2, diffCount int64

	// Calculate difference
	for y := chunk.startY; y < chunk.endY; y++ {
		for x := chunk.startX; x < chunk.endX; x++ {
			// Get colors from both images
			r1, g1, b1, a1 := img1.At(x, y).RGBA()
			r2, g2, b2, a2 := img2.At(x, y).RGBA()

			// Inverse of 8-bit to 16-bit conversion: (2^16 - 1) / (2^8 - 1) = 65535 / 255 ≈ 257
			r1f := float64(r1) / 257 // RGBA returns 16-bit values
			g1f := float64(g1) / 257
			b1f := float64(b1) / 257
			a1f := float64(a1) / 257
			r2f := float64(r2) / 257
			g2f := float64(g2) / 257
			b2f := float64(b2) / 257
			a2f := float64(a2) / 257

			if (r1 + g1 + b1 + a1) > 0 {
				count1++
			}
			if (r2 + g2 + b2 + a2) > 0 {
				count2++
			}

			var rDiff, gDiff, bDiff, aDiff float64
			if normalized {
				normR1 := normalizePixel(r1f, stats1.meanR, stats1.stdR)
				normG1 := normalizePixel(g1f, stats1.meanG, stats1.stdG)
				normB1 := normalizePixel(b1f, stats1.meanB, stats1.stdB)
				normA1 := normalizePixel(a1f, stats1.meanA, stats1.stdA)
				normR2 := normalizePixel(r2f, stats2.meanR, stats2.stdR)
				normG2 := normalizePixel(g2f, stats2.meanG, stats2.stdG)
				normB2 := normalizePixel(b2f, stats2.meanB, stats2.stdB)
				normA2 := normalizePixel(a2f, stats2.meanA, stats2.stdA)

				rDiff = math.Abs(normR1 - normR2)
				gDiff = math.Abs(normG1 - normG2)
				bDiff = math.Abs(normB1 - normB2)
				aDiff = math.Abs(normA1 - normA2)
			} else {
				// Calculate absolute differences
				rDiff = math.Abs(r1f - r2f)
				gDiff = math.Abs(g1f - g2f)
				bDiff = math.Abs(b1f - b2f)
				aDiff = math.Abs(a1f - a2f)
			}

			if (rDiff + gDiff + bDiff + aDiff) > 0 {
				diffCount++
			}

			// Ensure values stay within 8-bit range
			var r, g, b uint8
			switch diffMode {
			case "bw":
				// Black and white: any difference becomes white
				if rDiff > 0 || gDiff > 0 || bDiff > 0 {
					r, g, b = 255, 255, 255
				} else {
					r, g, b = 0, 0, 0
				}
			case "gray":
				// Grayscale: average the differences
				avgDiff := (rDiff + gDiff + bDiff) / 3.0
				gray := uint8(min(avgDiff*scaleFactor, 255))
				r, g, b = gray, gray, gray
			case "color":
				// RGB difference
				r = uint8(min(rDiff*scaleFactor, 255))
				g = uint8(min(gDiff*scaleFactor, 255))
				b = uint8(min(bDiff*scaleFactor, 255))
			default:
				// Default to color if unspecified or invalid
				r = uint8(min(rDiff*scaleFactor, 255))
				g = uint8(min(gDiff*scaleFactor, 255))
				b = uint8(min(bDiff*scaleFactor, 255))
			}

			a := uint8(255) // Hardcode alpha to 255 for full opacity

			// Set pixel in difference image
			diffImg.Set(x, y, color.RGBA{r, g, b, a})
		}
	}

	return count1, count2, diffCount
}

func createChunks(bounds image.Rectangle, numChunksX, numChunksY int) iter.Seq[Chunk] {
	return func(yield func(Chunk) bool) {
		width := bounds.Dx()
		height := bounds.Dy()

		chunkWidth := width / numChunksX
		chunkHeight := height / numChunksY

		for i := 0; i < numChunksY; i++ {
			for j := 0; j < numChunksX; j++ {
				startX := bounds.Min.X + j*chunkWidth
				startY := bounds.Min.Y + i*chunkHeight
				endX := startX + chunkWidth
				endY := startY + chunkHeight

				if j == numChunksX-1 {
					endX = bounds.Max.X
				}
				if i == numChunksY-1 {
					endY = bounds.Max.Y
				}

				chunk := Chunk{startX, endX, startY, endY}
				if !yield(chunk) {
					return
				}
			}
		}
	}
}

func openImage(filename, viewer string, wait, verbose bool) error {
	var cmd *exec.Cmd

	if viewer != "" {
		if verbose {
			log.Printf("Opening image with custom viewer: %s %s", viewer, filename)
		}
		// Use custom viewer
		if wait {
			cmd = exec.Command(viewer, filename)
			return cmd.Run() // Run waits for completion
		} else {
			cmd = exec.Command(viewer, filename)
			return cmd.Start() // Start doesn't wait
		}
	}

	// Use default system viewer
	if verbose {
		log.Printf("Opening image with default system viewer: %s", filename)
	}
	switch runtime.GOOS {
	case "darwin": // macOS
		if wait {
			cmd = exec.Command("open", "-W", filename)
		} else {
			cmd = exec.Command("open", filename)
		}
	case "linux": // Linux
		if wait {
			cmd = exec.Command("sh", "-c", fmt.Sprintf("xdg-open %q && wait", filename))
		} else {
			cmd = exec.Command("xdg-open", filename)
		}
	case "windows": // Windows
		if wait {
			cmd = exec.Command("cmd", "/c", "start", "/wait", filename)
		} else {
			cmd = exec.Command("cmd", "/c", "start", filename)
		}
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return cmd.Run()
}

// Helper function to get viewer name for output message
func getViewerName(viewer string) string {
	if viewer != "" {
		return fmt.Sprintf("custom viewer (%s)", viewer)
	}
	switch runtime.GOOS {
	case "darwin":
		return "default macOS viewer"
	case "linux":
		return "default Linux viewer"
	case "windows":
		return "default Windows viewer"
	default:
		return "default viewer"
	}
}

func createCompositeImage(img1, img2, diffImg image.Image) image.Image {
	bounds1 := img1.Bounds()
	width := bounds1.Dx() * 3 // Input1 + Diff + Input2 side by side
	height := bounds1.Dy()    // Same height as input images

	composite := image.NewRGBA(image.Rect(0, 0, width, height))

	// Draw first input image (left)
	draw.Draw(composite, image.Rect(0, 0, bounds1.Dx(), bounds1.Dy()), img1, bounds1.Min, draw.Src)

	// Draw difference image (center)
	draw.Draw(composite, image.Rect(bounds1.Dx(), 0, bounds1.Dx()*2, bounds1.Dy()), diffImg, bounds1.Min, draw.Src)

	// Draw second input image (right)
	draw.Draw(composite, image.Rect(bounds1.Dx()*2, 0, width, bounds1.Dy()), img2, bounds1.Min, draw.Src)

	return composite
}

func configureGitDifftool(enable bool, verbose bool) {
	toolName := "imagediff"
	binaryPath, err := os.Executable()
	if err != nil {
		if verbose {
			log.Printf("Error getting executable path: %v", err)
		} else {
			fmt.Printf("Error getting executable path: %v\n", err)
		}
		os.Exit(1)
	}

	if enable {
		if verbose {
			log.Printf("Enabling imagediff as git difftool with path: %s", binaryPath)
		}
		cmd := exec.Command("git", "config", "--global", "diff.tool", toolName)
		if err := cmd.Run(); err != nil {
			if verbose {
				log.Printf("Error setting diff.tool: %v", err)
			} else {
				fmt.Printf("Error setting diff.tool: %v\n", err)
			}
			os.Exit(1)
		}

		cmdStr := fmt.Sprintf("%s -left \"$LOCAL\" -right \"$REMOTE\" -wait", binaryPath)
		cmd = exec.Command("git", "config", "--global", fmt.Sprintf("difftool.%s.cmd", toolName), cmdStr)
		if err := cmd.Run(); err != nil {
			if verbose {
				log.Printf("Error setting difftool.%s.cmd: %v", toolName, err)
			} else {
				fmt.Printf("Error setting difftool.%s.cmd: %v\n", toolName, err)
			}
			os.Exit(1)
		}

		fmt.Println("imagediff successfully enabled as git difftool")
	} else {
		if verbose {
			log.Printf("Disabling imagediff as git difftool")
		}
		cmd := exec.Command("git", "config", "--global", "--unset", "diff.tool")
		if err := cmd.Run(); err != nil && err.Error() != "exit status 5" { // 5 means key not found, which is fine
			if verbose {
				log.Printf("Error unsetting diff.tool: %v", err)
			} else {
				fmt.Printf("Error unsetting diff.tool: %v\n", err)
			}
			os.Exit(1)
		}

		cmd = exec.Command("git", "config", "--global", "--unset", fmt.Sprintf("difftool.%s.cmd", toolName))
		if err := cmd.Run(); err != nil && err.Error() != "exit status 5" {
			if verbose {
				log.Printf("Error unsetting difftool.%s.cmd: %v", toolName, err)
			} else {
				fmt.Printf("Error unsetting difftool.%s.cmd: %v\n", toolName, err)
			}
			os.Exit(1)
		}

		fmt.Println("imagediff successfully disabled as git difftool")
	}
}

// printUsageWithExamples prints the standard flag usage followed by example runs
func printUsageWithExamples() {
	flag.CommandLine.SetOutput(os.Stderr) // Ensure usage goes to stderr
	flag.Usage()
	exe := os.Args[0]
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  Basic non-normalized difference:\n")
	fmt.Fprintf(os.Stderr, "    %s -left image1.png -right image2.png\n", exe)
	fmt.Fprintf(os.Stderr, "  Normalized grayscale difference with custom scale:\n")
	fmt.Fprintf(os.Stderr, "    %s -left image1.png -right image2.png -normalized -diff-mode gray -normalized-scale 25.0\n", exe)
	fmt.Fprintf(os.Stderr, "  Composite output with verbose logging:\n")
	fmt.Fprintf(os.Stderr, "    %s -left image1.png -right image2.png -include-inputs -verbose\n", exe)
	fmt.Fprintf(os.Stderr, "  Configure as git difftool:\n")
	fmt.Fprintf(os.Stderr, "    %s -git-config enable\n", exe)
	fmt.Fprintf(os.Stderr, "\n")
}

func main() {
	flag.CommandLine.Usage = printUsageWithExamples
	flag.Parse()

	if *verbosePtr {
		log.SetFlags(log.LstdFlags | log.Lshortfile) // Include timestamp and file:line
	}

	// Handle git-config flag
	if *gitConfigPtr != "" {
		if *gitConfigPtr == "enable" {
			configureGitDifftool(true, *verbosePtr)
			os.Exit(0)
		} else if *gitConfigPtr == "disable" {
			configureGitDifftool(false, *verbosePtr)
			os.Exit(0)
		} else {
			log.Printf("Error: Invalid -git-config value '%s'. Use 'enable' or 'disable'.", *gitConfigPtr)
			printUsageWithExamples()
			os.Exit(1)
		}
	}

	if *leftPtr == "" || *rightPtr == "" {
		log.Println("Error: Both left and right input files are required")
		printUsageWithExamples()
		os.Exit(1)
	}

	if *verbosePtr {
		log.Printf("Starting imagediff with left=%s, right=%s", *leftPtr, *rightPtr)
	}

	// Validate diffMode
	if *diffModePtr != "bw" && *diffModePtr != "gray" && *diffModePtr != "color" {
		log.Printf("Error: Invalid -diff-mode value '%s'. Use 'bw', 'gray', or 'color'.", *diffModePtr)
		printUsageWithExamples()
		os.Exit(1)
	}

	// Open the left image
	img1File, err := os.Open(*leftPtr)
	if err != nil {
		if *verbosePtr {
			log.Printf("Error opening left image file %s: %v", *leftPtr, err)
		} else {
			fmt.Printf("Error opening left image: %v\n", err)
		}
		os.Exit(1)
	}
	defer img1File.Close()

	// Open the right image
	img2File, err := os.Open(*rightPtr)
	if err != nil {
		if *verbosePtr {
			log.Printf("Error opening right image file %s: %v", *rightPtr, err)
		} else {
			fmt.Printf("Error opening right image: %v\n", err)
		}
		os.Exit(1)
	}
	defer img2File.Close()

	if *verbosePtr {
		log.Println("Decoding left image")
	}
	img1, _, err := image.Decode(img1File)
	if err != nil {
		if *verbosePtr {
			log.Printf("Error decoding left image: %v", err)
		} else {
			fmt.Printf("Error decoding left image: %v\n", err)
		}
		os.Exit(1)
	}

	if *verbosePtr {
		log.Println("Decoding right image")
	}
	img2, _, err := image.Decode(img2File)
	if err != nil {
		if *verbosePtr {
			log.Printf("Error decoding right image: %v", err)
		} else {
			fmt.Printf("Error decoding right image: %v\n", err)
		}
		os.Exit(1)
	}

	// Check if images have the same dimensions
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()
	if bounds1 != bounds2 {
		log.Println("Error: Images must have the same dimensions")
		os.Exit(1)
	}

	var stats1, stats2 ImageStats
	if *normalizedPtr {
		if *verbosePtr {
			log.Println("Calculating statistics for left image")
		}
		stats1 = calculateImageStats(img1)
		if *verbosePtr {
			log.Println("Calculating statistics for right image")
		}
		stats2 = calculateImageStats(img2)
	}

	// Create output image
	diffImg := image.NewRGBA(bounds1)
	scaleFactor := *scalePtr
	if *normalizedPtr {
		scaleFactor = *normalizedScalePtr
	}

	numCPU := runtime.NumCPU()
	numChunksX := int(math.Sqrt(float64(numCPU)))
	numChunksY := numCPU / numChunksX
	if numChunksX*numChunksY < numCPU {
		numChunksY++
	}

	minChunkSize := 32
	if bounds1.Dx()/numChunksX < minChunkSize {
		numChunksX = bounds1.Dx() / minChunkSize
	}
	if bounds1.Dy()/numChunksY < minChunkSize {
		numChunksY = bounds1.Dy() / minChunkSize
	}
	if numChunksX < 1 {
		numChunksX = 1
	}
	if numChunksY < 1 {
		numChunksY = 1
	}

	if *verbosePtr {
		log.Printf("Splitting image into %d chunks (%dx%d)", numChunksX*numChunksY, numChunksX, numChunksY)
	}

	var wg sync.WaitGroup
	var count1, count2, diffCount int64

	for chunk := range createChunks(bounds1, numChunksX, numChunksY) {
		wg.Add(1)
		go func(c Chunk) {
			defer wg.Done()
			c1, c2, c3 := computeDiffChunk(img1, img2, diffImg, c, stats1, stats2, *normalizedPtr, scaleFactor, *diffModePtr, *verbosePtr)
			atomic.AddInt64(&count1, c1)
			atomic.AddInt64(&count2, c2)
			atomic.AddInt64(&diffCount, c3)
		}(chunk)
	}

	wg.Wait() // Wait for all chunks to complete

	if *verbosePtr {
		log.Printf("Non-zero pixels left %v right %v diff %v\n", count1, count2, diffCount)
	}

	// Handle output file
	outputFile := *outputPtr
	if outputFile == "" {
		// Create a temporary file
		tmpFile, err := os.CreateTemp("", "imagediff-*.png")
		if err != nil {
			if *verbosePtr {
				log.Printf("Error creating temporary file: %v", err)
			} else {
				fmt.Printf("Error creating temporary file: %v\n", err)
			}
			os.Exit(1)
		}
		outputFile = tmpFile.Name()
		defer tmpFile.Close()
		if *verbosePtr {
			log.Printf("Created temporary output file: %s", outputFile)
		}
	}

	// Create output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		if *verbosePtr {
			log.Printf("Error creating output file %s: %v", outputFile, err)
		} else {
			fmt.Printf("Error creating output file: %v\n", err)
		}
		os.Exit(1)
	}
	defer outFile.Close()

	// Decide which image to save
	var finalImg image.Image = diffImg
	if *includeInputsPtr {
		if *verbosePtr {
			log.Println("Creating composite image with inputs")
		}
		finalImg = createCompositeImage(img1, img2, diffImg)
	}

	// Encode and save the difference image
	if *verbosePtr {
		log.Printf("Encoding image to %s", outputFile)
	}
	err = png.Encode(outFile, finalImg)
	if err != nil {
		if *verbosePtr {
			log.Printf("Error encoding output image: %v", err)
		} else {
			fmt.Printf("Error encoding output image: %v\n", err)
		}
		os.Exit(1)
	}

	diffType := ""
	diffMsg := ""
	if *normalizedPtr {
		diffType = "Normalized "
	} else {
		diffPercent := float64(diffCount) * 100 / float64(bounds1.Dx()*bounds1.Dy())
		diffMsg = fmt.Sprintf(" (%.2f%% %d differing pixels)", diffPercent, diffCount)
	}
	outputMode := "Color"
	if *diffModePtr == "bw" {
		outputMode = "Black-and-White"
	} else if *diffModePtr == "gray" {
		outputMode = "Grayscale"
	}
	fmt.Printf("%s%s difference image successfully created with scale factor %.1f: %s%s\n", diffType, outputMode, scaleFactor, outputFile, diffMsg)

	err = openImage(outputFile, *viewerPtr, *waitPtr, *verbosePtr)
	if err != nil {
		if *verbosePtr {
			log.Printf("Error opening image: %v", err)
		} else {
			fmt.Printf("Error opening image: %v\n", err)
		}
		os.Exit(1)
	}
	if *verbosePtr {
		if *waitPtr {
			fmt.Println("Image viewer closed")
		} else {
			fmt.Println("Image opened in", getViewerName(*viewerPtr))
		}
	}
}
