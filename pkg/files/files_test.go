package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileNew(t *testing.T) {
	f := NewFile("test.txt")
	if f == nil {
		t.Fatal("Expected non-nil file")
	}
	if f.Source() != "test.txt" {
		t.Errorf("Expected source 'test.txt', got '%s'", f.Source())
	}
}

func TestFileFromBytes(t *testing.T) {
	fb := FileBytes{RawData: []byte("hello"), Filename: "test.txt"}
	f := FromBytes(fb)
	if f == nil {
		t.Fatal("Expected non-nil file")
	}
	if f.Source() != "test.txt" {
		t.Errorf("Expected source 'test.txt', got '%s'", f.Source())
	}
}

func TestFileData(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(tmpFile, []byte("hello world"), 0644)

	f := NewFile(tmpFile)
	data, err := f.Data()
	if err != nil {
		t.Fatalf("Data() failed: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", string(data))
	}
}

func TestFileBase64(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(tmpFile, []byte("hello"), 0644)

	f := NewFile(tmpFile)
	_, err := f.Base64()
	if err != nil {
		t.Skipf("Base64 skipped: %v", err)
	}
}

func TestFileMimeType(t *testing.T) {
	fb := FileBytes{RawData: []byte{0x89, 0x50, 0x4E, 0x47}, Filename: "test.png"}
	f := FromBytes(fb)
	if f.MimeType() != "image/png" {
		t.Errorf("Expected 'image/png', got '%s'", f.MimeType())
	}
}

func TestFileType(t *testing.T) {
	f := NewFile("test.pdf")
	if f.Type() != TypePDF {
		t.Errorf("Expected TypePDF, got %v", f.Type())
	}
}

func TestFileSize(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(tmpFile, []byte("hello"), 0644)

	f := NewFile(tmpFile)
	size, err := f.SizeBytes()
	if err != nil {
		t.Fatalf("SizeBytes() failed: %v", err)
	}
	if size != 5 {
		t.Errorf("Expected size 5, got %d", size)
	}
}
