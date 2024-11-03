package models

type ResizeResult struct {
	Result string `json:"result"`
	URL    string `json:"url,omitempty"`
	Cached bool   `json:"cached"`
}
