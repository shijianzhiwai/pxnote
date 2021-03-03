package es

import (
	"context"
	"github.com/olivere/elastic/v7"
)

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

	IndexName = "note_index"
)

// Storage es storage, index note data.
type Storage struct {
	Client        *elastic.Client
	IndexProducer map[string]IndexInterface
}

// NewESStorage Initialize es storage
func NewESStorage(url string, snifferEnabled bool) (*Storage, error) {
	client, err := elastic.NewClient(
		elastic.SetSniff(snifferEnabled),
		elastic.SetURL(url),
	)
	if err != nil {
		return nil, err
	}
	return &Storage{
		Client:        client,
		IndexProducer: make(map[string]IndexInterface),
	}, nil
}

// AddIndex add an index data producer.
func (s *Storage) AddIndex(ind IndexInterface) {
	s.IndexProducer[ind.Type()] = ind
}

// IndexAll index all producer.
func (s *Storage) IndexAll() error {
	for _, v := range s.IndexProducer {
		inds, err := v.FetchIndex()
		if err != nil {
			return err
		}
		bu := s.Client.Bulk()
		for _, v := range inds {
			bu.Add(elastic.NewBulkIndexRequest().Index(IndexName).Doc(v))
		}
		_, err = bu.Do(context.Background())
		if err != nil {
			return err
		}
	}
	return nil
}
