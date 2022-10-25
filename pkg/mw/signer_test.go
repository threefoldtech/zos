package mw

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestSigner(t *testing.T) {
	action := func(r *http.Request) (interface{}, Response) {
		return "Hello World", Created()
	}

	pk, sk, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	signer := NewSigner(sk)

	handler := signer.Action(action)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/", nil)
	handler(recorder, request)

	response := recorder.Result()
	require.EqualValues(t, http.StatusCreated, response.StatusCode)
	ts := response.Header.Get("x-timestamp")
	require.NotEmpty(t, ts)

	signature := response.Header.Get("x-signature")
	require.NotEmpty(t, signature)
	sig, err := hex.DecodeString(signature)
	require.NoError(t, err)

	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	var msg bytes.Buffer
	msg.WriteString(ts)
	hash := sha256.Sum256(data)
	msg.Write(hash[:])

	require.True(t, ed25519.Verify(pk, msg.Bytes(), sig))

	var returned string
	err = json.Unmarshal(data, &returned)
	require.NoError(t, err)

	require.Equal(t, "Hello World", returned)
}

type twinsMap map[uint32]ed25519.PublicKey

func (t twinsMap) GetKey(id uint32) ([]byte, error) {
	pk, ok := t[id]
	if !ok {
		return nil, fmt.Errorf("twin not found")
	}
	return pk, nil
}

// TestSignerClientNotAuthorized test authorization when NO auth info is given
func TestSignerClientNotAuthorized(t *testing.T) {
	action := func(r *http.Request) (interface{}, Response) {
		return "Hello World", Created()
	}
	_, sk, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	signer := NewSigner(sk)

	router := mux.NewRouter()
	twins := make(twinsMap)
	mw := NewAuthMiddleware(twins)
	router.Use(mw.Middleware)
	router.Handle("/test", signer.Action(action))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/test", nil)
	router.ServeHTTP(recorder, request)

	response := recorder.Result()
	require.EqualValues(t, http.StatusUnauthorized, response.StatusCode)
}

// TestSignerClientAuth test authorization when real auth info
func TestSignerClientAuth(t *testing.T) {
	action := func(r *http.Request) (interface{}, Response) {
		return "Hello World", Created()
	}

	serverPk, serverSk, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	clientPk, clientSk, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	signer := NewSigner(serverSk)

	router := mux.NewRouter()
	twins := make(twinsMap)
	twins[10] = clientPk

	mw := NewAuthMiddleware(twins)
	router.Use(mw.Middleware)
	router.Handle("/test", signer.Action(action))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/test", nil)
	SignedRequest(10, clientSk, request)
	router.ServeHTTP(recorder, request)

	response := recorder.Result()
	require.EqualValues(t, http.StatusCreated, response.StatusCode)

	response, err = VerifyResponse(serverPk, response)
	require.NoError(t, err)

	var returned string
	dec := json.NewDecoder(response.Body)
	err = dec.Decode(&returned)
	require.NoError(t, err)

	require.Equal(t, "Hello World", returned)

}
