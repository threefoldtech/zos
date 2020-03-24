package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/phonebook"
)

const (
	httpContentType = "application/json"
)

var (
	successCodes = []int{
		http.StatusOK,
		http.StatusCreated,
	}
)

type httpClient struct {
	u  *url.URL
	cl http.Client
}

func newHTTPClient(raw string) (*httpClient, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, errors.Wrap(err, "invalid url")
	}

	return &httpClient{u: u}, nil
}

func (c *httpClient) url(p ...string) string {
	b := *c.u
	b.Path = filepath.Join(b.Path, filepath.Join(p...))

	return b.String()
}

func (c *httpClient) process(response *http.Response, output interface{}, expect ...int) error {
	defer response.Body.Close()

	if len(expect) == 0 {
		expect = successCodes
	}

	in := func(i int, l []int) bool {
		for _, x := range l {
			if x == i {
				return true
			}
		}
		return false
	}

	dec := json.NewDecoder(response.Body)
	if !in(response.StatusCode, expect) {
		var output struct {
			E string `json:"error"`
		}

		if err := dec.Decode(&output); err != nil {
			return errors.Wrapf(err, "failed to load error while processing invalid return code of: %s", response.StatusCode)
		}

		return fmt.Errorf("%s: %s", response.Status, output.E)
	}

	if err := dec.Decode(output); err != nil {
		return errors.Wrap(err, "failed to load output")
	}

	return nil
}

func (c *httpClient) post(u string, input interface{}, output interface{}, expect ...int) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(input); err != nil {
		return errors.Wrap(err, "failed to serialize request body")
	}

	response, err := http.Post(u, httpContentType, &buf)
	if err != nil {
		return errors.Wrap(err, "failed to send request")
	}

	return c.process(response, output, expect...)
}

func (c *httpClient) put(u string, input interface{}, output interface{}, expect ...int) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(input); err != nil {
		return errors.Wrap(err, "failed to serialize request body")
	}
	req, err := http.NewRequest(http.MethodPut, u, &buf)
	if err != nil {
		return errors.Wrap(err, "failed to build request")
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send request")
	}

	return c.process(response, output, expect...)
}

func (c *httpClient) get(u string, query url.Values, output interface{}, expect ...int) error {
	if len(query) > 0 {
		u = fmt.Sprintf("%s?%s", u, query.Encode())
	}

	response, err := http.Get(u)
	if err != nil {
		return errors.Wrap(err, "failed to send request")
	}

	return c.process(response, output, expect...)
}

type httpPhonebook struct {
	*httpClient
}

func (p *httpPhonebook) Create(user phonebook.TfgridPhonebookUser1) (schema.ID, error) {
	var out phonebook.TfgridPhonebookUser1
	if err := p.post(p.url("users"), user, &out); err != nil {
		return 0, err
	}

	return out.ID, nil
}

func (p *httpPhonebook) List(name, email string, page *Pager) (output []phonebook.TfgridPhonebookUser1, err error) {
	query := url.Values{}
	page.apply(query)
	if len(name) != 0 {
		query.Set("name", name)
	}
	if len(email) != 0 {
		query.Set("email", email)
	}

	err = p.get(p.url("users"), query, &output, http.StatusOK)

	return
}

func (p *httpPhonebook) Get(id schema.ID) (user phonebook.TfgridPhonebookUser1, err error) {
	err = p.get(p.url("users", fmt.Sprint(id)), nil, &user, http.StatusOK)
	return
}

// Validate the signature of this message for the user, signature and message are hex encoded
func (p *httpPhonebook) Validate(id schema.ID, message, signature string) (bool, error) {
	var input struct {
		S string `json:"signature"`
		M string `json:"payload"`
	}
	input.S = signature
	input.M = message

	var output struct {
		V bool `json:"is_valid"`
	}

	err := p.post(p.url("users", fmt.Sprint(id), "validate"), input, &output, http.StatusOK)
	if err != nil {
		return false, err
	}

	return output.V, nil
}
