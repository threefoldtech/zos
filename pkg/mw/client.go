package mw

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

var (
	// ErrInvalidSignature is returned if a response identity can't
	// be verified
	ErrInvalidSignature = fmt.Errorf("failed to validate signature")
)

func SignedRequest(id uint32, sk ed25519.PrivateKey, req *http.Request) (*http.Request, error) {
	var buf bytes.Buffer
	hash := sha256.New()
	writer := io.MultiWriter(&buf, hash)
	if req.Body != nil {
		defer req.Body.Close()
		if _, err := io.Copy(writer, req.Body); err != nil {
			return nil, err
		}
	}

	sha := hash.Sum(nil)
	ts := fmt.Sprint(time.Now().Unix())
	var msg bytes.Buffer
	msg.WriteString(ts)
	msg.Write(sha)

	sig := ed25519.Sign(sk, msg.Bytes())
	req.Header.Set(twinHeader, fmt.Sprint(id))
	req.Header.Set(signatureHeader, hex.EncodeToString(sig))
	req.Header.Set(timestampHeader, ts)
	req.Body = io.NopCloser(&buf)

	return req, nil
}

func VerifyResponse(pk ed25519.PublicKey, response *http.Response) (*http.Response, error) {
	sig, err := hex.DecodeString(response.Header.Get(signatureHeader))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode signature")
	}
	ts := response.Header.Get(timestampHeader)
	var buf bytes.Buffer

	hash := sha256.New()
	writer := io.MultiWriter(&buf, hash)
	body := response.Body
	if _, err := io.Copy(writer, body); err != nil {
		return nil, err
	}
	sha := hash.Sum(nil)
	var msg bytes.Buffer
	msg.WriteString(ts)
	msg.Write(sha)

	if !ed25519.Verify(pk, msg.Bytes(), sig) {
		return nil, ErrInvalidSignature
	}

	response.Body = io.NopCloser(&buf)
	return response, nil
}

func VerifyRequest(pk ed25519.PublicKey, request *http.Request) (*http.Request, error) {
	sig, err := hex.DecodeString(request.Header.Get(signatureHeader))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode signature")
	}
	ts := request.Header.Get(timestampHeader)
	var buf bytes.Buffer

	hash := sha256.New()
	writer := io.MultiWriter(&buf, hash)
	body := request.Body
	if _, err := io.Copy(writer, body); err != nil {
		return nil, err
	}
	sha := hash.Sum(nil)
	var msg bytes.Buffer
	msg.WriteString(ts)
	msg.Write(sha)

	if !ed25519.Verify(pk, msg.Bytes(), sig) {
		return nil, ErrInvalidSignature
	}

	request.Body = io.NopCloser(&buf)
	return request, nil
}
