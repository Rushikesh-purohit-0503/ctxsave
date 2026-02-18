package store

import "time"

type EntryType string

const (
	EntryConversation EntryType = "conversation"
	EntryCodeChange   EntryType = "code_change"
	EntryDecision     EntryType = "decision"
	EntryError        EntryType = "error"
	EntryGitCommit    EntryType = "git_commit"
	EntryGitDiff      EntryType = "git_diff"
	EntryNote         EntryType = "note"
	EntryFile         EntryType = "file"
)

type Session struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Source    string    `json:"source"`
	Project  string    `json:"project"`
	Label    string    `json:"label"`
}

type Entry struct {
	ID        int64     `json:"id"`
	SessionID string    `json:"session_id"`
	Type      EntryType `json:"type"`
	Content   string    `json:"content"`
	Metadata  string    `json:"metadata"`
	OrderIdx  int       `json:"order_idx"`
	CreatedAt time.Time `json:"created_at"`
}

type Summary struct {
	ID           int64     `json:"id"`
	SessionID    string    `json:"session_id"`
	Level        string    `json:"level"`
	Content      string    `json:"content"`
	TokenEstimate int      `json:"token_estimate"`
	CreatedAt    time.Time `json:"created_at"`
}
