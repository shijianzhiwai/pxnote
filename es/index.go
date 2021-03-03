package es

// NoteIndex es note index.
type NoteIndex struct {
	Type           string `json:"type"` // Notion VuePress file etc...
	Title          string `json:"title"`
	Alive          bool   `json:"alive"`
	Index          int    `json:"Index"`
	PageID         string `json:"page_id"`
	BlockID        string `json:"block_id"`
	BlockContent   string `json:"block_content"`
	BlockType      string `json:"block_type"`
	CodeLanguage   string `json:"code_language,omitempty"`
	LastEditedTime int64  `json:"last_edited_time"`
	CreatedTime    int64  `json:"created_time"`
}

// index type
const (
	TypeNotion   = "Notion"
	TypeVuePress = "VuePress"
	TypeFile     = "file"
)
