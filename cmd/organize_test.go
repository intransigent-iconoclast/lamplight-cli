package cmd

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

// --- resolveConflict ---

func TestResolveConflict_PathDoesNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.epub")
	assert.Equal(t, path, resolveConflict(path))
}

func TestResolveConflict_PathExists_Appends2(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.epub")
	require.NoError(t, os.WriteFile(path, []byte{}, 0o644))

	result := resolveConflict(path)
	assert.Equal(t, filepath.Join(dir, "book_2.epub"), result)
}

func TestResolveConflict_PathAnd2Exist_Appends3(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.epub")
	path2 := filepath.Join(dir, "book_2.epub")
	require.NoError(t, os.WriteFile(path, []byte{}, 0o644))
	require.NoError(t, os.WriteFile(path2, []byte{}, 0o644))

	result := resolveConflict(path)
	assert.Equal(t, filepath.Join(dir, "book_3.epub"), result)
}

// --- resolveConflictDir ---

func TestResolveConflictDir_DoesNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AudioBook")
	assert.Equal(t, path, resolveConflictDir(path))
}

func TestResolveConflictDir_Exists_Appends2(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "AudioBook")
	require.NoError(t, os.MkdirAll(existing, 0o755))

	result := resolveConflictDir(existing)
	assert.Equal(t, filepath.Join(dir, "AudioBook_2"), result)
}

// --- organizeEntry: routing ---

func TestOrganizeEntry_FileRouted_ToOrganizeFile(t *testing.T) {
	libraryRoot := t.TempDir()
	// filename with no " - " separator → no author parsed → incomplete → uncategorized
	src := filepath.Join(t.TempDir(), "mystery_book_no_author.pdf")
	require.NoError(t, os.WriteFile(src, []byte{}, 0o644))

	_, placed, err := organizeEntry(src, libraryRoot, "", "{author}/{title}", false)
	require.NoError(t, err)
	assert.Equal(t, "uncategorized", placed)
}

func TestOrganizeEntry_NonExistentPath_ReturnsError(t *testing.T) {
	_, _, err := organizeEntry("/tmp/does-not-exist.epub", t.TempDir(), "", "{author}/{title}", false)
	assert.Error(t, err)
}

// --- organizeFile: library vs uncategorized ---

func TestOrganizeFile_IncompleteMetadata_GoesToUncategorized(t *testing.T) {
	libraryRoot := t.TempDir()
	src := filepath.Join(t.TempDir(), "mystery.epub")
	require.NoError(t, os.WriteFile(src, []byte("not a real epub"), 0o644))

	_, placed, err := organizeFile(src, libraryRoot, "", "{author}/{title}", false)
	require.NoError(t, err)
	assert.Equal(t, "uncategorized", placed)
	_, statErr := os.Stat(filepath.Join(libraryRoot, "uncategorized", "mystery.epub"))
	assert.NoError(t, statErr)
}

func TestOrganizeFile_CompleteMetadata_GoesToLibrary(t *testing.T) {
	libraryRoot := t.TempDir()
	src := makeOrganizeEPUB(t, "Dune", "Frank Herbert", "1965")

	dest, placed, err := organizeFile(src, libraryRoot, "", "{author}/{title} ({year})", false)
	require.NoError(t, err)
	assert.Equal(t, "library", placed)
	assert.Equal(t, filepath.Join(libraryRoot, "Frank Herbert", "Dune (1965).epub"), dest)
	_, statErr := os.Stat(dest)
	assert.NoError(t, statErr)
}

func TestOrganizeFile_DryRun_DoesNotMoveFile(t *testing.T) {
	libraryRoot := t.TempDir()
	src := makeOrganizeEPUB(t, "Dune", "Frank Herbert", "1965")

	_, _, err := organizeFile(src, libraryRoot, "", "{author}/{title} ({year})", true)
	require.NoError(t, err)
	// source must still exist
	_, statErr := os.Stat(src)
	assert.NoError(t, statErr)
	// nothing in library
	_, statErr = os.Stat(filepath.Join(libraryRoot, "Frank Herbert"))
	assert.True(t, os.IsNotExist(statErr))
}

func TestOrganizeFile_ConflictResolved(t *testing.T) {
	libraryRoot := t.TempDir()
	tmpl := "{author}/{title} ({year})"

	// organize first copy
	src1 := makeOrganizeEPUB(t, "Dune", "Frank Herbert", "1965")
	_, _, err := organizeFile(src1, libraryRoot, "", tmpl, false)
	require.NoError(t, err)

	// organize second copy — should get _2 suffix
	src2 := makeOrganizeEPUB(t, "Dune", "Frank Herbert", "1965")
	dest2, _, err := organizeFile(src2, libraryRoot, "", tmpl, false)
	require.NoError(t, err)

	// dest must reflect the actual _2 file, not the original name
	assert.Equal(t, filepath.Join(libraryRoot, "Frank Herbert", "Dune (1965)_2.epub"), dest2)
	_, statErr := os.Stat(dest2)
	assert.NoError(t, statErr)
}

// --- organizeDir ---

