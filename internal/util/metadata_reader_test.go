package utils

import (
	"archive/zip"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- FormatFromExt ---

func TestFormatFromExt_Epub(t *testing.T) {
	assert.Equal(t, "epub", FormatFromExt(".epub"))
}

func TestFormatFromExt_MobiVariants(t *testing.T) {
	for _, ext := range []string{".mobi", ".azw", ".azw3"} {
		assert.Equal(t, "mobi", FormatFromExt(ext), "ext=%s", ext)
	}
}

func TestFormatFromExt_AudioVariants(t *testing.T) {
	assert.Equal(t, "mp3", FormatFromExt(".mp3"))
	assert.Equal(t, "m4b", FormatFromExt(".m4b"))
	assert.Equal(t, "m4b", FormatFromExt(".m4a"))
}

func TestFormatFromExt_Comics(t *testing.T) {
	assert.Equal(t, "cbz", FormatFromExt(".cbz"))
	assert.Equal(t, "cbr", FormatFromExt(".cbr"))
}

func TestFormatFromExt_NoDot(t *testing.T) {
	assert.Equal(t, "epub", FormatFromExt("epub"))
}

func TestFormatFromExt_Unknown(t *testing.T) {
	assert.Equal(t, "xyz", FormatFromExt(".xyz"))
}

// --- filename fallback ---

func TestReadMetadata_FilenameAuthorDashTitle(t *testing.T) {
	// empty file with no parseable content → falls through to filename
	dir := t.TempDir()
	path := filepath.Join(dir, "Frank Herbert - Dune.pdf")
	require.NoError(t, os.WriteFile(path, []byte{}, 0o644))

	meta, _ := ReadMetadata(path)
	assert.Equal(t, "Frank Herbert", meta.Author)
	assert.Equal(t, "Dune", meta.Title)
	assert.Equal(t, "pdf", meta.Format)
}

func TestReadMetadata_FilenameTitleOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Dune.pdf")
	require.NoError(t, os.WriteFile(path, []byte{}, 0o644))

	meta, _ := ReadMetadata(path)
	assert.Equal(t, "Dune", meta.Title)
	assert.Equal(t, "", meta.Author)
}

func TestReadMetadata_NonExistentFile(t *testing.T) {
	_, err := ReadMetadata("/tmp/does-not-exist.epub")
	assert.Error(t, err)
}

// --- EPUB ---

func TestReadMetadata_EPUB_FullMetadata(t *testing.T) {
	path := makeTestEPUB(t, "Dune", "Frank Herbert", "1965", "Chilton", "9780441013593")
	meta, err := ReadMetadata(path)
	require.NoError(t, err)
	assert.Equal(t, "Dune", meta.Title)
	assert.Equal(t, "Frank Herbert", meta.Author)
	assert.Equal(t, "1965", meta.Year)
	assert.Equal(t, "Chilton", meta.Publisher)
	assert.Equal(t, "9780441013593", meta.ISBN)
	assert.Equal(t, "epub", meta.Format)
}

func TestReadMetadata_EPUB_YearFromFullDate(t *testing.T) {
	path := makeTestEPUB(t, "T", "A", "2005-06-15", "", "")
	meta, err := ReadMetadata(path)
	require.NoError(t, err)
	assert.Equal(t, "2005", meta.Year)
}

func TestReadMetadata_EPUB_UppercaseFilenames_CaseInsensitiveFallback(t *testing.T) {
	// some EPUBs store META-INF/CONTAINER.XML or OEBPS/CONTENT.OPF in uppercase
	path := makeTestEPUBUppercase(t, "Dune", "Frank Herbert", "1965")
	meta, err := ReadMetadata(path)
	require.NoError(t, err)
	assert.Equal(t, "Dune", meta.Title)
	assert.Equal(t, "Frank Herbert", meta.Author)
}

func TestReadMetadata_EPUB_EmptyFields_FallsBackToFilename(t *testing.T) {
	// EPUB with no author — should fall back to filename parsing
	path := makeTestEPUBInDir(t, t.TempDir(), "Frank Herbert - Dune.epub", "Dune", "", "", "", "")
	meta, err := ReadMetadata(path)
	require.NoError(t, err)
	assert.Equal(t, "Dune", meta.Title)
	assert.Equal(t, "Frank Herbert", meta.Author) // from filename fallback
}

// --- ID3v2 (MP3) ---

func TestReadMetadata_MP3_TitleAndAuthor(t *testing.T) {
	path := makeTestMP3(t, "Dune", "Frank Herbert", "1965")
	meta, err := ReadMetadata(path)
	require.NoError(t, err)
	assert.Equal(t, "Dune", meta.Title)
	assert.Equal(t, "Frank Herbert", meta.Author)
	assert.Equal(t, "1965", meta.Year)
	assert.Equal(t, "mp3", meta.Format)
}

func TestReadMetadata_MP3_UTF16LE_BOM(t *testing.T) {
	// encoding 0x01, BOM 0xFF 0xFE → little-endian
	path := makeTestMP3WithEncoding(t, "Дюна", "Фрэнк Херберт", 0x01, false)
	meta, err := ReadMetadata(path)
	require.NoError(t, err)
	assert.Equal(t, "Дюна", meta.Title)
	assert.Equal(t, "Фрэнк Херберт", meta.Author)
}

