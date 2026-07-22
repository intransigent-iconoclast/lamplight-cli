package dao

import "encoding/json"

type DelugeAuthResponseBody struct {
	Result bool            `json:"result"`
	Error  json.RawMessage `json:"error"`
	ID     int             `json:"id"`
}

// returned by core.add_torrent_magnet / core.add_torrent_file — result is the hash
type DelugeAddResponseBody struct {
	Result string          `json:"result"`
	Error  json.RawMessage `json:"error"`
	ID     int             `json:"id"`
}

// returned by core.get_torrent_status
type DelugeTorrentStatusResponse struct {
	Result DelugeTorrentStatus `json:"result"`
	Error  json.RawMessage     `json:"error"`
	ID     int                 `json:"id"`
}

type DelugeTorrentStatus struct {
	State    string            `json:"state"`
	Progress float64           `json:"progress"`
	SavePath string            `json:"save_path"`
	Name     string            `json:"name"`
	Files    []DelugeTorrentFile `json:"files"`
}

type DelugeTorrentFile struct {
	Path string `json:"path"`
}
