package mw

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"net/http"
	"sync"
	"time"
)

const (
	signatureHeader = "x-signature"
	twinHeader      = "x-twin"
	timestampHeader = "x-timestamp"
)

type Signer struct {
	sk ed25519.PrivateKey
}

func NewSigner(sk ed25519.PrivateKey) Signer {
	return Signer{sk}
}

func (s *Signer) Action(action Action) http.HandlerFunc {
	inner := AsAction(action)

	return func(w http.ResponseWriter, r *http.Request) {
		capture := newCaptureWriter()
		inner.ServeHTTP(capture, r)

		// capture now contains all the content written by the action
		headers := w.Header()
		for k, v := range capture.Header() {
			headers[k] = v
		}
		ts := time.Now().Unix()
		headers.Set(timestampHeader, fmt.Sprint(ts))

		var msg bytes.Buffer
		msg.WriteString(fmt.Sprint(ts))
		msg.Write(capture.Hash())
		signature := ed25519.Sign(s.sk, msg.Bytes())
		headers.Set(signatureHeader, hex.EncodeToString(signature))

		if capture.code != -1 {
			w.WriteHeader(capture.code)
		}

		w.Write(capture.body.Bytes())
	}
}

type captureWriter struct {
	body   bytes.Buffer
	sha    hash.Hash
	header http.Header
	code   int
	o      sync.Once
}

func newCaptureWriter() *captureWriter {
	return &captureWriter{
		header: make(http.Header),
		sha:    sha256.New(),
		code:   -1,
	}
}

var (
	_ http.ResponseWriter = (*captureWriter)(nil)
)

func (c *captureWriter) Hash() []byte {
	return c.sha.Sum(nil)
}

func (c *captureWriter) Header() http.Header {
	return c.header
}

func (c *captureWriter) Write(p []byte) (int, error) {
	if n, err := c.sha.Write(p); err != nil {
		return n, err
	}

	return c.body.Write(p)
}

func (c *captureWriter) WriteHeader(statusCode int) {
	c.o.Do(func() {
		c.code = statusCode
	})
}
