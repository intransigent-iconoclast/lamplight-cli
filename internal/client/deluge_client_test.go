package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDeluge starts a test server that routes JSON-RPC calls by method name.
// responses maps method name → value to put in "result".
func mockDeluge(t *testing.T, responses map[string]any) (*DelugeClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		method, _ := req["method"].(string)
		result, ok := responses[method]
		if !ok {
			http.Error(w, "unknown method", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"result": result, "error": nil, "id": req["id"]})
	}))

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err)

	dc := NewDelugeClient(nil, &entity.Downloader{
		Scheme:   "http",
		Host:     u.Hostname(),
		Port:     port,
		Password: "testpass",
	})
	return dc, srv
}

// --- Authenticate ---

func TestDelugeClient_Authenticate_Success(t *testing.T) {
	dc, srv := mockDeluge(t, map[string]any{
		"auth.login": true,
	})
	defer srv.Close()

	err := dc.Authenticate(context.Background())
	assert.NoError(t, err)
}

func TestDelugeClient_Authenticate_WrongPassword(t *testing.T) {
	dc, srv := mockDeluge(t, map[string]any{
		"auth.login": false, // Deluge returns false for bad password
	})
	defer srv.Close()

	err := dc.Authenticate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestDelugeClient_Authenticate_ServerDown(t *testing.T) {
	dc := NewDelugeClient(nil, &entity.Downloader{
		Scheme:   "http",
		Host:     "127.0.0.1",
		Port:     19999, // nothing listening here
		Password: "x",
	})
	err := dc.Authenticate(context.Background())
	assert.Error(t, err)
}

// --- Add (magnet) ---

func TestDelugeClient_Add_Magnet_ReturnsHash(t *testing.T) {
	dc, srv := mockDeluge(t, map[string]any{
		"auth.login":           true,
		"core.add_torrent_magnet": "abc123deadbeef",
	})
	defer srv.Close()

	hash, err := dc.Add(context.Background(), &ResolvedTorrent{
		Magnet: "magnet:?xt=urn:btih:abc123deadbeef",
	})
	require.NoError(t, err)
	assert.Equal(t, "abc123deadbeef", hash)
}

func TestDelugeClient_Add_File_ReturnsHash(t *testing.T) {
	dc, srv := mockDeluge(t, map[string]any{
		"auth.login":          true,
		"core.add_torrent_file": "def456cafebabe",
	})
	defer srv.Close()

	hash, err := dc.Add(context.Background(), &ResolvedTorrent{
		FileBytes: []byte("d8:announce27:http://tracker.example.come"),
	})
	require.NoError(t, err)
	assert.Equal(t, "def456cafebabe", hash)
}

func TestDelugeClient_Add_EmptyResolved_ReturnsError(t *testing.T) {
	dc, srv := mockDeluge(t, map[string]any{
		"auth.login": true,
	})
	defer srv.Close()

	_, err := dc.Add(context.Background(), &ResolvedTorrent{})
	assert.Error(t, err)
}

func TestDelugeClient_Add_AuthFails_ReturnsError(t *testing.T) {
	dc, srv := mockDeluge(t, map[string]any{
		"auth.login": false,
	})
	defer srv.Close()

	_, err := dc.Add(context.Background(), &ResolvedTorrent{Magnet: "magnet:?xt=urn:btih:abc"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authenticate")
}

// --- GetTorrentStatus ---

func TestDelugeClient_GetTorrentStatus_SingleFile(t *testing.T) {
	dc, srv := mockDeluge(t, map[string]any{
		"auth.login": true,
		"core.get_torrent_status": map[string]any{
			"state":     "Downloading",
			"progress":  42.5,
			"save_path": "/data/incomplete",
			"name":      "Dune",
			"files":     []map[string]any{{"path": "Dune/Dune.epub"}},
		},
	})
	defer srv.Close()

	status, err := dc.GetTorrentStatus(context.Background(), "abc123")
	require.NoError(t, err)
	assert.Equal(t, "Downloading", status.State)
	assert.InDelta(t, 42.5, status.Progress, 0.01)
	// single file → save_path + files[0].path
	assert.Equal(t, "/data/incomplete/Dune/Dune.epub", status.FilePath)
}

func TestDelugeClient_GetTorrentStatus_MultiFile_UsesFolder(t *testing.T) {
	dc, srv := mockDeluge(t, map[string]any{
		"auth.login": true,
		"core.get_torrent_status": map[string]any{
			"state":     "Downloading",
			"progress":  70.0,
			"save_path": "/data/incomplete",
			"name":      "Dune Audiobook",
			"files": []map[string]any{
				{"path": "Dune Audiobook/01.mp3"},
				{"path": "Dune Audiobook/02.mp3"},
			},
		},
	})
	defer srv.Close()

	status, err := dc.GetTorrentStatus(context.Background(), "abc123")
	require.NoError(t, err)
	// multi-file → save_path + name (the containing folder)
	assert.Equal(t, "/data/incomplete/Dune Audiobook", status.FilePath)
}

func TestDelugeClient_GetTorrentStatus_NoFiles_FallsBackToName(t *testing.T) {
	dc, srv := mockDeluge(t, map[string]any{
		"auth.login": true,
		"core.get_torrent_status": map[string]any{
			"state":     "Seeding",
			"progress":  100.0,
			"save_path": "/data/complete",
			"name":      "foundation.epub",
			"files":     []map[string]any{},
		},
	})
	defer srv.Close()

	status, err := dc.GetTorrentStatus(context.Background(), "abc123")
	require.NoError(t, err)
	assert.Equal(t, "/data/complete/foundation.epub", status.FilePath)
}

func TestDelugeClient_GetTorrentStatus_Seeding_FullProgress(t *testing.T) {
	dc, srv := mockDeluge(t, map[string]any{
		"auth.login": true,
		"core.get_torrent_status": map[string]any{
			"state":     "Seeding",
			"progress":  100.0,
			"save_path": "/data",
			"name":      "book.epub",
			"files":     []map[string]any{{"path": "book.epub"}},
		},
	})
	defer srv.Close()

	status, err := dc.GetTorrentStatus(context.Background(), "hash")
	require.NoError(t, err)
	assert.Equal(t, "Seeding", status.State)
	assert.InDelta(t, 100.0, status.Progress, 0.01)
}

func TestDelugeClient_GetTorrentStatus_AuthFails_ReturnsError(t *testing.T) {
	dc, srv := mockDeluge(t, map[string]any{
		"auth.login": false,
	})
	defer srv.Close()

	_, err := dc.GetTorrentStatus(context.Background(), "abc123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authenticate")
}
