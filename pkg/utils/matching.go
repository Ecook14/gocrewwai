package utils

import (
	"strings"
	"unicode"
)

// Match budget: worst-case work is file-lines × target-lines per pass.
// Normalize once and cap inputs so modest attacker-controlled search text
// cannot force disproportionate CPU (CWE-407).
const (
	maxMatchFileBytes   = 2 << 20 // 2MiB workspace content
	maxMatchTargetBytes = 256 * 1024
	maxMatchLines       = 20000
)

// FindMatchingBlock implements a 4-pass heuristic search for a target block in text.
// Returns the start and end byte offsets of the match, and true if found.
func FindMatchingBlock(fullText string, search string) (start int, end int, found bool) {
	if search == "" {
		return 0, 0, false
	}
	if len(fullText) > maxMatchFileBytes || len(search) > maxMatchTargetBytes {
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
	if len(fullLines) > maxMatchLines || len(searchLines) > maxMatchLines {
		return 0, 0, false
	}

	// Normalize each line once instead of inside the nested loops.
	normFull := make([]string, len(fullLines))
	for i, l := range fullLines {
		normFull[i] = normLine(l, mode)
	}
	normSearch := make([]string, len(searchLines))
	for j, l := range searchLines {
		normSearch[j] = normLine(l, mode)
	}

	for i := 0; i <= len(fullLines)-len(searchLines); i++ {
		match := true
		for j := 0; j < len(searchLines); j++ {
			if normFull[i+j] != normSearch[j] {
				match = false
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

func normLine(s, mode string) string {
	switch mode {
	case "rstrip":
		return strings.TrimRightFunc(s, unicode.IsSpace)
	case "trim":
		return strings.TrimSpace(s)
	default:
		return normalize(s)
	}
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
