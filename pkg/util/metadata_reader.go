package utils

import (
	"archive/zip"
	"encoding/binary"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// BookMetadata holds whatever we could pull out of a file.
// Empty string means we didn't find it.
type BookMetadata struct {
	Title     string
	Author    string
	Year      string
	Publisher string
	ISBN      string
	Format    string // epub, pdf, mobi, mp3, m4b, azw3, cbz, cbr …
}

// ReadMetadata reads metadata from a book file.
// Falls back to filename parsing if deep reading isn't supported or fails.
func ReadMetadata(path string) (*BookMetadata, error) {
	ext := strings.ToLower(filepath.Ext(path))
	meta := &BookMetadata{Format: FormatFromExt(ext)}

	var err error
	switch meta.Format {
	case "epub":
		err = readEPUB(path, meta)
	case "mp3":
		err = readID3v2(path, meta)
	case "m4b":
		err = readM4B(path, meta)
	}

	// if deep reading failed or left fields empty, try the filename
	if meta.Title == "" || meta.Author == "" {
		parseFilename(filepath.Base(path), meta)
	}

	return meta, err
}

// FormatFromExt maps a file extension to a human-readable format label.
func FormatFromExt(ext string) string {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "epub":
		return "epub"
	case "pdf":
		return "pdf"
	case "mobi", "azw", "azw3":
		return "mobi"
	case "mp3":
		return "mp3"
	case "m4b", "m4a":
		return "m4b"
	case "cbz":
		return "cbz"
	case "cbr":
		return "cbr"
	default:
		return strings.ToLower(strings.TrimPrefix(ext, "."))
	}
}

// --- EPUB ---

// epub/META-INF/container.xml
type epubContainer struct {
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

// OPF package metadata (Dublin Core subset we care about)
type opfMetadata struct {
	Titles      []string   `xml:"metadata>title"`
	Creators    []string   `xml:"metadata>creator"`
	Dates       []string   `xml:"metadata>date"`
	Publishers  []string   `xml:"metadata>publisher"`
	Identifiers []struct {
		Scheme string `xml:"scheme,attr"`
		Value  string `xml:",chardata"`
	} `xml:"metadata>identifier"`
}

func readEPUB(path string, meta *BookMetadata) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()

	// find the OPF path from container.xml
	opfPath, err := epubOPFPath(r)
	if err != nil {
		return err
	}

	// read + parse the OPF
	opfFile := zipFind(r, opfPath)
	if opfFile == nil {
		return nil
	}
	rc, err := opfFile.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return err
	}

	var pkg opfMetadata
	if err := xml.Unmarshal(data, &pkg); err != nil {
		return err
	}

	if len(pkg.Titles) > 0 {
		meta.Title = strings.TrimSpace(pkg.Titles[0])
	}
	if len(pkg.Creators) > 0 {
		meta.Author = strings.TrimSpace(pkg.Creators[0])
	}
	if len(pkg.Dates) > 0 {
		meta.Year = extractYear(pkg.Dates[0])
	}
	if len(pkg.Publishers) > 0 {
		meta.Publisher = strings.TrimSpace(pkg.Publishers[0])
	}
	for _, id := range pkg.Identifiers {
		if strings.EqualFold(id.Scheme, "isbn") || strings.HasPrefix(id.Value, "978") || strings.HasPrefix(id.Value, "979") {
			meta.ISBN = strings.TrimSpace(id.Value)
			break
		}
	}

	return nil
}

func epubOPFPath(r *zip.ReadCloser) (string, error) {
	f := zipFind(r, "META-INF/container.xml")
	if f == nil {
		return "", nil
	}
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	var c epubContainer
	if err := xml.NewDecoder(rc).Decode(&c); err != nil {
		return "", err
	}
	if len(c.Rootfiles) > 0 {
		return c.Rootfiles[0].FullPath, nil
	}
	return "", nil
}

func zipFind(r *zip.ReadCloser, name string) *zip.File {
	for _, f := range r.File {
		if f.Name == name {
			return f
		}
	}
	// some EPUBs don't normalise case
	lower := strings.ToLower(name)
	for _, f := range r.File {
		if strings.ToLower(f.Name) == lower {
			return f
		}
	}
	return nil
}

// --- ID3v2 (MP3) ---