func TestOrganizeDir_SingleFile_Unwrapped(t *testing.T) {
	libraryRoot := t.TempDir()
	srcDir := t.TempDir()
	// one epub in the folder → should be treated as a flat file
	epubPath := makeOrganizeEPUBInDir(t, srcDir, "test.epub", "Dune", "Frank Herbert", "1965")
	_ = epubPath

	_, placed, err := organizeDir(srcDir, libraryRoot, "", "{author}/{title} ({year})", false)
	require.NoError(t, err)
	assert.Equal(t, "library", placed)
	// file exists flat, not in a subfolder
	_, statErr := os.Stat(filepath.Join(libraryRoot, "Frank Herbert", "Dune (1965).epub"))
	assert.NoError(t, statErr)
}

func TestOrganizeDir_NoBookFiles_ReturnsError(t *testing.T) {
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "notes.txt"), []byte{}, 0o644))

	_, _, err := organizeDir(srcDir, t.TempDir(), "", "{author}/{title}", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no book files")
}

func TestOrganizeDir_MultipleAudioFiles_StaysTogether(t *testing.T) {
	libraryRoot := t.TempDir()
	srcDir := t.TempDir()

	// two mp3 chapter files with matching metadata
	makeOrganizeMP3InDir(t, srcDir, "01.mp3", "Dune", "Frank Herbert", "1965")
	makeOrganizeMP3InDir(t, srcDir, "02.mp3", "Dune", "Frank Herbert", "1965")

	dest, placed, err := organizeDir(srcDir, libraryRoot, "", "{author}/{title} ({year})", false)
	require.NoError(t, err)
	assert.Equal(t, "library", placed)
	assert.Equal(t, filepath.Join(libraryRoot, "Frank Herbert", "Dune (1965)"), dest)

	// both files should be inside the destination folder
	_, err1 := os.Stat(filepath.Join(dest, "01.mp3"))
	_, err2 := os.Stat(filepath.Join(dest, "02.mp3"))
	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func TestOrganizeDir_MultipleFiles_IncompleteMetadata_GoesToUncategorized(t *testing.T) {
	libraryRoot := t.TempDir()
	srcDir := t.TempDir()

	// empty files — no parseable metadata, no author/title in filename
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "chapter01.mp3"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "chapter02.mp3"), []byte(""), 0o644))

	_, placed, err := organizeDir(srcDir, libraryRoot, "", "{author}/{title}", false)
	require.NoError(t, err)
	assert.Equal(t, "uncategorized", placed)
}

func TestOrganizeDir_SourceRemovedAfterMove(t *testing.T) {
	libraryRoot := t.TempDir()
	srcDir := t.TempDir()

	makeOrganizeMP3InDir(t, srcDir, "01.mp3", "Dune", "Frank Herbert", "1965")
	makeOrganizeMP3InDir(t, srcDir, "02.mp3", "Dune", "Frank Herbert", "1965")

	_, _, err := organizeDir(srcDir, libraryRoot, "", "{author}/{title} ({year})", false)
	require.NoError(t, err)

	// source directory should be gone
	_, statErr := os.Stat(srcDir)
	assert.True(t, os.IsNotExist(statErr), "source directory should be removed after organize")
}

func TestOrganizeDir_SourceRemovedEvenWithSubdirs(t *testing.T) {
	// torrent clients sometimes create nested subdirs — RemoveAll should handle it
	libraryRoot := t.TempDir()
	srcDir := t.TempDir()

	subDir := filepath.Join(srcDir, "Subs")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	// book files are at the top level
	makeOrganizeMP3InDir(t, srcDir, "01.mp3", "Dune", "Frank Herbert", "1965")
	makeOrganizeMP3InDir(t, srcDir, "02.mp3", "Dune", "Frank Herbert", "1965")

	_, _, err := organizeDir(srcDir, libraryRoot, "", "{author}/{title} ({year})", false)
	require.NoError(t, err)

	_, statErr := os.Stat(srcDir)
	assert.True(t, os.IsNotExist(statErr), "source directory with subdirs should be removed")
}

func TestOrganizeDir_DryRun_DoesNotMove(t *testing.T) {
	libraryRoot := t.TempDir()
	srcDir := t.TempDir()

	makeOrganizeMP3InDir(t, srcDir, "01.mp3", "Dune", "Frank Herbert", "1965")
	makeOrganizeMP3InDir(t, srcDir, "02.mp3", "Dune", "Frank Herbert", "1965")

	_, _, err := organizeDir(srcDir, libraryRoot, "", "{author}/{title} ({year})", true)
	require.NoError(t, err)

	// source files still exist
	_, err1 := os.Stat(filepath.Join(srcDir, "01.mp3"))
	_, err2 := os.Stat(filepath.Join(srcDir, "02.mp3"))
	assert.NoError(t, err1)
	assert.NoError(t, err2)

	// nothing in library
	_, statErr := os.Stat(filepath.Join(libraryRoot, "Frank Herbert"))
	assert.True(t, os.IsNotExist(statErr))
}

// --- audiobook path routing ---

