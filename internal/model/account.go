package model

import "time"

type AccountStatus string

const (
	StatusOperational  AccountStatus = "OPERATIONAL"
	StatusAuthRequired AccountStatus = "AUTH_REQUIRED"
	StatusCheckpoint   AccountStatus = "CHECKPOINT"
	StatusInterrupted  AccountStatus = "INTERRUPTED"
	StatusPaused       AccountStatus = "PAUSED"
	StatusConnecting   AccountStatus = "CONNECTING"
)

type Account struct {
	Object       string         `json:"object"`
	ID           string         `json:"id"`
	Provider     string         `json:"provider"`
	Name         string         `json:"name"`
	Identifier   string         `json:"identifier"`
	Status       AccountStatus  `json:"status"`
	StatusDetail *string        `json:"status_detail,omitempty"`
	Capabilities []string       `json:"capabilities"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	LastSyncedAt *time.Time     `json:"last_synced_at,omitempty"`
	Proxy        *ProxyConfig   `json:"proxy,omitempty"`
	Metadata     map[string]any `json:"metadata"`
}

type ProxyConfig struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}
