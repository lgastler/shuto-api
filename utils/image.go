package utils

import (
	"errors"
	"fmt"
	"math"

	"github.com/davidbyttow/govips/v2/vips"
)

type ImageTransformOptions struct {
	Width         int
	Height        int
	Fit          string    // clip, crop, fill, max, min, scale
	Format       string    // jpg, png, webp
	Quality      int       // 0-100
	Dpr          float64   // 1.0-3.0
	Blur         int       // 0-100
	ForceDownload bool
}

// ImageUtils interface for image operations
type ImageUtils interface {
	GetMimeType(data []byte) (string, error)
	TransformImage(imgData []byte, opts ImageTransformOptions) ([]byte, error)
	GetImageDimensions(data []byte) (width int, height int, err error)
}

// imageUtils is the concrete implementation of ImageUtils
type imageUtils struct{}

func NewImageUtils() ImageUtils {
	return &imageUtils{}
}

func (iu *imageUtils) GetMimeType(data []byte) (string, error) {
	Debug("Getting MIME type for image data",
		"dataSize", len(data),
	)

	image, err := vips.NewImageFromBuffer(data)
	if err != nil {
		Error("Failed to create image from buffer",
			"error", err,
			"dataSize", len(data),
		)
		return "", fmt.Errorf("failed to create image from buffer: %v", err)
	}
	defer image.Close()

	format := image.Format()
	var mimeType string
	switch format {
	case vips.ImageTypeJPEG:
		mimeType = "image/jpeg"
	case vips.ImageTypePNG:
		mimeType = "image/png"
	case vips.ImageTypeWEBP:
		mimeType = "image/webp"
	}

	Debug("Determined image MIME type",
		"mimeType", mimeType,
		"format", format,
	)
	return mimeType, nil
}

// New function to transform images
func (iu *imageUtils) TransformImage(imgData []byte, opts ImageTransformOptions) ([]byte, error) {
	Debug("Starting image transformation",
		"inputSize", len(imgData),
		"width", opts.Width,
		"height", opts.Height,
		"fit", opts.Fit,
		"format", opts.Format,
		"quality", opts.Quality,
		"dpr", opts.Dpr,
		"blur", opts.Blur,
	)

	image, err := vips.NewImageFromBuffer(imgData)
	if err != nil {
		Error("Failed to read image",
			"error", err,
			"inputSize", len(imgData),
		)
		return nil, fmt.Errorf("failed to read image: %v", err)
	}
	defer image.Close()

	originalWidth, originalHeight := image.Width(), image.Height()
	Debug("Original image dimensions",
		"width", originalWidth,
		"height", originalHeight,
	)

	// Apply DPR scaling
	width := int(math.Round(float64(opts.Width) * opts.Dpr))
	height := int(math.Round(float64(opts.Height) * opts.Dpr))

	// Apply fit modes according to spec
	switch opts.Fit {
	case "clip", "":
		if width == 0 && height == 0 {
			width, height = image.Width(), image.Height()
		} else if width == 0 {
			scale := float64(height) / float64(image.Height())
			width = int(float64(image.Width()) * scale)
		} else if height == 0 {
			scale := float64(width) / float64(image.Width())
			height = int(float64(image.Height()) * scale)
		}

		scaleWidth := float64(width) / float64(image.Width())
		scaleHeight := float64(height) / float64(image.Height())
		scale := scaleWidth
		if scaleHeight < scaleWidth {
			scale = scaleHeight
		}

		Debug("Applying clip resize",
			"scale", scale,
			"targetWidth", width,
			"targetHeight", height,
		)

		if err := image.Resize(scale, vips.KernelAuto); err != nil {
			Error("Failed to resize image",
				"error", err,
				"scale", scale,
			)
			return nil, fmt.Errorf("failed to resize image: %w", err)
		}

	case "crop":
		Debug("Applying crop resize",
			"targetWidth", width,
			"targetHeight", height,
		)
		if err := image.Thumbnail(width, height, vips.InterestingCentre); err != nil {
			Error("Failed to crop image",
				"error", err,
				"width", width,
				"height", height,
			)
			return nil, fmt.Errorf("failed to crop image: %w", err)
		}

	case "fill":
		scale := float64(width) / float64(image.Width())
		Debug("Applying fill resize",
			"scale", scale,
			"targetWidth", width,
		)
		if err := image.Resize(scale, vips.KernelAuto); err != nil {
			Error("Failed to fill image",
				"error", err,
				"scale", scale,
			)
			return nil, fmt.Errorf("failed to fill image: %w", err)
		}

	default:
		Error("Invalid fit option provided",
			"fit", opts.Fit,
		)
		return nil, errors.New("invalid fit option")
	}

	// Apply blur if specified
	if opts.Blur > 0 {
		sigma := float64(opts.Blur) * 0.3
		Debug("Applying blur",
			"blur", opts.Blur,
			"sigma", sigma,
		)
		if err := image.GaussianBlur(sigma); err != nil {
			Error("Failed to apply blur",
				"error", err,
				"blur", opts.Blur,
				"sigma", sigma,
			)
			return nil, fmt.Errorf("failed to apply blur: %w", err)
		}
	}

	// Export with correct format
	var modifiedImg []byte
	Debug("Exporting image",
		"format", opts.Format,
		"quality", opts.Quality,
	)

	switch opts.Format {
	case "jpg", "jpeg":
		modifiedImg, _, err = image.ExportJpeg(&vips.JpegExportParams{Quality: opts.Quality})
	case "png":
		modifiedImg, _, err = image.ExportPng(&vips.PngExportParams{})
	case "webp":
		modifiedImg, _, err = image.ExportWebp(&vips.WebpExportParams{Quality: opts.Quality})
	default:
		modifiedImg, _, err = image.ExportJpeg(&vips.JpegExportParams{Quality: opts.Quality})
	}

	if err != nil {
		Error("Failed to export image",
			"error", err,
			"format", opts.Format,
		)
		return nil, fmt.Errorf("failed to export image: %w", err)
	}

	Debug("Image transformation completed",
		"outputSize", len(modifiedImg),
		"finalWidth", image.Width(),
		"finalHeight", image.Height(),
	)

	return modifiedImg, nil
}

func (iu *imageUtils) GetImageDimensions(data []byte) (width int, height int, err error) {
	Debug("Getting image dimensions",
		"dataSize", len(data),
	)

	image, err := vips.NewImageFromBuffer(data)
	if err != nil {
		Error("Failed to create image from buffer",
			"error", err,
			"dataSize", len(data),
		)
		return 0, 0, fmt.Errorf("failed to create image from buffer: %v", err)
	}
	defer image.Close()

	width = image.Width()
	height = image.Height()

	Debug("Retrieved image dimensions",
		"width", width,
		"height", height,
	)
	return width, height, nil
}
