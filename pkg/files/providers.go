package files

import "fmt"

// ============================================================
// Provider Constraints Matrix
// ============================================================

// ProviderConstraints defines the file size and count limits for an LLM provider.
type ProviderConstraints struct {
	Provider       string
	ImageMaxBytes  int64
	ImageMaxCount  int
	ImageMaxWidth  int
	ImageMaxHeight int
	PDFMaxBytes    int64
	PDFMaxPages    int
	AudioMaxBytes  int64
	AudioMaxMinutes float64
	VideoMaxBytes  int64
	VideoMaxMinutes float64
	SupportsImage  bool
	SupportsPDF    bool
	SupportsAudio  bool
	SupportsVideo  bool
	SupportsText   bool
}

// Known provider constraints (from CrewAI Python documentation).
var Providers = map[string]ProviderConstraints{
	"openai": {
		Provider:       "openai",
		SupportsImage:  true,
		SupportsPDF:    true,
		SupportsAudio:  true,
		SupportsVideo:  false,
		SupportsText:   true,
		ImageMaxBytes:  20 * 1024 * 1024,  // 20 MB
		ImageMaxCount:  10,
		PDFMaxBytes:    32 * 1024 * 1024,  // 32 MB
		PDFMaxPages:    100,
		AudioMaxBytes:  25 * 1024 * 1024,  // 25 MB
		AudioMaxMinutes: 25,
	},
	"anthropic": {
		Provider:       "anthropic",
		SupportsImage:  true,
		SupportsPDF:    true,
		SupportsAudio:  false,
		SupportsVideo:  false,
		SupportsText:   true,
		ImageMaxBytes:  5 * 1024 * 1024,   // 5 MB
		ImageMaxCount:  100,
		ImageMaxWidth:  8000,
		ImageMaxHeight: 8000,
		PDFMaxBytes:    32 * 1024 * 1024,  // 32 MB
		PDFMaxPages:    100,
	},
	"gemini": {
		Provider:       "gemini",
		SupportsImage:  true,
		SupportsPDF:    true,
		SupportsAudio:  true,
		SupportsVideo:  true,
		SupportsText:   true,
		ImageMaxBytes:  100 * 1024 * 1024,  // 100 MB
		PDFMaxBytes:    50 * 1024 * 1024,   // 50 MB
		AudioMaxBytes:  100 * 1024 * 1024,  // 100 MB
		AudioMaxMinutes: 570,               // 9.5 hours
		VideoMaxBytes:  2 * 1024 * 1024 * 1024, // 2 GB
		VideoMaxMinutes: 60,
	},
	"bedrock": {
		Provider:       "bedrock",
		SupportsImage:  true,
		SupportsPDF:    true,
		SupportsAudio:  false,
		SupportsVideo:  false,
		SupportsText:   true,
		ImageMaxBytes:  4608 * 1024,        // 4.5 MB
		ImageMaxWidth:  8000,
		ImageMaxHeight: 8000,
		PDFMaxBytes:    3840 * 1024,         // 3.75 MB
		PDFMaxPages:    100,
	},
}

// ValidateFile checks if a file is compatible with a given provider.
func ValidateFile(file File, provider string) error {
	constraints, ok := Providers[provider]
	if !ok {
		return nil // Unknown provider — allow everything
	}

	ft := file.Type()

	// Check type support
	switch ft {
	case TypeImage:
		if !constraints.SupportsImage { return fmt.Errorf("provider %s does not support images", provider) }
	case TypePDF:
		if !constraints.SupportsPDF { return fmt.Errorf("provider %s does not support PDFs", provider) }
	case TypeAudio:
		if !constraints.SupportsAudio { return fmt.Errorf("provider %s does not support audio", provider) }
	case TypeVideo:
		if !constraints.SupportsVideo { return fmt.Errorf("provider %s does not support video", provider) }
	}

	// Check size
	size, err := file.SizeBytes()
	if err != nil {
		return nil // Can't check size
	}

	switch ft {
	case TypeImage:
		if constraints.ImageMaxBytes > 0 && size > constraints.ImageMaxBytes {
			return fmt.Errorf("image exceeds %s limit: %d bytes > %d bytes", provider, size, constraints.ImageMaxBytes)
		}
	case TypePDF:
		if constraints.PDFMaxBytes > 0 && size > constraints.PDFMaxBytes {
			return fmt.Errorf("PDF exceeds %s limit: %d bytes > %d bytes", provider, size, constraints.PDFMaxBytes)
		}
	case TypeAudio:
		if constraints.AudioMaxBytes > 0 && size > constraints.AudioMaxBytes {
			return fmt.Errorf("audio exceeds %s limit: %d bytes > %d bytes", provider, size, constraints.AudioMaxBytes)
		}
	case TypeVideo:
		if constraints.VideoMaxBytes > 0 && size > constraints.VideoMaxBytes {
			return fmt.Errorf("video exceeds %s limit: %d bytes > %d bytes", provider, size, constraints.VideoMaxBytes)
		}
	}

	return nil
}
