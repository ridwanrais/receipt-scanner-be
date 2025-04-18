package util

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"strings"

	"github.com/disintegration/imaging"
)

// ImageProcessor handles image preprocessing for OCR
type ImageProcessor struct {
	// Configuration options could be added here
	MaxWidth  int
	MaxHeight int
}

// NewImageProcessor creates a new image processor
func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{
		MaxWidth:  2000, // Default max width
		MaxHeight: 2000, // Default max height
	}
}

// PreprocessImage performs preprocessing steps to improve OCR accuracy
// Returns the processed image as a byte array
func (p *ImageProcessor) PreprocessImage(imgData []byte) ([]byte, error) {
	// Decode the image
	img, format, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize if necessary
	bounds := img.Bounds()
	if bounds.Dx() > p.MaxWidth || bounds.Dy() > p.MaxHeight {
		img = imaging.Resize(img, p.MaxWidth, p.MaxHeight, imaging.Lanczos)
	}

	// Convert to grayscale
	grayImg := imaging.Grayscale(img)

	// Apply contrast enhancement
	contrastImg := imaging.AdjustContrast(grayImg, 20) // Increase contrast by 20%

	// Apply sharpening
	sharpenedImg := imaging.Sharpen(contrastImg, 1.0)

	// Binarize the image (convert to black and white)
	binarizedImg := binarize(sharpenedImg, 128)

	// Encode the processed image
	var buf bytes.Buffer
	
	// Normalize format string to lowercase
	format = strings.ToLower(format)
	
	// Handle different image formats
	switch format {
	case "jpeg", "jpg":
		err = jpeg.Encode(&buf, binarizedImg, &jpeg.Options{Quality: 95})
	case "png":
		err = png.Encode(&buf, binarizedImg)
	default:
		// Default to JPEG if format is unknown
		err = jpeg.Encode(&buf, binarizedImg, &jpeg.Options{Quality: 95})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode processed image: %w", err)
	}

	return buf.Bytes(), nil
}

// binarize converts an image to black and white using a threshold
func binarize(img image.Image, threshold uint8) *image.Gray {
	bounds := img.Bounds()
	grayImg := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			oldColor := img.At(x, y)
			grayColor := color.GrayModel.Convert(oldColor).(color.Gray)
			
			// Apply threshold
			if grayColor.Y > threshold {
				grayImg.Set(x, y, color.Gray{Y: 255}) // White
			} else {
				grayImg.Set(x, y, color.Gray{Y: 0}) // Black
			}
		}
	}

	return grayImg
}

// DetectSkew attempts to detect and correct skew in an image
// This is a simplified implementation and might need improvement for production
func (p *ImageProcessor) DetectSkew(img image.Image) (float64, error) {
	// A real implementation would use Hough transform or other algorithms
	// to detect lines and calculate skew angle
	// This is a placeholder
	return 0.0, nil
}

// DeskewImage rotates an image to correct skew
func (p *ImageProcessor) DeskewImage(img image.Image, angle float64) image.Image {
	// Convert angle to radians
	radians := angle * math.Pi / 180.0
	_ = radians // We'll use this in a more advanced implementation
	return imaging.Rotate(img, angle, color.White)
}

// SaveImageToTempFile saves an image to a temporary file and returns the path
func SaveImageToTempFile(imgData []byte, prefix string) (string, error) {
	tempFile, err := os.CreateTemp("", prefix+"*.jpg")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	if _, err := tempFile.Write(imgData); err != nil {
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}

	return tempFile.Name(), nil
}

// CleanupTempFile removes a temporary file
func CleanupTempFile(path string) error {
	return os.Remove(path)
}
