# Imagediff

`imagediff` is a Go-based command-line tool for computing and visualizing the pixel-wise difference between two images. It supports customizable output modes (color, grayscale, black-and-white), brightness/contrast normalization, and composite image generation with input images alongside the difference.

## Design

### Features

- **Difference Calculation**: Computes pixel-wise differences between two images, with options for:
  - Non-normalized (default): Simple absolute difference.
  - Normalized: Adjusts for brightness/contrast using mean and standard deviation.

- **Output Modes**:
  - Color (RGB): Default, shows differences per channel.
  - Grayscale: Averages differences into a single intensity.
  - Black-and-White: Binary output (white for any difference, black for none).

- **Scale Factor**: Customizable amplification of differences (default: 50.0).

- **Composite Output**: Optionally includes input images (left and right) with the difference (center).

- **Parallel Processing**: Splits the image into chunks processed concurrently using goroutines.

- **Viewer Integration**: Opens the result in a system-default or custom viewer, with an option to wait for closure.

- **Temporary Output**: Generates a temporary file if no output path is specified.

- **Verbose Logging**: Optional detailed logs for debugging and process tracking.

### Architecture

- **Core Logic**:
  - `computeDiffChunk`: Calculates differences for a chunk of the image, supporting all modes and scaling.
  - `createChunks`: Returns an `iter.Seq[Chunk]` iterator for parallel processing.
  - `createCompositeImage`: Combines input and difference images into a single output.

- **Concurrency**: Uses Go’s goroutines and channels for efficient parallel computation.

- **Thread Safety**:
  - `image.RGBA.Set` is not inherently thread-safe for concurrent writes to overlapping regions, but since `createChunks` ensures non-overlapping chunks (each goroutine processes a distinct region), this implementation is safe without additional synchronization.


- **Flags**: Command-line interface via Go’s `flag` package for configuration.

## Usage

### Basic Command

```bash
imagediff -left <left-image.png> -right <right-image.png>
```

- Computes a non-normalized RGB difference image and opens it in the default viewer.

- The output message includes the number of differing pixels, e.g., "Color difference image successfully created with scale factor 2.0: output.png (2.34% 1234 differing pixels)".

## Options

- `-left <file>`: Left input image file (required).

- `-right <file>`: Right input image file (required).

- `-output <file>`: Output image file (default: temporary file).

- `-diff-mode <mode>`: Difference mode:
  - `color`: RGB difference (default).
  - `gray`: Grayscale difference.
  - `bw`: Black-and-white difference.

- `-git-config <mode>`: Configure `imagediff` as git difftool:
  - `enable`: Sets `imagediff` as the git difftool.
  - `disable`: Removes `imagediff` from git difftool configuration.

- `-include-inputs`: Include input images in the output (left and right of diff).

- `-normalized`: Use normalized difference (adjusts for brightness/contrast).

- `-normalized-scale <float>`: Scale factor for amplifying differences in normalized mode (default: 50.0).

- `-scale <float>`: Scale factor for amplifying differences in non-normalized mode (default: 2.0).

- `-verbose`: Enable verbose logging for detailed process output.

- `-viewer <command>`: Custom image viewer command (e.g., gimp).

- `-wait`: Wait for the image viewer to close before exiting.

## Examples

```bash
# Grayscale difference with custom scale
imagediff -left image1.png -right image2.png -diff-mode gray -scale 100.0

# Normalized with custom normalized scale
imagediff -left image1.png -right image2.png -normalized -normalized-scale 25.0

# Normalized with defaults
imagediff -left image1.png -right image2.png -normalized

# Normalized black-and-white difference with composite output
imagediff -left image1.png -right image2.png -normalized -diff-mode bw -include-inputs -output diff.png

# Wait for viewer with verbose output
imagediff -left image1.png -right image2.png -wait -verbose

# Enable imagediff as git difftool
imagediff -git-config enable -verbose

# Disable imagediff as git difftool
imagediff -git-config disable
```