func TestOrganizeFile_AudiobookPath_RoutesMP3(t *testing.T) {
	libraryRoot := t.TempDir()
	audiobookRoot := t.TempDir()
	src := makeOrganizeMP3InDir(t, t.TempDir(), "book.mp3", "Dune", "Frank Herbert", "1965")

	dest, placed, err := organizeFile(src, libraryRoot, audiobookRoot, "{author}/{title} ({year})", false)
	require.NoError(t, err)
	assert.Equal(t, "library", placed)
	assert.Equal(t, filepath.Join(audiobookRoot, "Frank Herbert", "Dune (1965).mp3"), dest)
	// should be in the audiobook root, not the library root
	_, statErr := os.Stat(dest)
	assert.NoError(t, statErr)
	// should NOT be in the library root
	_, statErr = os.Stat(filepath.Join(libraryRoot, "Frank Herbert", "Dune (1965).mp3"))
	assert.True(t, os.IsNotExist(statErr))
}

func TestOrganizeFile_AudiobookPath_EPUBStaysInLibrary(t *testing.T) {
	libraryRoot := t.TempDir()
	audiobookRoot := t.TempDir()
	src := makeOrganizeEPUB(t, "Dune", "Frank Herbert", "1965")

	_, placed, err := organizeFile(src, libraryRoot, audiobookRoot, "{author}/{title} ({year})", false)
	require.NoError(t, err)
	assert.Equal(t, "library", placed)
	// should be in library root, not audiobook root
	_, statErr := os.Stat(filepath.Join(libraryRoot, "Frank Herbert", "Dune (1965).epub"))
	assert.NoError(t, statErr)
}

func TestOrganizeDir_AudiobookPath_RoutesAudioDir(t *testing.T) {
	libraryRoot := t.TempDir()
	audiobookRoot := t.TempDir()
	srcDir := t.TempDir()

	makeOrganizeMP3InDir(t, srcDir, "01.mp3", "Dune", "Frank Herbert", "1965")
	makeOrganizeMP3InDir(t, srcDir, "02.mp3", "Dune", "Frank Herbert", "1965")

	_, placed, err := organizeDir(srcDir, libraryRoot, audiobookRoot, "{author}/{title} ({year})", false)
	require.NoError(t, err)
	assert.Equal(t, "library", placed)
	// should be in audiobook root
	_, err1 := os.Stat(filepath.Join(audiobookRoot, "Frank Herbert", "Dune (1965)", "01.mp3"))
	_, err2 := os.Stat(filepath.Join(audiobookRoot, "Frank Herbert", "Dune (1965)", "02.mp3"))
	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func TestOrganizeFile_NoAudiobookPath_AudioGoesToLibrary(t *testing.T) {
	libraryRoot := t.TempDir()
	src := makeOrganizeMP3InDir(t, t.TempDir(), "book.mp3", "Dune", "Frank Herbert", "1965")

	// empty audiobook path — should fall back to library root
	_, placed, err := organizeFile(src, libraryRoot, "", "{author}/{title} ({year})", false)
	require.NoError(t, err)
	assert.Equal(t, "library", placed)
	_, statErr := os.Stat(filepath.Join(libraryRoot, "Frank Herbert", "Dune (1965).mp3"))
	assert.NoError(t, statErr)
}

// --- helpers ---

func makeOrganizeEPUB(t *testing.T, title, author, year string) string {
	t.Helper()
	return makeOrganizeEPUBInDir(t, t.TempDir(), "test.epub", title, author, year)
}

func makeOrganizeEPUBInDir(t *testing.T, dir, filename, title, author, year string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	f, err := os.Create(path)
	require.NoError(t, err)

	w := zip.NewWriter(f)
	mh := &zip.FileHeader{Name: "mimetype", Method: zip.Store}
	mime, err := w.CreateHeader(mh)
	require.NoError(t, err)
	_, _ = mime.Write([]byte("application/epub+zip"))

	container, err := w.Create("META-INF/container.xml")
	require.NoError(t, err)
	_, _ = fmt.Fprint(container, `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)

	opf, err := w.Create("OEBPS/content.opf")
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

func makeOrganizeMP3InDir(t *testing.T, dir, filename, title, author, year string) string {
	t.Helper()
	path := filepath.Join(dir, filename)

	frames := organizeID3Frame("TIT2", title)
	frames = append(frames, organizeID3Frame("TPE1", author)...)
	frames = append(frames, organizeID3Frame("TYER", year)...)

	header := make([]byte, 10)
	copy(header[0:3], "ID3")
	header[3] = 3
	sz := len(frames)
	header[6] = byte((sz >> 21) & 0x7f)
	header[7] = byte((sz >> 14) & 0x7f)
	header[8] = byte((sz >> 7) & 0x7f)
	header[9] = byte(sz & 0x7f)

	data := append(header, frames...)
	require.NoError(t, os.WriteFile(path, data, 0o644))
	return path
}

func organizeID3Frame(id, text string) []byte {
	body := append([]byte{0x00}, []byte(text)...)
	hdr := make([]byte, 10)
	copy(hdr[0:4], id)
	binary.BigEndian.PutUint32(hdr[4:8], uint32(len(body)))
	return append(hdr, body...)
}
