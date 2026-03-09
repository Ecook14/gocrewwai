package files

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ============================================================
// File Types
// ============================================================

// FileType represents the category of a file.
type FileType string

const (
	TypeImage   FileType = "image"
	TypePDF     FileType = "pdf"
	TypeAudio   FileType = "audio"
	TypeVideo   FileType = "video"
	TypeText    FileType = "text"
	TypeGeneric FileType = "generic"
)

// FileMode controls how files are handled when they exceed provider limits.
type FileMode string

const (
	ModeStrict FileMode = "strict" // Error if file exceeds limits
	ModeAuto   FileMode = "auto"   // Auto-resize, compress, etc.
	ModeWarn   FileMode = "warn"   // Warn but attempt to send
	ModeChunk  FileMode = "chunk"  // Split large files into chunks
)

// TransmitMethod indicates how the file content is sent to the LLM.
type TransmitMethod string

const (
	TransmitInlineBase64 TransmitMethod = "inline_base64"
	TransmitFileUpload   TransmitMethod = "file_upload"
	TransmitURLReference TransmitMethod = "url_reference"
)

// ============================================================
// File Interface & Implementations
// ============================================================

// File is the base interface for all file types.
type File interface {
	Type() FileType
	Source() string
	Mode() FileMode
	Data() ([]byte, error)
	Base64() (string, error)
	MimeType() string
	SizeBytes() (int64, error)
	TransmitAs() TransmitMethod
}

// FileBytes wraps raw bytes with a filename for in-memory file creation.
type FileBytes struct {
	RawData  []byte
	Filename string
}

// baseFile provides shared implementation for all file types.
type baseFile struct {
	source   string
	mode     FileMode
	fileType FileType
	mimeType string
}

func (f *baseFile) Type() FileType       { return f.fileType }
func (f *baseFile) Source() string        { return f.source }
func (f *baseFile) Mode() FileMode       { return f.mode }
func (f *baseFile) MimeType() string      { return f.mimeType }

func (f *baseFile) Data() ([]byte, error) {
	if strings.HasPrefix(f.source, "http://") || strings.HasPrefix(f.source, "https://") {
		resp, err := http.Get(f.source)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch URL %s: %w", f.source, err)
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}
	return os.ReadFile(f.source)
}

func (f *baseFile) Base64() (string, error) {
	data, err := f.Data()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func (f *baseFile) SizeBytes() (int64, error) {
	if strings.HasPrefix(f.source, "http://") || strings.HasPrefix(f.source, "https://") {
		resp, err := http.Head(f.source)
		if err != nil {
			return 0, err
		}
		return resp.ContentLength, nil
	}
	info, err := os.Stat(f.source)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (f *baseFile) TransmitAs() TransmitMethod {
	if strings.HasPrefix(f.source, "http://") || strings.HasPrefix(f.source, "https://") {
		return TransmitURLReference
	}
	size, err := f.SizeBytes()
	if err != nil || size > 5*1024*1024 { // > 5MB
		return TransmitFileUpload
	}
	return TransmitInlineBase64
}

// ============================================================
// Typed File Constructors
// ============================================================

// ImageFile creates a file handle for an image.
func ImageFile(source string, mode ...FileMode) File {
	m := ModeAuto
	if len(mode) > 0 { m = mode[0] }
	return &baseFile{source: source, mode: m, fileType: TypeImage, mimeType: detectMIME(source, "image/png")}
}

// PDFFile creates a file handle for a PDF.
func PDFFile(source string, mode ...FileMode) File {
	m := ModeAuto
	if len(mode) > 0 { m = mode[0] }
	return &baseFile{source: source, mode: m, fileType: TypePDF, mimeType: "application/pdf"}
}

// AudioFile creates a file handle for audio content.
func AudioFile(source string, mode ...FileMode) File {
	m := ModeAuto
	if len(mode) > 0 { m = mode[0] }
	return &baseFile{source: source, mode: m, fileType: TypeAudio, mimeType: detectMIME(source, "audio/mpeg")}
}

// VideoFile creates a file handle for video content.
func VideoFile(source string, mode ...FileMode) File {
	m := ModeAuto
	if len(mode) > 0 { m = mode[0] }
	return &baseFile{source: source, mode: m, fileType: TypeVideo, mimeType: detectMIME(source, "video/mp4")}
}

// TextFile creates a file handle for text content.
func TextFile(source string, mode ...FileMode) File {
	m := ModeAuto
	if len(mode) > 0 { m = mode[0] }
	return &baseFile{source: source, mode: m, fileType: TypeText, mimeType: "text/plain"}
}

// NewFile auto-detects the file type from extension/content.
func NewFile(source string, mode ...FileMode) File {
	ext := strings.ToLower(filepath.Ext(source))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".bmp":
		return ImageFile(source, mode...)
	case ".pdf":
		return PDFFile(source, mode...)
	case ".mp3", ".wav", ".ogg", ".flac", ".m4a", ".aac":
		return AudioFile(source, mode...)
	case ".mp4", ".webm", ".avi", ".mov", ".mkv":
		return VideoFile(source, mode...)
	default:
		return TextFile(source, mode...)
	}
}

// FromBytes creates a file from raw bytes.
func FromBytes(fb FileBytes, mode ...FileMode) File {
	m := ModeAuto
	if len(mode) > 0 { m = mode[0] }
	ext := strings.ToLower(filepath.Ext(fb.Filename))
	ft := TypeGeneric
	mime := "application/octet-stream"
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		ft = TypeImage
		mime = "image/" + strings.TrimPrefix(ext, ".")
	case ".pdf":
		ft = TypePDF
		mime = "application/pdf"
	case ".mp3", ".wav":
		ft = TypeAudio
		mime = "audio/" + strings.TrimPrefix(ext, ".")
	case ".mp4", ".webm":
		ft = TypeVideo
		mime = "video/" + strings.TrimPrefix(ext, ".")
	}
	return &bytesFile{
		baseFile: baseFile{source: fb.Filename, mode: m, fileType: ft, mimeType: mime},
		rawData:  fb.RawData,
	}
}

// bytesFile wraps in-memory bytes as a File.
type bytesFile struct {
	baseFile
	rawData []byte
}

func (f *bytesFile) Data() ([]byte, error) { return f.rawData, nil }
func (f *bytesFile) SizeBytes() (int64, error) { return int64(len(f.rawData)), nil }

// detectMIME infers MIME type from file extension.
func detectMIME(source, fallback string) string {
	ext := strings.ToLower(filepath.Ext(source))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	default:
		return fallback
	}
}