**The output message varies by mode**:
- **Non-normalized**: "Color difference image successfully created with scale factor 2.0: output.png (2.34% 1234 differing pixels)"
- **Normalized**: "Normalized Color difference image successfully created with scale factor 50.0: output.png"

**Sample output**:

```
2023/10/05 12:34:56 imagediff.go:123: Starting imagediff with left=image1.png, right=image2.png
2023/10/05 12:34:56 imagediff.go:135: Decoding left image
2023/10/05 12:34:56 imagediff.go:145: Decoding right image
2023/10/05 12:34:56 imagediff.go:169: Splitting image into 4 chunks (2x2)
2023/10/05 12:34:56 imagediff.go:75: Processing chunk: startX=0, endX=100, startY=0, endY=100
2023/10/05 12:34:56 imagediff.go:115: Difference at (10, 10): R=50, G=0, B=0
2023/10/05 12:34:56 imagediff.go:187: Collecting difference pixels
2023/10/05 12:34:56 imagediff.go:199: Created temporary output file: /tmp/imagediff-123456.png
2023/10/05 12:34:56 imagediff.go:215: Encoding image to /tmp/imagediff-123456.png
Color difference image successfully created with scale factor 50.0: /tmp/imagediff-123456.png (2.34% 1234 differing pixels)
2023/10/05 12:34:56 imagediff.go:230: Opening image with default system viewer: /tmp/imagediff-123456.png
Image opened in default macOS viewer
```

## Help Overview

Run `imagediff -h` to see the full help:

```
Usage of imagediff:
  -diff-mode string
        Difference mode: 'bw' (black-and-white), 'gray' (grayscale), 'color' (default) (default "color")
  -git-config string
        Configure imagediff as git difftool: 'enable' or 'disable'
  -include-inputs
        Include input images in output (left and right of diff)
  -left string
        Left input image file (required)
  -normalized
        Use normalized difference (adjusts for brightness/contrast)
  -normalized-scale float
        Scale factor for amplifying differences in normalized mode (default: 50.0) (default 50)
  -output string
        Output image file (default: temporary file)
  -right string
        Right input image file (required)
  -scale float
        Scale factor for amplifying differences in non-normalized mode (default: 2.0) (default 2)
  -verbose
        Enable verbose logging
  -viewer string
        Custom image viewer command (overrides default)
  -wait
        Wait for image viewer to close before exiting

Examples:
  Basic non-normalized difference:
    imagediff -left image1.png -right image2.png
  Normalized grayscale difference with custom scale:
    imagediff -left image1.png -right image2.png -normalized -diff-mode gray -normalized-scale 25.0
  Composite output with verbose logging:
    imagediff -left image1.png -right image2.png -include-inputs -verbose
  Configure as git difftool:
    imagediff -git-config enable
```

## Setup with Git `difftool`

You can configure `imagediff` as a custom diff tool in Git to visually compare image changes in a repository. Below are instructions for setting it up.

### Prerequisites

- Go installed (`go` command available).

- `imagediff` binary installed (see Installation below).

### Git Configuration with Default Viewer

1. **Edit Git Config:**

   Open your global Git configuration file:

   ```bash
   git config --global -e
   ```

   Add the following lines:

   ```
   [diff]
       tool = imagediff
   [difftool "imagediff"]
       cmd = imagediff -left \"$LOCAL\" -right \"$REMOTE\" -wait
   ```

   - `$LOCAL` and `$REMOTE` are Git-provided paths to the old and new versions of the file.

   - Outputs to `/tmp/imagediff_output.png` and waits for the viewer to close.

2. **Set as Default Difftool (Optional):**

   ```bash
   git config --global diff.tool imagediff
   ```