func TestReadMetadata_MP3_UTF16BE_BOM(t *testing.T) {
	// encoding 0x01, BOM 0xFE 0xFF → big-endian
	path := makeTestMP3WithEncoding(t, "Dune", "Frank Herbert", 0x01, true)
	meta, err := ReadMetadata(path)
	require.NoError(t, err)
	assert.Equal(t, "Dune", meta.Title)
	assert.Equal(t, "Frank Herbert", meta.Author)
}

func TestReadMetadata_MP3_UTF16BE_NoBOM(t *testing.T) {
	// encoding 0x02 = UTF-16BE without BOM, always big-endian
	path := makeTestMP3WithEncoding(t, "Foundation", "Isaac Asimov", 0x02, true)
	meta, err := ReadMetadata(path)
	require.NoError(t, err)
	assert.Equal(t, "Foundation", meta.Title)
	assert.Equal(t, "Isaac Asimov", meta.Author)
}

func TestReadMetadata_MP3_NoID3Tag_FallsBackToFilename(t *testing.T) {
	dir := t.TempDir()
	// plain file with no ID3 header
	path := filepath.Join(dir, "Frank Herbert - Dune.mp3")
	require.NoError(t, os.WriteFile(path, []byte("not an mp3"), 0o644))

	meta, _ := ReadMetadata(path)
	assert.Equal(t, "Dune", meta.Title)
	assert.Equal(t, "Frank Herbert", meta.Author)
}

// --- helpers ---

func makeTestEPUB(t *testing.T, title, author, year, publisher, isbn string) string {
	t.Helper()
	return makeTestEPUBInDir(t, t.TempDir(), "test.epub", title, author, year, publisher, isbn)
}

