// Package graphql for grid graphql support
package graphql

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

// GraphQl for tf graphql
type GraphQl struct {
	url string
}

// NewGraphQl creates new tf graphql
func NewGraphQl(url string) (GraphQl, error) {
	return GraphQl{url}, nil
}

// Query queries graphql
func (g *GraphQl) Query(body string, variables map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	requestBody := map[string]interface{}{"query": body, "variables": variables}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return result, err
	}

	bodyReader := bytes.NewReader(jsonBody)

	resp, err := http.Post(g.url, "application/json", bodyReader)
	if err != nil {
		return result, err
	}

	queryData, err := parseHTTPResponse(resp)
	if err != nil {
		return result, err
	}

	result = queryData["data"].(map[string]interface{})
	return result, nil
}

func parseHTTPResponse(resp *http.Response) (map[string]interface{}, error) {
	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]interface{}{}, err
	}

	defer resp.Body.Close()

	var data map[string]interface{}
	err = json.Unmarshal(resBody, &data)
	if err != nil {
		return map[string]interface{}{}, err
	}

	if resp.StatusCode >= 400 {
		return map[string]interface{}{}, errors.Errorf("request failed with status code: %d with error %v", resp.StatusCode, data)
	}

	return data, nil
}
