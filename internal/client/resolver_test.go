package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Resolve ---

func TestResolve_DirectMagnet_NoHTTPCall(t *testing.T) {
	// magnet links should be returned immediately without touching the network
	link := "magnet:?xt=urn:btih:abc123&dn=Dune"
	result, err := Resolve(context.Background(), &http.Client{}, link)
	require.NoError(t, err)
	assert.Equal(t, link, result.Magnet)
	assert.Nil(t, result.FileBytes)
}

func TestResolve_MagnetCaseInsensitive(t *testing.T) {
	link := "MAGNET:?xt=urn:btih:abc123"
	result, err := Resolve(context.Background(), &http.Client{}, link)
	require.NoError(t, err)
	assert.Equal(t, link, result.Magnet)
}

func TestResolve_LinkWithLeadingWhitespace(t *testing.T) {
	link := "  magnet:?xt=urn:btih:abc123"
	result, err := Resolve(context.Background(), &http.Client{}, link)
	require.NoError(t, err)
	assert.Equal(t, "magnet:?xt=urn:btih:abc123", result.Magnet)
}

func TestResolve_URLRedirectsToMagnet(t *testing.T) {
	magnet := "magnet:?xt=urn:btih:deadbeef"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, magnet, http.StatusFound)
	}))
	defer srv.Close()

	result, err := Resolve(context.Background(), &http.Client{}, srv.URL+"/torrent")
	require.NoError(t, err)
	assert.Equal(t, magnet, result.Magnet)
	assert.Nil(t, result.FileBytes)
}

func TestResolve_URLReturnsTorrentFile(t *testing.T) {
	// bencoded torrent starts with 'd'
	torrentBytes := []byte("d8:announce27:http://tracker.example.come")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-bittorrent")
		_, _ = w.Write(torrentBytes)
	}))
	defer srv.Close()

	result, err := Resolve(context.Background(), &http.Client{}, srv.URL+"/file.torrent")
	require.NoError(t, err)
	assert.Equal(t, torrentBytes, result.FileBytes)
	assert.Empty(t, result.Magnet)
}

func TestResolve_HTMLResponse_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<!DOCTYPE html><html><body>login required</body></html>"))
	}))
	defer srv.Close()

	_, err := Resolve(context.Background(), &http.Client{}, srv.URL+"/torrent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "html")
}

func TestResolve_HTMLLowercase_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html><body>nope</body></html>"))
	}))
	defer srv.Close()

	_, err := Resolve(context.Background(), &http.Client{}, srv.URL)
	assert.Error(t, err)
}

func TestResolve_EmptyBody_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// write nothing
	}))
	defer srv.Close()

	_, err := Resolve(context.Background(), &http.Client{}, srv.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestResolve_NonTorrentBinary_ReturnsError(t *testing.T) {
	// valid non-empty body that isn't a torrent (doesn't start with 'd')
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("this is not a torrent"))
	}))
	defer srv.Close()

	_, err := Resolve(context.Background(), &http.Client{}, srv.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "torrent")
}

func TestResolve_404Response_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	_, err := Resolve(context.Background(), &http.Client{}, srv.URL+"/missing.torrent")
	assert.Error(t, err)
}

func TestResolve_ServerDown_ReturnsError(t *testing.T) {
	_, err := Resolve(context.Background(), &http.Client{}, "http://127.0.0.1:19998/torrent")
	assert.Error(t, err)
}