func readID3v2(path string, meta *BookMetadata) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	header := make([]byte, 10)
	if _, err := io.ReadFull(f, header); err != nil {
		return err
	}
	if string(header[0:3]) != "ID3" {
		return nil // no ID3v2 tag
	}

	majorVer := header[3]
	// total tag size is syncsafe encoded
	tagSize := id3SyncsafeInt(header[6:10])

	tagData := make([]byte, tagSize)
	if _, err := io.ReadFull(f, tagData); err != nil {
		return err
	}

	// walk frames
	pos := 0
	for pos+10 < len(tagData) {
		frameID := string(tagData[pos : pos+4])
		if frameID == "\x00\x00\x00\x00" {
			break
		}

		var frameSize int
		if majorVer >= 4 {
			frameSize = int(id3SyncsafeInt(tagData[pos+4 : pos+8]))
		} else {
			frameSize = int(binary.BigEndian.Uint32(tagData[pos+4 : pos+8]))
		}
		pos += 10

		if frameSize <= 0 || pos+frameSize > len(tagData) {
			break
		}

		frameBody := tagData[pos : pos+frameSize]
		pos += frameSize

		// text frames: first byte is encoding
		if len(frameBody) < 2 {
			continue
		}
		text := id3DecodeText(frameBody[0], frameBody[1:])

		switch frameID {
		case "TIT2":
			meta.Title = text
		case "TPE1", "TPE2":
			if meta.Author == "" {
				meta.Author = text
			}
		case "TYER", "TDRC":
			if meta.Year == "" {
				meta.Year = extractYear(text)
			}
		case "TPUB":
			meta.Publisher = text
		}
	}

	return nil
}

func id3SyncsafeInt(b []byte) int {
	return int(b[0])<<21 | int(b[1])<<14 | int(b[2])<<7 | int(b[3])
}

func id3DecodeText(enc byte, data []byte) string {
	switch enc {
	case 0x01: // UTF-16 with BOM
		if len(data) < 2 {
			return ""
		}
		bigEndian := false
		if data[0] == 0xFF && data[1] == 0xFE {
			data = data[2:] // LE BOM
		} else if data[0] == 0xFE && data[1] == 0xFF {
			data = data[2:] // BE BOM
			bigEndian = true
		}
		// strip UTF-16 null terminator (two zero bytes) before decoding
		for len(data) >= 2 && data[len(data)-2] == 0 && data[len(data)-1] == 0 {
			data = data[:len(data)-2]
		}
		return decodeUTF16(data, bigEndian)
	case 0x02: // UTF-16BE without BOM
		for len(data) >= 2 && data[len(data)-2] == 0 && data[len(data)-1] == 0 {
			data = data[:len(data)-2]
		}
		return decodeUTF16(data, true)
	default: // 0x00 ISO-8859-1, 0x03 UTF-8
		for len(data) > 0 && data[len(data)-1] == 0 {
			data = data[:len(data)-1]
		}
		return strings.TrimSpace(string(data))
	}
}

func decodeUTF16(data []byte, bigEndian bool) string {
	if len(data) == 0 {
		return ""
	}
	runes := make([]rune, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		var r uint16
		if bigEndian {
			r = binary.BigEndian.Uint16(data[i : i+2])
		} else {
			r = binary.LittleEndian.Uint16(data[i : i+2])
		}
		runes = append(runes, rune(r))
	}
	return strings.TrimSpace(string(runes))
}

// --- M4B (MP4 atoms) ---
// Parses just enough of the iTunes metadata atoms to get title/artist/year.

func readM4B(path string, meta *BookMetadata) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	// look for ©nam (title), ©ART (artist), ©day (year), aART (album artist)
	tags := map[string]*string{
		"\xa9nam": &meta.Title,
		"\xa9ART": &meta.Author,
		"aART":    &meta.Author, // album artist, fallback
		"\xa9day": &meta.Year,
	}

	for tag, dest := range tags {
		if *dest != "" {
			continue
		}
		idx := strings.Index(string(data), tag)
		if idx < 0 {
			continue
		}
		// atom layout: [parent_size:4][tag:4] → rest starts here
		//   [data_atom_size:4]["data":4][flags:4][locale:4][value...]
		rest := data[idx+4:]
		dataIdx := strings.Index(string(rest[:min(256, len(rest))]), "data")
		if dataIdx < 4 {
			// need at least 4 bytes before "data" for the size field
			continue
		}
		// the 4 bytes BEFORE "data" are the data atom's size
		atomSize := int(binary.BigEndian.Uint32(rest[dataIdx-4 : dataIdx]))
		atomStart := dataIdx - 4
		valStart := dataIdx + 4 + 4 + 4 // skip "data" + flags + locale
		valEnd := atomStart + atomSize
		if valStart >= len(rest) || valEnd > len(rest) || valEnd <= valStart {
			continue
		}
		*dest = strings.TrimSpace(string(rest[valStart:valEnd]))
	}

	if meta.Year != "" {
		meta.Year = extractYear(meta.Year)
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// --- filename fallback ---

// parseFilename tries "Author - Title" or just uses the name as the title.
func parseFilename(name string, meta *BookMetadata) {
	// strip extension
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = strings.TrimSpace(name)

	if idx := strings.Index(name, " - "); idx > 0 {
		left := strings.TrimSpace(name[:idx])
		right := strings.TrimSpace(name[idx+3:])
		if meta.Author == "" {
			meta.Author = left
		}
		if meta.Title == "" {
			meta.Title = right
		}
	} else if meta.Title == "" {
		meta.Title = name
	}
}

// extractYear pulls the first 4-digit year out of a date string.
func extractYear(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 4 && isDigits(s[:4]) {
		return s[:4]
	}
	return s
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
