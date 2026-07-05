package coordination

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// VertexGateway is the production transport boundary. Endpoint is the fully
// qualified Vertex generateContent URL; Token is expected to be short-lived.
type VertexGateway struct {
	Endpoint string
	Token    string
	Client   *http.Client
}

func (g VertexGateway) Generate(ctx context.Context, request ModelRequest) (ModelResponse, error) {
	if g.Endpoint == "" || g.Token == "" {
		return ModelResponse{}, fmt.Errorf("vertex gateway is not configured")
	}
	body := map[string]any{
		"systemInstruction": map[string]any{
			"parts": []map[string]string{{"text": request.SystemInstruction}},
		},
		"contents": []map[string]any{{
			"role": "user",
			"parts": []map[string]string{{
				"text": request.Task + "\n" + request.UntrustedEvidence,
			}},
		}},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
		},
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.Endpoint, bytes.NewReader(raw))
	if err != nil {
		return ModelResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+g.Token)
	req.Header.Set("Content-Type", "application/json")
	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return ModelResponse{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return ModelResponse{}, err
	}
	if resp.StatusCode/100 != 2 {
		return ModelResponse{}, fmt.Errorf("vertex returned %d", resp.StatusCode)
	}
	var envelope struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Usage struct {
			Prompt     int `json:"promptTokenCount"`
			Candidates int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}
	if json.Unmarshal(data, &envelope) != nil || len(envelope.Candidates) == 0 || len(envelope.Candidates[0].Content.Parts) == 0 {
		return ModelResponse{}, fmt.Errorf("invalid vertex response")
	}
	text := strings.TrimSpace(envelope.Candidates[0].Content.Parts[0].Text)
	var proposals []ModelProposal
	if err = json.Unmarshal([]byte(text), &proposals); err != nil {
		var wrapped struct {
			Proposals []ModelProposal `json:"proposals"`
		}
		if err = json.Unmarshal([]byte(text), &wrapped); err != nil {
			return ModelResponse{}, fmt.Errorf("invalid structured model output: %w", err)
		}
		proposals = wrapped.Proposals
	}
	return ModelResponse{
		Proposals:    proposals,
		InputTokens:  envelope.Usage.Prompt,
		OutputTokens: envelope.Usage.Candidates,
	}, nil
}
