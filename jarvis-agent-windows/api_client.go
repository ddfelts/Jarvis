/*
The api_client.go file is a client for the API. It is used to send requests to the API and receive responses.
*/

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type APIRequest struct {
	Payload    interface{}
	Result     interface{}
	ResponseCh chan<- APIResponse
	Config     APILogRemote
}

type APIResponse struct {
	Error error
	Data  interface{}
}

type APIClient struct {
	requestCh chan APIRequest
	workers   int
	wg        *sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewAPIClient(workers int, wg *sync.WaitGroup) *APIClient {
	ctx, cancel := context.WithCancel(context.Background())
	client := &APIClient{
		requestCh: make(chan APIRequest, 1000),
		workers:   workers,
		wg:        wg,
		ctx:       ctx,
		cancel:    cancel,
	}
	client.Start()
	return client
}

func (c *APIClient) Start() {
	for i := 0; i < c.workers; i++ {
		c.wg.Add(1)
		go c.processRequests()
	}
}

func (c *APIClient) processRequests() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case req, ok := <-c.requestCh:
			if !ok {
				return
			}
			client := createHTTPClient(req.Config)
			resp, err := s_Request(client, req)

			select {
			case <-c.ctx.Done():
				return
			default:
				if req.ResponseCh != nil {
					req.ResponseCh <- APIResponse{
						Error: err,
						Data:  resp,
					}
					close(req.ResponseCh)
				}
			}
		}
	}
}

func createHTTPClient(config APILogRemote) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.APISkipVerify,
		},
	}

	return &http.Client{
		Transport: transport,
		Timeout:   time.Duration(config.APITimeout) * time.Second,
	}
}

func s_Request(client *http.Client, req APIRequest) (interface{}, error) {
	// Debug log the request configuration
	log.Printf("API Request Config - URL: %s, Method: %s", req.Config.APIURL, req.Config.APIMethod)

	var body io.Reader
	if req.Payload != nil {
		jsonData, err := json.Marshal(req.Payload)
		if err != nil {
			return nil, fmt.Errorf("error marshaling payload: %v", err)
		}
		log.Printf("Sending request to %s: %s", req.Config.APIURL, string(jsonData))
		body = bytes.NewBuffer(jsonData)
	}

	httpReq, err := http.NewRequest(req.Config.APIMethod, req.Config.APIURL, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Add headers
	for _, h := range req.Config.APIHeaders {
		httpReq.Header.Set(h.Key, h.Value)
	}

	// Add auth header
	if req.Config.APIAuthType == "apikey" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", req.Config.APIKey))
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result interface{}
	if req.Result != nil {
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("error parsing response: %v", err)
		}
	}

	return result, nil
}

func (c *APIClient) SendRequest(payload interface{}, config APILogRemote) chan APIResponse {
	responseCh := make(chan APIResponse, 1)

	select {
	case <-c.ctx.Done():
		close(responseCh)
		return responseCh
	case c.requestCh <- APIRequest{
		Payload:    payload,
		ResponseCh: responseCh,
		Config:     config,
	}:
	default:
		// Channel is full, log error and return
		log.Printf("API request channel is full, dropping request")
		close(responseCh)
	}
	return responseCh
}

func (c *APIClient) Close() {
	c.cancel() // Cancel context first
	if c.requestCh != nil {
		close(c.requestCh)
		c.requestCh = nil
	}
	// Wait for all workers to finish
	c.wg.Wait()
}
