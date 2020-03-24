package explorer

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/threefoldtech/rivine/modules"
	"github.com/threefoldtech/rivine/pkg/api"
	"github.com/threefoldtech/rivine/types"
)

type (
	// Explorer is a backend which operates by querying a remote public explorer
	Explorer struct {
		userAgent string
		url       string
		password  string
		client    *http.Client
	}
)

// NewExplorer creates a new explorer client for a public explorer running at the given url, expecting the given user agent string.
// The api can optionally be password protected
func NewExplorer(url, userAgent, password string) *Explorer {
	return &Explorer{
		userAgent: userAgent,
		url:       url,
		password:  password,
		client:    &http.Client{Timeout: time.Second * 30},
	}
}

// CheckAddress returns all interesting transactions and blocks related to a given unlockhash
func (e *Explorer) CheckAddress(addr types.UnlockHash) ([]api.ExplorerBlock, []api.ExplorerTransaction, error) {
	body := api.ExplorerHashGET{}
	_, err := e.get("/explorer/hashes/"+addr.String(), &body)
	return body.Blocks, body.Transactions, err
}

// SendTxn posts a transaction to the explorer to include it in the transactionpool
func (e *Explorer) SendTxn(tx types.Transaction) (types.TransactionID, error) {
	_, err := e.post("/transactionpool/transactions", tx, nil)
	if err != nil {
		return types.TransactionID{}, err
	}
	return tx.ID(), nil
}

// CurrentHeight gets the current height of the explorer
func (e *Explorer) CurrentHeight() (types.BlockHeight, error) {
	body := api.ConsensusGET{}
	_, err := e.get("/explorer", &body)
	return body.Height, err
}

// GetChainConstants fetches the chainconstants used by the explorer
func (e *Explorer) GetChainConstants() (modules.DaemonConstants, error) {
	body := modules.DaemonConstants{}
	_, err := e.get("/explorer/constants", &body)
	return body, err
}

func (e *Explorer) Get(endpoint string) error {
	_, err := e.get(endpoint, nil)
	return err
}

func (e *Explorer) GetWithResponse(endpoint string, responseBody interface{}) error {
	_, err := e.get(endpoint, responseBody)
	return err
}

func (e *Explorer) Post(endpoint, data string) error {
	_, err := e.post(endpoint, data, nil)
	return err
}

func (e *Explorer) PostWithResponse(endpoint, data string, responseBody interface{}) error {
	_, err := e.post(endpoint, data, responseBody)
	return err
}

func (e *Explorer) get(endpoint string, responseBody interface{}) (*http.Response, error) {
	return e.request("GET", endpoint, nil, responseBody)
}

func (e *Explorer) post(endpoint string, body interface{}, responseBody interface{}) (*http.Response, error) {
	return e.request("POST", endpoint, body, responseBody)
}

func (e *Explorer) request(method string, endpoint string, body interface{}, responseBody interface{}) (*http.Response, error) {
	var br io.Reader
	if s, ok := body.(string); ok {
		br = strings.NewReader(s)
	} else if b, ok := body.([]byte); ok {
		br = bytes.NewReader(b)
	} else {
		buf := bytes.NewBuffer(nil)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
		br = buf
	}
	req, err := http.NewRequest(method, e.url+endpoint, br)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", e.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.SetBasicAuth("", e.password)
	res, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= http.StatusBadRequest {
		errBody := api.Error{}
		if err = json.NewDecoder(res.Body).Decode(&errBody); err != nil {
			return nil, err
		}
		return nil, errors.New(errBody.Message)
	}
	if responseBody != nil {
		err = json.NewDecoder(res.Body).Decode(responseBody)
		if err == io.EOF {
			// mask EOF errors, the caller needs to be able to handle the body not being present
			err = nil
		}
	}
	return res, err
}
