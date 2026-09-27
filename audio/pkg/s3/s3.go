// Package s3 is the minimal S3 client for the audio service (spec 06):
// only PutObject/GetObject/DeleteObject/CreateBucket, SigV4 over net/http.
package s3

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client talks to one bucket on an S3-compatible endpoint (RustFS).
type Client struct {
	Endpoint string // e.g. http://rustfs:9000, no trailing slash
	Bucket   string // e.g. emergency-audio
	Key      string
	Secret   string
	HTTP     *http.Client
}

// New builds a client with a 30s default transport.
func New(endpoint, bucket, key, secret string) *Client {
	return &Client{
		Endpoint: strings.TrimSuffix(endpoint, "/"),
		Bucket:   bucket,
		Key:      key,
		Secret:   secret,
		HTTP:     &http.Client{Timeout: 30 * time.Second},
	}
}

func sha256hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

// sign builds SigV4 Authorization + x-amz-* headers for one request.
// body is hashed as UNSIGNED-PAYLOAD for streaming safety; RustFS accepts
// both signed and unsigned payload modes.
func (c *Client) sign(req *http.Request, bodyHash string) {
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", bodyHash)
	payload := bodyHash

	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		req.URL.Host, bodyHash, amzDate)
	canonical := strings.Join([]string{
		req.Method,
		req.URL.EscapedPath(),
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeaders,
		payload,
	}, "\n")

	region := "us-east-1"
	scope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, region)
	toSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		sha256hex([]byte(canonical)),
	}, "\n")

	kDate := hmacSHA256([]byte("AWS4"+c.Secret), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, "s3")
	kSigning := hmacSHA256(kService, "aws4_request")
	sig := hex.EncodeToString(hmacSHA256(kSigning, toSign))
	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		c.Key, scope, signedHeaders, sig))
}

func (c *Client) url(key string) string {
	return fmt.Sprintf("%s/%s/%s", c.Endpoint, c.Bucket, strings.TrimPrefix(key, "/"))
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		resp.Body.Close()
		return nil, fmt.Errorf("s3 %s %s: %s: %s", req.Method, req.URL.Path, resp.Status, strings.TrimSpace(string(body)))
	}
	return resp, nil
}

// EnsureBucket creates the bucket; existing buckets (200/409) both pass.
func (c *Client) EnsureBucket(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "PUT", c.url(""), nil)
	if err != nil {
		return err
	}
	c.sign(req, sha256hex(nil))
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
	if resp.StatusCode == 200 || resp.StatusCode == 409 {
		return nil
	}
	return fmt.Errorf("s3 create bucket: %s", resp.Status)
}

// Put uploads body as key with immutable wav headers.
func (c *Client) Put(ctx context.Context, key string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, "PUT", c.url(key), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "audio/wav")
	req.Header.Set("Cache-Control", "immutable")
	req.ContentLength = int64(len(body))
	c.sign(req, sha256hex(body))
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
	return nil
}

// Get downloads key fully (blobs are ~48KB; no range reads needed).
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.url(key), nil)
	if err != nil {
		return nil, err
	}
	c.sign(req, sha256hex(nil))
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, 32<<20))
}

// Delete removes key; missing keys (404/NoSuchKey) are success (sweep idempotency).
func (c *Client) Delete(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", c.url(key), nil)
	if err != nil {
		return err
	}
	c.sign(req, sha256hex(nil))
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
	if resp.StatusCode == 200 || resp.StatusCode == 204 || resp.StatusCode == 404 {
		return nil
	}
	return fmt.Errorf("s3 delete %s: %s", key, resp.Status)
}
