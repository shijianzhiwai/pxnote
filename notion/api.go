package notion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kjk/notionapi"
	"github.com/shijianzhiwai/pxnote/es"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const (
	notionHost = "https://www.notion.so"
	userAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/69.0.3483.0 Safari/537.36"
	acceptLang = "en-US,en;q=0.9"
)

// API notion API:
// - get pages in all spaces
// - return pages index info
type API struct {
	Client     *notionapi.Client
	HTTPClient *http.Client
}

// Conf notion API conf.
type Conf struct {
	AuthToken string        `json:"auth_token" yaml:"auth_token"`
	Timeout   time.Duration `json:"timeout" yaml:"timeout"`
}

// NewNotionAPI initialize notion API.
func NewNotionAPI(conf *Conf) *API {
	if conf.Timeout == 0 {
		conf.Timeout = 20 * time.Second
	}
	httpClient := &http.Client{
		Timeout: conf.Timeout,
	}
	api := API{
		Client: &notionapi.Client{
			AuthToken:  conf.AuthToken,
			HTTPClient: httpClient,
		},
		HTTPClient: httpClient,
	}
	return &api
}

// Type index type name.
func (a *API) Type() string {
	return es.TypeNotion
}

// FetchIndex index
func (a *API) FetchIndex() ([]es.NoteIndex, error) {
	ret, err := a.GetAllRootPagesID()
	if err != nil {
		return nil, err
	}
	allNoteIndex := make([]es.NoteIndex, 0)
	for _, v := range ret {
		ind, err := a.GetPageIndexData(v)
		if err != nil {
			return nil, err
		}
		allNoteIndex = append(allNoteIndex, ind...)
	}
	return allNoteIndex, nil
}

// GetPageIndexData get notion page content index data with sub page by ID.
func (a *API) GetPageIndexData(id string) ([]es.NoteIndex, error) {
	page, err := a.Client.DownloadPage(id)
	if err != nil {
		return nil, err
	}
	var (
		ind   int
		title string
	)
	notes := make([]es.NoteIndex, 0)
	page.ForEachBlock(func(block *notionapi.Block) {
		if ind == 0 && block.Type == "page" {
			title = block.Title
		}
		b := es.NoteIndex{
			Type:           es.TypeNotion,
			Title:          title,
			PageID:         id,
			Alive:          block.Alive,
			Index:          ind,
			BlockID:        block.ID,
			BlockType:      block.Type,
			CreatedTime:    block.CreatedTime,
			LastEditedTime: block.LastEditedTime,
		}

		if block.IsCode() {
			b.BlockContent = block.Code
			b.CodeLanguage = block.CodeLanguage
		} else {
			strs := make([]string, 0, len(block.InlineContent))
			for _, v := range block.InlineContent {
				strs = append(strs, v.Text)
			}
			b.BlockContent = strings.Join(strs, "\n")
		}

		notes = append(notes, b)
		ind++
	})
	for _, sub := range page.GetSubPages() {
		pgc, err := a.GetPageIndexData(sub)
		if err != nil {
			return nil, err
		}
		notes = append(notes, pgc...)
	}
	return notes, nil
}

// GetAllRootPagesID get notion all root pages ID.
func (a *API) GetAllRootPagesID() ([]string, error) {
	resp, err := a.LoadUserContent()
	if err != nil {
		return nil, err
	}
	ret := make([]string, 0)
	if val, ok := resp["block"]; ok {
		for key := range val {
			ret = append(ret, key)
		}
	}
	return ret, nil
}

// LoadUserContent pkg github.com/kjk/notionapi LoadUserContent func rewrite.
func (a *API) LoadUserContent() (map[string]map[string]*notionapi.LoadUserResponse, error) {
	apiURL := "/api/v3/loadUserContent"
	var rsp struct {
		RecordMap map[string]map[string]*notionapi.LoadUserResponse `json:"recordMap"`
	}
	return rsp.RecordMap, a.doNotionAPI(apiURL, struct{}{}, &rsp)
}

func (a *API) doNotionAPI(apiURL string, requestData interface{}, result interface{}) error {
	var js []byte
	var err error
	if requestData != nil {
		js, err = json.Marshal(requestData)
		if err != nil {
			return err
		}
	}
	uri := notionHost + apiURL
	body := bytes.NewBuffer(js)

	req, err := http.NewRequest("POST", uri, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Language", acceptLang)
	if a.Client.AuthToken != "" {
		req.Header.Set("cookie", fmt.Sprintf("token_v2=%v", a.Client.AuthToken))
	}
	var rsp *http.Response
	rsp, err = a.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != 200 {
		return fmt.Errorf("http.Post('%s') returned non-200 status code of %d", uri, rsp.StatusCode)
	}
	d, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(d, result)
	if err != nil {
		return err
	}
	return nil
}