func makeTestEPUBInDir(t *testing.T, dir, filename, title, author, year, publisher, isbn string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	f, err := os.Create(path)
	require.NoError(t, err)

	w := zip.NewWriter(f)

	// mimetype must be first and uncompressed per EPUB spec
	mh := &zip.FileHeader{Name: "mimetype", Method: zip.Store}
	mime, err := w.CreateHeader(mh)
	require.NoError(t, err)
	_, err = mime.Write([]byte("application/epub+zip"))
	require.NoError(t, err)

	container, err := w.Create("META-INF/container.xml")
	require.NoError(t, err)
	_, err = fmt.Fprint(container, `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)
	require.NoError(t, err)

	opf, err := w.Create("OEBPS/content.opf")
	require.NoError(t, err)
	_, err = fmt.Fprintf(opf, `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
    <dc:title>%s</dc:title>
    <dc:creator>%s</dc:creator>
    <dc:date>%s</dc:date>
    <dc:publisher>%s</dc:publisher>
    <dc:identifier opf:scheme="isbn">%s</dc:identifier>
  </metadata>
</package>`, title, author, year, publisher, isbn)
	require.NoError(t, err)

	require.NoError(t, w.Close())
	require.NoError(t, f.Close())
	return path
}

// --- M4B ---

func TestReadMetadata_M4B_TitleAuthorYear(t *testing.T) {
	path := makeTestM4B(t, "Dune", "Frank Herbert", "1965")
	meta, err := ReadMetadata(path)
	require.NoError(t, err)
	assert.Equal(t, "Dune", meta.Title)
	assert.Equal(t, "Frank Herbert", meta.Author)
	assert.Equal(t, "1965", meta.Year)
	assert.Equal(t, "m4b", meta.Format)
}

func TestReadMetadata_M4B_YearFromFullDate(t *testing.T) {
	path := makeTestM4B(t, "Foundation", "Isaac Asimov", "1951-05-01")
	meta, err := ReadMetadata(path)
	require.NoError(t, err)
	assert.Equal(t, "1951", meta.Year)
}

func TestReadMetadata_M4B_FallsBackToFilenameWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Frank Herbert - Dune.m4b")
	// empty file — no atoms → falls back to filename
	require.NoError(t, os.WriteFile(path, []byte{}, 0o644))
	meta, _ := ReadMetadata(path)
	assert.Equal(t, "Frank Herbert", meta.Author)
	assert.Equal(t, "Dune", meta.Title)
}

// makeTestM4B creates a minimal M4B file with iTunes metadata atoms.
func makeTestM4B(t *testing.T, title, author, year string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.m4b")

	var buf []byte
	buf = append(buf, m4bAtom("\xa9nam", title)...)
	buf = append(buf, m4bAtom("\xa9ART", author)...)
	buf = append(buf, m4bAtom("\xa9day", year)...)

	require.NoError(t, os.WriteFile(path, buf, 0o644))
	return path
}

// m4bAtom builds a minimal iTunes metadata atom:
// [parent_size:4][tag:4][data_atom_size:4]["data":4][flags:4][locale:4][value]
func m4bAtom(tag, value string) []byte {
	val := []byte(value)
	dataAtomSize := 4 + 4 + 4 + 4 + len(val) // size + "data" + flags + locale + value
	total := 4 + 4 + dataAtomSize             // parent size + tag + data atom

	buf := make([]byte, total)
	binary.BigEndian.PutUint32(buf[0:4], uint32(total))
	copy(buf[4:8], tag)

	offset := 8
	binary.BigEndian.PutUint32(buf[offset:offset+4], uint32(dataAtomSize))
	copy(buf[offset+4:offset+8], "data")
	binary.BigEndian.PutUint32(buf[offset+8:offset+12], 1)  // flags = 1 (UTF-8 text)
	binary.BigEndian.PutUint32(buf[offset+12:offset+16], 0) // locale
	copy(buf[offset+16:], val)

	return buf
}

// makeTestMP3 creates a minimal ID3v2.3 tagged MP3 file.
func makeTestMP3(t *testing.T, title, author, year string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mp3")

	frames := id3Frame("TIT2", title)
	frames = append(frames, id3Frame("TPE1", author)...)
	if year != "" {
		frames = append(frames, id3Frame("TYER", year)...)
	}

	// ID3v2.3 header: "ID3" + version(2.3) + revision(0) + flags(0) + syncsafe size
	header := make([]byte, 10)
	copy(header[0:3], "ID3")
	header[3] = 3
	header[4] = 0
	header[5] = 0
	sz := len(frames)
	header[6] = byte((sz >> 21) & 0x7f)
	header[7] = byte((sz >> 14) & 0x7f)
	header[8] = byte((sz >> 7) & 0x7f)
	header[9] = byte(sz & 0x7f)

	data := append(header, frames...)
	require.NoError(t, os.WriteFile(path, data, 0o644))
	return path
}

// makeTestEPUBUppercase creates an EPUB where container.xml and content.opf use uppercase names.
func makeTestEPUBUppercase(t *testing.T, title, author, year string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.epub")
	f, err := os.Create(path)
	require.NoError(t, err)

	w := zip.NewWriter(f)
	mh := &zip.FileHeader{Name: "mimetype", Method: zip.Store}
	mime, err := w.CreateHeader(mh)
	require.NoError(t, err)
	_, _ = mime.Write([]byte("application/epub+zip"))

	container, err := w.Create("META-INF/CONTAINER.XML") // uppercase
	require.NoError(t, err)
	_, _ = fmt.Fprint(container, `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/CONTENT.OPF" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)

	opf, err := w.Create("OEBPS/CONTENT.OPF") // uppercase
	require.NoError(t, err)
	_, _ = fmt.Fprintf(opf, `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>%s</dc:title>
    <dc:creator>%s</dc:creator>
    <dc:date>%s</dc:date>
  </metadata>
</package>`, title, author, year)

	require.NoError(t, w.Close())
	require.NoError(t, f.Close())
	return path
}

// makeTestMP3WithEncoding creates an MP3 with ID3v2.3 frames using a specific text encoding.
// enc=0x01 → UTF-16 with BOM, enc=0x02 → UTF-16BE without BOM.
func makeTestMP3WithEncoding(t *testing.T, title, author string, enc byte, bigEndian bool) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mp3")

	frames := id3FrameEncoded("TIT2", enc, title, bigEndian)
	frames = append(frames, id3FrameEncoded("TPE1", enc, author, bigEndian)...)

	header := make([]byte, 10)
	copy(header[0:3], "ID3")
	header[3] = 3
	sz := len(frames)
	header[6] = byte((sz >> 21) & 0x7f)
	header[7] = byte((sz >> 14) & 0x7f)
	header[8] = byte((sz >> 7) & 0x7f)
	header[9] = byte(sz & 0x7f)

	require.NoError(t, os.WriteFile(path, append(header, frames...), 0o644))
	return path
}

// id3FrameEncoded builds an ID3v2.3 text frame with the given encoding.
func id3FrameEncoded(id string, enc byte, text string, bigEndian bool) []byte {
	var body []byte
	body = append(body, enc)

	if enc == 0x01 {
		// UTF-16 with BOM
		if bigEndian {
			body = append(body, 0xFE, 0xFF) // BE BOM
		} else {
			body = append(body, 0xFF, 0xFE) // LE BOM
		}
	}

	for _, r := range text {
		if bigEndian || enc == 0x02 {
			body = append(body, byte(r>>8), byte(r&0xff))
		} else {
			body = append(body, byte(r&0xff), byte(r>>8))
		}
	}

	hdr := make([]byte, 10)
	copy(hdr[0:4], id)
	binary.BigEndian.PutUint32(hdr[4:8], uint32(len(body)))
	return append(hdr, body...)
}

// id3Frame builds a raw ID3v2.3 text frame.
func id3Frame(id, text string) []byte {
	body := append([]byte{0x00}, []byte(text)...) // encoding=latin1 + text
	hdr := make([]byte, 10)
	copy(hdr[0:4], id)
	binary.BigEndian.PutUint32(hdr[4:8], uint32(len(body)))
	// flags: 0x00 0x00 (already zeroed)
	return append(hdr, body...)
}
