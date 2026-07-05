package coordination

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// HTTPMetricsClient implements the read-only Prometheus HTTP API contract.
type HTTPMetricsClient struct {
	BaseURL string
	Client  *http.Client
}

func (c HTTPMetricsClient) Query(ctx context.Context, query string, start, end time.Time, q ContextQuery) (any, string, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		return nil, "", fmt.Errorf("prometheus URL is empty")
	}
	v := url.Values{"query": {query}, "start": {formatUnix(start)}, "end": {formatUnix(end)}}
	if q.Step != "" {
		v.Set("step", q.Step)
	} else {
		v.Set("step", "30s")
	}
	sourceURL := base + "/graph?g0.expr=" + url.QueryEscape(query)
	var body struct {
		Status string `json:"status"`
		Data   any    `json:"data"`
		Error  string `json:"error"`
	}
	if err := getJSON(ctx, c.Client, base+"/api/v1/query_range?"+v.Encode(), &body); err != nil {
		return nil, sourceURL, err
	}
	if body.Status != "success" {
		return nil, sourceURL, fmt.Errorf("prometheus: %s", body.Error)
	}
	return body.Data, sourceURL, nil
}

// HTTPLokiClient implements Loki's range query API and enforces recipe limits.
type HTTPLokiClient struct {
	BaseURL string
	Client  *http.Client
}

func (c HTTPLokiClient) Query(ctx context.Context, query string, start, end time.Time, q ContextQuery) (any, string, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		return nil, "", fmt.Errorf("loki URL is empty")
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 200
	}
	v := url.Values{"query": {query}, "start": {strconv.FormatInt(start.UnixNano(), 10)}, "end": {strconv.FormatInt(end.UnixNano(), 10)}, "limit": {strconv.Itoa(limit)}, "direction": {"backward"}}
	sourceURL := base + "/explore?query=" + url.QueryEscape(query)
	var body struct {
		Status string `json:"status"`
		Data   any    `json:"data"`
		Error  string `json:"error"`
	}
	if err := getJSON(ctx, c.Client, base+"/loki/api/v1/query_range?"+v.Encode(), &body); err != nil {
		return nil, sourceURL, err
	}
	if body.Status != "success" {
		return nil, sourceURL, fmt.Errorf("loki: %s", body.Error)
	}
	return body.Data, sourceURL, nil
}
func formatUnix(t time.Time) string {
	return strconv.FormatFloat(float64(t.UnixNano())/1e9, 'f', 3, 64)
}
func getJSON(ctx context.Context, client *http.Client, endpoint string, out any) error {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("upstream status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
