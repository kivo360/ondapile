package model

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type PaginatedList[T any] struct {
	Object  string `json:"object"`
	Items   []T    `json:"items"`
	Cursor  string `json:"cursor"`
	HasMore bool   `json:"has_more"`
}

type ListOptions struct {
	Cursor    string  `form:"cursor"`
	Limit     int     `form:"limit"`
	Before    *string `form:"before"`
	After     *string `form:"after"`
	AccountID *string `form:"account_id"`
	Provider  *string `form:"provider"`
	IsGroup   *bool   `form:"is_group"`
}

type CursorData struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at,omitempty"`
}

func (o *ListOptions) GetLimit() int {
	if o.Limit <= 0 {
		return 25
	}
	if o.Limit > 100 {
		return 100
	}
	return o.Limit
}

func (o *ListOptions) GetCursor() (*CursorData, error) {
	if o.Cursor == "" {
		return nil, nil
	}
	data, err := base64.URLEncoding.DecodeString(o.Cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	var cursor CursorData
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	return &cursor, nil
}

func EncodeCursor(id, createdAt string) string {
	data := CursorData{ID: id, CreatedAt: createdAt}
	jsonData, _ := json.Marshal(data)
	return base64.URLEncoding.EncodeToString(jsonData)
}

func NewPaginatedList[T any](items []T, cursor string, hasMore bool) *PaginatedList[T] {
	return &PaginatedList[T]{
		Object:  "list",
		Items:   items,
		Cursor:  cursor,
		HasMore: hasMore,
	}
}
