## Notes on the Tests

1. `TestCalculateImageStats`

   **Purpose**: Tests the calculateImageStats function, which computes the mean and standard deviation of RGBA channels for an image.

   **Description**:

   *   This test uses a table-driven approach to evaluate the function with different image inputs.

   *   It checks if the calculated means and standard deviations for red (R), green (G), blue (B), and alpha (A) channels match the expected values within a tolerance of 0.1.

   **Test Cases**:

   1.  **Solid White**:

       *   **Input**: A 2x2 image with all pixels set to `RGBA{255, 255, 255, 255}` (white, fully opaque).

       *   **Expected**: Mean of 255 for all channels (R, G, B, A) and standard deviation of 0 (no variation).

       *   **Purpose**: Verifies that uniform pixel values result in correct mean and zero variance.

   2.  **Checkerboard**:

       *   **Input**: A 2x2 image with alternating white (`255, 255, 255, 255`) and black (`0, 0, 0, 255`) pixels.

       *   **Expected**: Mean of 127.5 for R, G, B (average of two 255s and two 0s), 255 for A, and standard deviation of 127.5 for R, G, B (high variation), 0 for A.

       *   **Purpose**: Tests handling of varied pixel values and correct statistical computation.

   3.  **Single Pixel**:

       *   **Input**: A 1x1 image with `RGBA{100, 150, 200, 255}`.

       *   **Expected**: Mean matches the pixel values (`100, 150, 200, 255`) and standard deviation of 0 (single pixel has no variation).

       *   **Purpose**: Ensures the function works with minimal image size.

   4.  **Transparent**:

       *   **Input**: A 2x2 image with all pixels `RGBA{100, 100, 100, 0}` (gray, fully transparent).

       *   **Expected**: Mean of 100 for R, G, B, 0 for A, and standard deviation of 0 for all channels.

       *   **Purpose**: Verifies handling of transparent pixels.

   **Verification**:

   *   Compares calculated `meanR`, `meanG`, `meanB`, `meanA`, `stdR`, `stdG`, `stdB`, and `stdA` against expected values using `approxEqual` for floating-point precision.

   --------

