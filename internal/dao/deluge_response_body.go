package dao

import "encoding/json"

type DelugeAuthResponseBody struct {
	Result bool            `json:"result"`
	Error  json.RawMessage `json:"error"`
	ID     int             `json:"id"`
}
