package imageutil

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/draw"
)

// DefaultMaxDimension is the default maximum dimension for resizing
const DefaultMaxDimension = 1024

// ResizeConfig holds configuration for image resizing
type ResizeConfig struct {
	MaxDimension int  // Maximum width or height (default 1024)
	Quality      int  // JPEG quality 1-100 (default 85)
	OutputFormat string // "png" or "jpeg" (default "png")
}

// DefaultConfig returns default resize configuration
func DefaultConfig() *ResizeConfig {
	return &ResizeConfig{
		MaxDimension: DefaultMaxDimension,
		Quality:      85,
		OutputFormat: "png",
	}
}

// ResizeImage resizes an image if it exceeds the max dimension while maintaining aspect ratio
func ResizeImage(imageData []byte, config *ResizeConfig) ([]byte, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Decode the image
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Check if resizing is needed
	if width <= config.MaxDimension && height <= config.MaxDimension {
		// No resize needed, return original
		return imageData, nil
	}

	// Calculate new dimensions maintaining aspect ratio
	var newWidth, newHeight int
	if width > height {
		newWidth = config.MaxDimension
		newHeight = int(float64(height) * float64(config.MaxDimension) / float64(width))
	} else {
		newHeight = config.MaxDimension
		newWidth = int(float64(width) * float64(config.MaxDimension) / float64(height))
	}

	// Create new image with calculated dimensions
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Use high-quality resampling (CatmullRom is similar to Lanczos)
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	// Encode the resized image
	var buf bytes.Buffer
	outputFormat := config.OutputFormat
	if outputFormat == "" {
		outputFormat = format // Use original format if not specified
	}

	switch outputFormat {
	case "jpeg", "jpg":
		err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: config.Quality})
	case "png":
		err = png.Encode(&buf, dst)
	default:
		err = png.Encode(&buf, dst) // Default to PNG
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode resized image: %w", err)
	}

	return buf.Bytes(), nil
}

// ResizeImageReader resizes an image from an io.Reader
func ResizeImageReader(r io.Reader, config *ResizeConfig) ([]byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}
	return ResizeImage(data, config)
}
