package db

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to Supabase's REST API (PostgREST), which sits in front of
// your project's Postgres database and is what @supabase/supabase-js calls
// under the hood on the frontend. There's no official Supabase Go SDK, so Go
// talks to the same HTTP API directly.
type Client struct {
	baseURL    string // e.g. https://xxxx.supabase.co/rest/v1
	serviceKey string // Supabase service_role key — bypasses RLS, server-side only
	httpClient *http.Client
}

// New creates a Supabase REST client.
//
//	projectURL: your project URL, e.g. https://xxxx.supabase.co
//	serviceKey: the service_role key from Settings -> API.
//	            NEVER send this to the frontend — it bypasses Row Level Security.
func New(projectURL, serviceKey string) *Client {
	return &Client{
		baseURL:    strings.TrimSuffix(projectURL, "/") + "/rest/v1",
		serviceKey: serviceKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string, query url.Values, body interface{}) (*http.Request, error) {
	fullURL := c.baseURL + "/" + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", c.serviceKey)
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c *Client) do(req *http.Request) ([]byte, int, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("supabase error (%d): %s", resp.StatusCode, string(data))
	}
	return data, resp.StatusCode, nil
}

// Select performs a GET against the table's REST endpoint.
// filters use PostgREST operator syntax, e.g. map[string]string{"status": "eq.active"}.
// orderBy uses PostgREST syntax too, e.g. "created_at.desc".
func (c *Client) Select(table, selectCols string, filters map[string]string, orderBy string) ([]byte, error) {
	if selectCols == "" {
		selectCols = "*"
	}
	q := url.Values{"select": {selectCols}}
	for k, v := range filters {
		q.Set(k, v)
	}
	if orderBy != "" {
		q.Set("order", orderBy)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := c.newRequest(ctx, http.MethodGet, table, q, nil)
	if err != nil {
		return nil, fmt.Errorf("db select %s: %w", table, err)
	}
	data, _, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("db select %s: %w", table, err)
	}
	return data, nil
}

// SelectSingle returns a single row, or (nil, nil) if none found.
func (c *Client) SelectSingle(table, selectCols string, filters map[string]string) ([]byte, error) {
	if selectCols == "" {
		selectCols = "*"
	}
	q := url.Values{"select": {selectCols}, "limit": {"1"}}
	for k, v := range filters {
		q.Set(k, v)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := c.newRequest(ctx, http.MethodGet, table, q, nil)
	if err != nil {
		return nil, fmt.Errorf("db selectSingle %s: %w", table, err)
	}
	// Ask PostgREST for a single JSON object instead of a one-element array.
	req.Header.Set("Accept", "application/vnd.pgrst.object+json")

	data, status, err := c.do(req)
	if err != nil {
		if status == http.StatusNotFound || strings.Contains(err.Error(), "PGRST116") {
			return nil, nil // no matching row
		}
		return nil, fmt.Errorf("db selectSingle %s: %w", table, err)
	}
	return data, nil
}

// Insert creates a row and returns the inserted row(s).
// selectCols controls which columns come back (default "*").
func (c *Client) Insert(table string, row interface{}, selectCols string) ([]byte, error) {
	if selectCols == "" {
		selectCols = "*"
	}
	q := url.Values{"select": {selectCols}}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := c.newRequest(ctx, http.MethodPost, table, q, row)
	if err != nil {
		return nil, fmt.Errorf("db insert %s: %w", table, err)
	}
	req.Header.Set("Prefer", "return=representation")

	data, _, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("db insert %s: %w", table, err)
	}
	return data, nil
}

// Upsert inserts a row, or updates it in place if a row with the same value(s)
// in conflictCol already exists (Postgres ON CONFLICT, via PostgREST's
// Prefer: resolution=merge-duplicates). conflictCol is typically "id".
func (c *Client) Upsert(table string, row interface{}, conflictCol, selectCols string) ([]byte, error) {
	if selectCols == "" {
		selectCols = "*"
	}
	q := url.Values{"select": {selectCols}}
	if conflictCol != "" {
		q.Set("on_conflict", conflictCol)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := c.newRequest(ctx, http.MethodPost, table, q, row)
	if err != nil {
		return nil, fmt.Errorf("db upsert %s: %w", table, err)
	}
	req.Header.Set("Prefer", "resolution=merge-duplicates,return=representation")

	data, _, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("db upsert %s: %w", table, err)
	}
	return data, nil
}

// Update patches rows matching filters and returns the updated row(s).
// Refuses to run with no filters, since an empty filter set would patch
// every row in the table.
func (c *Client) Update(table string, updates interface{}, filters map[string]string, selectCols string) ([]byte, error) {
	if len(filters) == 0 {
		return nil, fmt.Errorf("db update %s: refusing update with no filters (would affect the entire table)", table)
	}
	if selectCols == "" {
		selectCols = "*"
	}
	q := url.Values{"select": {selectCols}}
	for k, v := range filters {
		q.Set(k, v)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := c.newRequest(ctx, http.MethodPatch, table, q, updates)
	if err != nil {
		return nil, fmt.Errorf("db update %s: %w", table, err)
	}
	req.Header.Set("Prefer", "return=representation")

	data, _, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("db update %s: %w", table, err)
	}
	return data, nil
}
