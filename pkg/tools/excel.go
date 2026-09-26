package tools

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/Ecook14/gocrewwai/pkg/utils"
)

// ExcelReadTool extracts text content from .xlsx files.
// XLSX is a ZIP archive containing XML; this parser handles shared strings
// and sheet data for common single-sheet workbooks.
type ExcelReadTool struct {
	BaseTool
	Chroot string
}

func NewExcelReadTool(chroot string) *ExcelReadTool {
	return &ExcelReadTool{
		BaseTool: BaseTool{
			NameValue:        "ExcelReadTool",
			DescriptionValue: "Reads and extracts text content from an Excel (.xlsx) file. Input requires 'file_path' as a string.",
		},
		Chroot: chroot,
	}
}

func (t *ExcelReadTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
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

	return extractXLSX(safePath)
}

func (t *ExcelReadTool) ArgsSchema() []ArgSchema {
	return []ArgSchema{
		{Name: "file_path", Type: "string", Description: "Path to the Excel (.xlsx) file to read", Required: true},
	}
}

// --- XLSX parsing (stdlib-only, no external deps) ---

const (
	xlsxSharedStrings = "xl/sharedStrings.xml"
	xlsxWorkbook      = "xl/workbook.xml"
	xlsxSheetDir      = "xl/_rels/workbook.xml.rels"
)

type sharedStringsXML struct {
	XMLName xml.Name     `xml:"sst"`
	Items   []sharedItem `xml:"si"`
}

type sharedItem struct {
	TextParts []textPart `xml:"t"`
}

type textPart struct {
	Text string `xml:",chardata"`
}

type workbookXML struct {
	Sheets []sheetInfo `xml:"sheets>sheet"`
}

type sheetInfo struct {
	Name  string `xml:"name,attr"`
	Id    string `xml:"sheetId,attr"`
	RId   string `xml:"r:id,attr"`
	State string `xml:"state,attr"`
}

type sheetRelsXML struct {
	Relationships []rel `xml:"Relationship"`
}

type rel struct {
	Id     string `xml:"Id,attr"`
	Type   string `xml:"Type,attr"`
	Target string `xml:"Target,attr"`
}

type sheetXML struct {
	Sheets []sheetData `xml:"sheetData"`
}

type sheetData struct {
	Rows []row `xml:"row"`
}

type row struct {
	Cells []cell `xml:"c"`
}

type cell struct {
	Ref string    `xml:"r,attr"`
	T   string    `xml:"t,attr,omitempty"`
	VM  string    `xml:"v,omitempty"`
	Is  inlineStr `xml:"is,omitempty"`
}

type inlineStr struct {
	T string `xml:"t,omitempty"`
}

func extractXLSX(path string) (string, error) {
	// Open ZIP
	zr, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("failed to open xlsx: %w", err)
	}
	defer zr.Close()

	// Read shared strings
	shared := make(map[int]string)
	for _, f := range zr.File {
		if f.Name == xlsxSharedStrings {
			data, err := readZipFile(f)
			if err != nil {
				break
			}
			var ss sharedStringsXML
			if err := xml.Unmarshal(data, &ss); err == nil {
				for i, item := range ss.Items {
					var sb strings.Builder
					for _, tp := range item.TextParts {
						sb.WriteString(tp.Text)
					}
					shared[i] = sb.String()
				}
			}
			break
		}
	}

	// List sheets
	var sheetNames []string
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			sheetNames = append(sheetNames, f.Name)
		}
	}

	var outLines []string
	outLines = append(outLines, fmt.Sprintf("Sheets: %d", len(sheetNames)))

	for _, sn := range sheetNames {
		data, err := readZipFileByPath(zr, sn)
		if err != nil {
			continue
		}
		content, err := parseSheet(data, shared)
		if err != nil {
			continue
		}
		outLines = append(outLines, "")
		outLines = append(outLines, fmt.Sprintf("--- %s ---", sn))
		outLines = append(outLines, content)
	}

	return strings.Join(outLines, "\n"), nil
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func readZipFileByPath(zr *zip.ReadCloser, name string) ([]byte, error) {
	for _, f := range zr.File {
		if f.Name == name {
			return readZipFile(f)
		}
	}
	return nil, fmt.Errorf("file %s not found in archive", name)
}

func parseSheet(data []byte, shared map[int]string) (string, error) {
	var sx sheetXML
	if err := xml.Unmarshal(data, &sx); err != nil {
		return "", err
	}
	var lines []string
	for _, sd := range sx.Sheets {
		for _, r := range sd.Rows {
			var rowCells []string
			for _, c := range r.Cells {
				text := cellValue(c, shared)
				if text != "" {
					rowCells = append(rowCells, text)
				}
			}
			if len(rowCells) > 0 {
				lines = append(lines, strings.Join(rowCells, "\t"))
			}
		}
	}
	return strings.Join(lines, "\n"), nil
}

func cellValue(c cell, shared map[int]string) string {
	if c.T == "s" {
		// Shared string
		var idx int
		fmt.Sscanf(c.VM, "%d", &idx)
		if s, ok := shared[idx]; ok {
			return s
		}
		return ""
	}
	if c.T == "inlineStr" && c.Is.T != "" {
		return c.Is.T
	}
	if c.VM != "" {
		return c.VM
	}
	return ""
}

// CacheFunction isolates cache entries per file path.
func (t *ExcelReadTool) CacheFunction(input map[string]interface{}) string {
	if p, ok := input["file_path"].(string); ok && p != "" {
		return "ExcelReadTool:" + p
	}
	return ""
}

var _ Tool = (*ExcelReadTool)(nil)