2. `TestNormalizePixel`

   **Purpose**: Tests the `normalizePixel` function, which normalizes a pixel value based on its mean and standard deviation.

   **Description**:

   *   Uses a table-driven approach to test normalization under various conditions.

   *   Verifies that the output matches the expected normalized value within a tolerance of 0.001.

   **Test Cases**:

   1.  **Normal**:

       *   **Input**: Value=100, Mean=50, Std=25.

       *   **Expected**: 2.0 (computed as (`100 - 50) / 25`).

       *   **Purpose**: Tests typical normalization.

   2.  **Zero Std**:

       *   **Input**: Value=100, Mean=50, Std=0.

       *   **Expected**: 100.0 (returns original value when std is 0).

       *   **Purpose**: Ensures the function handles zero standard deviation correctly.

   3.  **Negative Std**:

       *   **Input**: Value=100, Mean=150, Std=-25.

       *   **Expected**: -2.0 (computed as (`100 - 150) / -25`).

       *   **Purpose**: Verifies that negative standard deviation works (though typically std is positive, this tests robustness).

   4.  **At Mean**:

       *   **Input**: Value=50, Mean=50, Std=10.

       *   **Expected**: 0.0 (computed as (`50 - 50) / 10`).

       *   **Purpose**: Ensures a value equal to the mean normalizes to 0.

   **Verification**:

   *   Compares the normalized result against the expected value using `approxEqual`.

   --------

3. `TestComputeDiffChunk`

   **Purpose**: Tests the `computeDiffChunk` function, which computes the difference between two images over a specified chunk and updates a difference image.

   **Description**:

   *   Tests different difference modes (`color`, `bw`, `gray`), normalization settings, and chunk sizes.

   *   Verifies pixel values in the output difference image and the counts of non-zero pixels (`c1`, `c2`, `c3`).

   **Test Cases**:

   1.  **Color Non-Normalized**:

       *   **Input**: Two 2x2 images (`RGBA{100, 150, 200, 255}` vs `RGBA{120, 170, 220, 255}`), full chunk, non-normalized, scale=1.0, mode=color.

       *   **Expected**: Pixel `RGBA{20, 20, 20, 255}`, counts `c1=4`, `c2=4`, `c3=4`.

       *   **Purpose**: Tests basic color difference calculation.

   2.  **BW Identical**:

       *   **Input**: Two identical 2x2 images (`RGBA{100, 100, 100, 255}`), full chunk, non-normalized, mode=`bw`.

       *   **Expected**: Pixel `RGBA{0, 0, 0, 255}`, counts `c1=4`, `c2=4`, `c3=0`.

       *   **Purpose**: Verifies black-and-white mode with no differences.

   3.  **Gray Difference**:

       *   **Input**: Two 2x2 images (`RGBA{100, 100, 100, 255}` vs `RGBA{150, 150, 150, 255}`), full chunk, non-normalized, mode=`gray`.

       *   **Expected**: Pixel `RGBA{50, 50, 50, 255}`, counts `c1=4`, `c2=4`, `c3=4`.

       *   **Purpose**: Tests grayscale mode with uniform differences.

   4.  **Normalized**:

       *   **Input**: Two 2x2 images (`RGBA{100, 100, 100, 255}` vs `RGBA{200, 200, 200, 255}`), full chunk, normalized, scale=50.0, mode=`color`.

       *   **Expected**: Pixel `RGBA{255, 255, 255, 255}`, counts `c1=4`, `c2=4`, `c3=4`.

       *   **Purpose**: Verifies normalized differences with amplification.

   5.  **Small Chunk**:

       *   **Input**: Two 2x2 images (`RGBA{100, 100, 100, 255}` vs `RGBA{101, 101, 101, 255}`), 1x1 chunk, non-normalized, mode=`color`.

       *   **Expected**: Pixel RGBA{1, 1, 1, 255}, counts c1=1, c2=1, c3=1.

       *   **Purpose**: Tests processing a partial image chunk.

   **Verification**:

   *   Checks the pixel at the chunk's starting position and the counts of non-zero pixels in both inputs (`c1`, `c2`) and differences (`c3`).

   --------

4. `TestCreateChunks`

   **Purpose**: Tests the createChunks function, which generates a sequence of chunks dividing an image.

   **Description**:

   *   Verifies that the function correctly splits an image into the specified number of chunks and covers the entire image area.

   **Test Cases**:

   1.  **2x2**:

       *   **Input**: 100x100 image, 2x2 chunks.

       *   **Expected**: 4 chunks, last chunk ends at (100, 100).

       *   **Purpose**: Tests standard chunking.

   2.  **1x1**:

       *   **Input**: 100x100 image, 1x1 chunks.

       *   **Expected**: 1 chunk, ends at (100, 100).

       *   **Purpose**: Tests single-chunk case.

   3.  **Odd Size**:

       *   **Input**: 101x103 image, 2x2 chunks.

       *   **Expected**: 4 chunks, last chunk ends at (101, 103).

       *   **Purpose**: Tests handling of non-divisible dimensions.

   4.  **Tiny Image**:

       *   **Input**: 2x2 image, 2x2 chunks.

       *   **Expected**: 4 chunks, last chunk ends at (2, 2).

       *   **Purpose**: Tests small image chunking.

   5.  **Large Chunks**:

       *   **Input**: 100x100 image, 10x10 chunks.

       *   **Expected**: 100 chunks, last chunk ends at (100, 100).

       *   **Purpose**: Tests high chunk count.

   **Verification**:

   *   Counts the number of chunks and ensures the last chunk reaches the image bounds, with valid dimensions (start < end).

   --------

5. `TestCreateCompositeImage`

   **Purpose**: Tests the createCompositeImage function, which creates a side-by-side composite of two input images and their difference.

   **Description**:

   *   Verifies the output image size (3x input width) and pixel placement from `img1`, `diffImg`, and `img2`.

   **Test Cases**:

   1.  **Basic Composite**:

       *   **Input**: 2x2 images: img1=`RGBA{100, 0, 0, 255}`, img2=`RGBA{0, 100, 0, 255}`, `diffImg=RGBA{0, 0, 100, 255}`.

       *   **Expected**: 6x2 image, pixels at (0,0)=`{100, 0, 0, 255}`, (2,0)=`{0, 0, 100, 255}`, (4,0)=`{0, 100, 0, 255}`.

       *   **Purpose**: Tests correct assembly of the composite image.

   **Verification**:

   *   Checks the bounds of the composite image and specific pixel values at key positions.

   --------

6. `TestUsage`

   **Purpose**: Tests the `usage` function, which prints usage instructions and examples.

   **Description**:

   *   Captures stdout to verify that the function outputs the expected help text.

   *   Does not test exact content beyond key phrases due to variability in flag output.

   **Test Case**:

   *   **Input**: None (just calls `usage()`).

   *   **Expected**: Output contains "Examples:" and "imagediff".

   *   **Purpose**: Ensures the usage message is printed with basic structure intact.

   **Verification**:

   *   Checks for presence of "Examples:" and "imagediff" in the captured output.

--------

### Helper Function: `approxEqual`

   *   **Purpose**: Compares floating-point numbers with a tolerance to account for precision errors.

   *   **Usage**: Used in `TestCalculateImageStats` and `TestNormalizePixel` to verify means, standard deviations, and normalized values.`
