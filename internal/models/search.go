package models

type SearchResult struct {
	Photo    Photo   `json:"photo"`
	Score    float64 `json:"score"`
	Distance float64 `json:"distance"`
}

type SearchRequest struct {
	Query string `json:"query" form:"q"`
	Limit int    `json:"limit" form:"limit"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Query   string         `json:"query"`
	Total   int            `json:"total"`
}

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string      `json:"response"`
	Results  interface{} `json:"results,omitempty"`
}
