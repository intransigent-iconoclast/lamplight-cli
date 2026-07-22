package utils

import "strings"

type Format string

const (
	FormatEPUB      Format = "epub"
	FormatPDF       Format = "pdf"
	FormatMOBI      Format = "mobi"
	FormatAudiobook Format = "audiobook"
	FormatComic     Format = "comic"
	FormatUnknown   Format = "unknown"
)

// DetectFormat resolves the display format using three signals in priority order:
//  1. Explicit torznab:attr name="format" value from the indexer (most reliable)
//  2. Newznab/Torznab category ID (3030 = Audiobook, 7030 = Comics)
//  3. Title keyword heuristics as a fallback
func DetectFormat(formatAttr string, categories []int, title string) Format {
	// 1. Explicit format attr from the indexer (Libgen, etc.)
	if formatAttr != "" {
		switch formatAttr {
		case "epub":
			return FormatEPUB
		case "pdf":
			return FormatPDF
		case "mobi", "azw", "azw3":
			return FormatMOBI
		case "mp3", "m4b", "aax", "flac", "ogg":
			return FormatAudiobook
		case "cbr", "cbz":
			return FormatComic
		}
	}

	// 2. Category ID signals
	for _, cat := range categories {
		switch cat {
		case 3030: // Audio/Audiobook
			return FormatAudiobook
		case 7030: // Books/Comics
			return FormatComic
		}
	}

	// 3. Title heuristics (last resort)
	t := strings.ToLower(title)

	for _, kw := range []string{
		"audiobook", "audio book", "unabridged", "narrated by", "read by",
		".m4b", "[m4b]", "(m4b)", " m4b",
		".aax", "[aax]", "(aax)", " aax",
	} {
		if strings.Contains(t, kw) {
			return FormatAudiobook
		}
	}
	for _, kw := range []string{".cbr", "[cbr]", "(cbr)", ".cbz", "[cbz]", "(cbz)"} {
		if strings.Contains(t, kw) {
			return FormatComic
		}
	}
	for _, kw := range []string{".epub", "[epub]", "(epub)", " epub"} {
		if strings.Contains(t, kw) {
			return FormatEPUB
		}
	}
	for _, kw := range []string{".pdf", "[pdf]", "(pdf)", " pdf"} {
		if strings.Contains(t, kw) {
			return FormatPDF
		}
	}
	for _, kw := range []string{".mobi", "[mobi]", "(mobi)", "azw3", ".azw", "[azw]", "(azw)"} {
		if strings.Contains(t, kw) {
			return FormatMOBI
		}
	}

	return FormatUnknown
}
