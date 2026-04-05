package utils

import (
	"strings"
	"unicode"
)

// FindMatchingBlock implements a 4-pass heuristic search for a target block in text.
// Returns the start and end byte offsets of the match, and true if found.
func FindMatchingBlock(fullText string, search string) (start int, end int, found bool) {
	if search == "" {
		return 0, 0, false
	}

	// Pass 1: Exact Match
	idx := strings.Index(fullText, search)
	if idx != -1 {
		return idx, idx + len(search), true
	}

	// Pass 2: Line-by-line Rstrip Match
	if start, end, ok := matchLines(fullText, search, "rstrip"); ok {
		return start, end, true
	}

	// Pass 3: Line-by-line Trim Match
	if start, end, ok := matchLines(fullText, search, "trim"); ok {
		return start, end, true
	}

	// Pass 4: Unicode Normalization Match
	if start, end, ok := matchLines(fullText, search, "normalize"); ok {
		return start, end, true
	}

	return 0, 0, false
}

func matchLines(fullText, search, mode string) (int, int, bool) {
	fullLines := strings.Split(fullText, "\n")
	searchLines := strings.Split(search, "\n")
	
	if len(searchLines) == 0 {
		return 0, 0, false
	}

	for i := 0; i <= len(fullLines)-len(searchLines); i++ {
		match := true
		for j := 0; j < len(searchLines); j++ {
			f := fullLines[i+j]
			s := searchLines[j]

			switch mode {
			case "rstrip":
				if strings.TrimRightFunc(f, unicode.IsSpace) != strings.TrimRightFunc(s, unicode.IsSpace) {
					match = false
				}
			case "trim":
				if strings.TrimSpace(f) != strings.TrimSpace(s) {
					match = false
				}
			case "normalize":
				if normalize(f) != normalize(s) {
					match = false
				}
			}

			if !match {
				break
			}
		}

		if match {
			// Calculate exact byte offsets
			startIdx := 0
			for k := 0; k < i; k++ {
				startIdx += len(fullLines[k]) + 1 // +1 for the \n
			}
			
			endIdx := startIdx
			for k := i; k < i+len(searchLines); k++ {
				endIdx += len(fullLines[k]) + 1
			}
			
			// Adjust endIdx for trailing newline edge cases
			if endIdx > len(fullText) {
				endIdx = len(fullText)
			} else if endIdx > 0 && fullText[endIdx-1] == '\n' {
				// Only subtract if the search didn't end with a newline but the fullText does
				if !strings.HasSuffix(search, "\n") {
					endIdx--
				}
			}

			return startIdx, endIdx, true
		}
	}

	return 0, 0, false
}

func normalize(s string) string {
	s = strings.TrimSpace(s)
	// Replace common LLM-generated smart quotes and dashes
	s = strings.ReplaceAll(s, "“", "\"")
	s = strings.ReplaceAll(s, "”", "\"")
	s = strings.ReplaceAll(s, "‘", "'")
	s = strings.ReplaceAll(s, "’", "'")
	s = strings.ReplaceAll(s, "—", "-")
	return s
}
