package jenkins

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const APISuffix = "api/json"

var defaultLogger = log.New(io.Discard, "", 0)

type auth struct {
	username string
	password string
}

type Client struct {
	client    http.Client
	auth      *auth
	serverURL string
	logger    *log.Logger
}

func (c *Client) WithAuth(username, password string) *Client {
	c.auth = &auth{username: username, password: password}

	return c
}

func (c *Client) WithTimeout(timeout time.Duration) *Client {
	c.client.Timeout = timeout

	return c
}

func (c *Client) WithLogger(l *log.Logger) *Client {
	c.logger = l

	return c
}

func NewClient(url string) *Client {
	if len(url) > 0 && url[len(url)-1] != '/' {
		url += "/"
	}

	return &Client{
		client:    http.Client{},
		serverURL: url,
		logger:    defaultLogger,
	}
}

type ErrHTTPRequestFailed struct {
	Code int
}

func (e *ErrHTTPRequestFailed) Error() string {
	return fmt.Sprintf("HTTP request failed with code: %d", e.Code)
}

func (c *Client) do(method, url string, result interface{}) error {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return err
	}

	if c.auth != nil {
		req.SetBasicAuth(c.auth.username, c.auth.password)
	}

	c.logger.Printf("sending HTTP %s request to %s", req.Method, req.URL)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		//  body has always be read until EOF and closed
		_, _ = io.ReadAll(resp.Body)
		return &ErrHTTPRequestFailed{Code: resp.StatusCode}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, result)
	if err != nil {
		return err
	}

	return nil
}
