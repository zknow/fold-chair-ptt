package ptt

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"

type Client struct {
	http *http.Client
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{},
	}
}

func (c *Client) Get(rawURL string) (string, error) {
	return c.GetCtx(context.Background(), rawURL, 20*time.Second)
}

func (c *Client) GetCtx(ctx context.Context, rawURL string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Language", "zh-TW,zh;q=0.9,en;q=0.8")
	req.AddCookie(&http.Cookie{Name: "over18", Value: "1"})

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", &statusError{code: resp.StatusCode}
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

type statusError struct {
	code int
}

func (e *statusError) Error() string {
	return fmt.Sprintf("HTTP %d", e.code)
}