3. **Usage with Git:**

   - Compare changes in a tracked image file:

     ```bash
     git difftool <commit1> <commit2> -- <image-file>
     ```

     Or, if set as default:

     ```bash
     git difftool <image-file>
     ```

   - For staged changes:

     ```bash
     git difftool --staged <image-file>
     ```

### Alternatively, use the built-in flag to configure:

```bash
# Enable
# This configures: imagediff -left "$LOCAL" -right "$REMOTE" -wait -verbose
imagediff -git-config enable -verbose

# Disable
imagediff -git-config disable
```

This modifies your global `.gitconfig` automatically, using the binary's current path.

### Notes

- **Viewer Compatibility**: Ensure your default viewer (or custom viewer if specified) accepts a file path as an argument.

- **Temporary Files**: Without `-output`, a temporary file is used and cleaned up after the viewer closes.

### Git Configuration with Custom Viewer (e.g., FlowVision)

1. **Install FlowVision:**

   - Ensure FlowVision (or your preferred viewer) is installed and accessible via the command line. For example, if it’s a macOS or Windows application, you might need its full path (e.g., `/Applications/FlowVision.app/Contents/MacOS/FlowVision` or `"C:\\Program Files\\FlowVision\\flowvision.exe"`), or on Unix-like systems, ensure it’s in your PATH (e.g., `flowvision`).

2. **Edit Git Config:**

   Open your global Git configuration file:

   ```bash
   git config --global -e
   ```

   Add the following lines, replacing `<viewer-command>` with the actual command for FlowVision:

   ```
   [diff]
       tool = imagediff
   [difftool "imagediff"]
       cmd = imagediff -left \"$LOCAL\" -right \"$REMOTE\" -viewer <viewer-command> -wait
   ```

   - Example with FlowVision on macOS:

     ```
     cmd = imagediff -left \"$LOCAL\" -right \"$REMOTE\" -viewer /Applications/FlowVision.app/Contents/MacOS/FlowVision -wait
     ```

3. **Set as Default Difftool (Optional):**

   ```bash
   git config --global diff.tool imagediff
   ```

4. **Usage with Git:**

   - Same as above; the custom viewer (e.g., FlowVision) will open the difference image:

     ```bash
     git difftool <image-file>
     ```

### Notes

- **Viewer Compatibility**: Ensure FlowVision (or your viewer) accepts a file path as an argument. Test it standalone first (e.g., flowvision /tmp/test.png).

- **Path Issues**: On Windows, use escaped backslashes or forward slashes in the path. On Unix-like systems, ensure the command is executable and in PATH.

- **Temporary Files**: The `-output` path (`/tmp/` on Unix-like, adjust for Windows like `C:\\Temp\\`) must be writable.

- **Uncompiled Setup**: If not compiling, use:

  ```
  cmd = go run /path/to/imagediff.go -left \"$LOCAL\" -right \"$REMOTE\" -viewer flowvision -wait
  ```

## Installation

1. **Install from GitHub**:
   Clone or download the repository from GitHub and build the binary:

   ```bash
   git clone https://github.com/erdichen/imagediff.git
   cd imagediff
   go build -o imagediff
   ```

   Optionally, move the binary to a directory in your PATH (e.g., /usr/local/bin/):

   ```bash
   sudo mv imagediff /usr/local/bin/
   ```

   Alternatively, use Go's install command to build and place it in $GOPATH/bin:
   ```bash
   go install github.com/erdichen/imagediff@latest
   ```

   Ensure `$GOPATH/bin` is in your PATH (e.g., `export PATH=$PATH:$HOME/go/bin`).

2. Run Directly with Go: If you prefer not to build a binary, run it directly from the source:

   ```bash
   git clone https://github.com/erdichen/imagediff.git
   cd imagediff
   go run . -left image1.png -right image2.png
   ```

## Testing

Run the included unit tests:

```bash
go test
```

Tests cover core functions like difference calculation, chunking, and composite image generation.

## License

This project is open-source under the MIT License. See the LICENSE file for details.
