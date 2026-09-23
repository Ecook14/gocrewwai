package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/Ecook14/gocrewwai/pkg/utils"
	"github.com/ledongthuc/pdf"
)

// PDFReadTool allows agents to extract text content from PDF files.
type PDFReadTool struct {
	BaseTool
	Chroot string
}

func NewPDFReadTool(chroot string) *PDFReadTool {
	return &PDFReadTool{
		BaseTool: BaseTool{
			NameValue:        "PDFReadTool",
			DescriptionValue: "Reads and extracts text content from a PDF file. Input requires 'file_path' as a string.",
		},
		Chroot: chroot,
	}
}

func (t *PDFReadTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	pathRaw, ok := input["file_path"]
	if !ok {
		return "", fmt.Errorf("missing 'file_path' in input")
	}
	path, ok := pathRaw.(string)
	if !ok {
		return "", fmt.Errorf("'file_path' must be a string")
	}

	safePath, err := utils.ValidatePath(path, t.Chroot)
	if err != nil {
		return "", err
	}

	f, pdfReader, err := pdf.Open(safePath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	var pages []string
	totalPage := pdfReader.NumPage()
	for i := 1; i <= totalPage; i++ {
		p := pdfReader.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		pages = append(pages, text)
	}

	if len(pages) == 0 {
		return "(empty PDF)", nil
	}
	return strings.Join(pages, "\n--- Page Break ---\n"), nil
}

func (t *PDFReadTool) ArgsSchema() []ArgSchema {
	return []ArgSchema{
		{Name: "file_path", Type: "string", Description: "Path to the PDF file to read", Required: true},
	}
}
