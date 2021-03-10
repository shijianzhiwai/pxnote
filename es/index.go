package es

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/olivere/elastic/v7"
)

// NoteIndex es note index.
type NoteIndex struct {
	Type           string `json:"type"` // Notion VuePress, file etc...
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

	IndexPrefix = "note_"
	IndexName   = "note_index"
)

type index struct {
	version int
	name    string
}

// Storage es storage, index note data.
type Storage struct {
	Client        *elastic.Client
	IndexProducer map[string]IndexInterface

	indexState map[string]int // index name => version
}

// NewESStorage Initialize es storage
func NewESStorage(url []string, snifferEnabled bool) (*Storage, error) {
	client, err := elastic.NewClient(
		elastic.SetSniff(snifferEnabled),
		elastic.SetURL(url...),
	)
	if err != nil {
		return nil, err
	}
	return &Storage{
		Client:        client,
		IndexProducer: make(map[string]IndexInterface),
		indexState:    make(map[string]int),
	}, nil
}

// AddIndex add an index data producer.
func (s *Storage) AddIndex(ind IndexInterface) {
	s.IndexProducer[ind.Type()] = ind
}

func (s *Storage) prepareIndex() error {
	s.indexState = make(map[string]int) // clear
	ctx, cfn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cfn()
	resp, err := s.Client.CatIndices().Do(ctx)
	if err != nil {
		return err
	}
	for _, v := range resp {
		if !strings.HasPrefix(v.Index, IndexPrefix) {
			continue
		}
		name, ver, err := parseIndexName(v.Index)
		if err != nil {
			return err
		}
		s.indexState[name] = ver
	}
	return nil
}

func (s *Storage) getCurIndex(name string) string {
	ret, ok := s.indexState[name]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s_%d", name, ret)
}

func (s *Storage) getNextIndex(name string) string {
	ret, ok := s.indexState[name]
	if !ok {
		return fmt.Sprintf("%s_0", name)
	}
	return fmt.Sprintf("%s_%d", name, ret+1)
}

// IndexAll index all producer.
func (s *Storage) IndexAll() error {
	err := s.prepareIndex()
	if err != nil {
		return err
	}
	for _, v := range s.IndexProducer {
		inds, err := v.FetchIndex()
		if err != nil {
			return err
		}
		nextIndex := s.getNextIndex(IndexName)
		_, err = s.Client.CreateIndex(nextIndex).BodyString(noteIndexMappings).Do(context.TODO())
		if err != nil {
			return fmt.Errorf("create index error: %v", err)
		}
		log.Printf("index type=%s data", v.Type())
		bu := s.Client.Bulk()
		for _, v := range inds {
			bu.Add(elastic.NewBulkIndexRequest().Index(nextIndex).Doc(v))
		}
		_, err = bu.Do(context.TODO())
		if err != nil {
			return err
		}
		// delete old version index
		curName := s.getCurIndex(IndexName)
		if curName != "" {
			log.Printf("delete old version index: %s", curName)
			_, err = s.Client.DeleteIndex(curName).Do(context.TODO())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func parseIndexName(str string) (name string, ver int, err error) {
	ret := strings.Split(str, "_")
	if len(ret) == 0 {
		err = fmt.Errorf("split index %s version error", str)
		return
	}

	ver, err = strconv.Atoi(ret[len(ret)-1])
	if err != nil {
		err = fmt.Errorf("parse index %s version error： %v", str, err)
		return
	}

	name = strings.Join(ret[:len(ret)-1], "_")
	return
}

const noteIndexMappings = `{
  "settings": {
    "index": {
      "refresh_interval": "5s",
      "number_of_shards": "1",
      "store": {
        "type": "mmapfs"
      }
    }
  },
  "mappings": {
    "properties": {
      "alive": {
        "type":  "boolean"
      },
      "index": {
        "type":  "integer"
      },
      "type": {
        "type":  "keyword"
      },
      "block_type": {
        "type":  "keyword"
      },
      "code_language": {
        "type":  "keyword"
      },
      "page_id": {
        "type":  "keyword"
      },
      "block_id": {
        "type":  "keyword"
      },
      "title": {
        "type": "text",
        "analyzer": "ik_max_word",
        "search_analyzer": "ik_smart"
      },
      "block_content": {
        "type": "text",
        "analyzer": "ik_max_word",
        "search_analyzer": "ik_smart"
      },
      "last_edited_time": {
        "type": "date",
        "format": "epoch_second"
      },
      "created_time": {
        "type": "date",
        "format": "epoch_second"
      }
    }
  }
}`
