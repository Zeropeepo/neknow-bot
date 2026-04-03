package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Zeropeepo/neknow-bot/internal/chat/domain"
	"github.com/Zeropeepo/neknow-bot/pkg/config"
)

type RAGClient struct {
	baseURL    string
	httpClient *http.Client
}

type ragRequestBody struct {
	BotID        string              `json:"bot_id"`
	SystemPrompt string              `json:"system_prompt"`
	Query        string              `json:"query"`
	History      []ragHistoryMessage `json:"history"`
}

type ragHistoryMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func NewRAGClient(cfg *config.Config) *RAGClient {
	return &RAGClient{
		baseURL:    cfg.AI.ServiceURL,
		httpClient: &http.Client{},
	}
}

func (c *RAGClient) Stream(ctx context.Context, req domain.RAGRequest) (<-chan string, error) {
	history := make([]ragHistoryMessage, 0, len(req.History))
	for _, h := range req.History {
		history = append(history, ragHistoryMessage{
			Role:    string(h.Role),
			Content: h.Content,
		})
	}

	body, err := json.Marshal(ragRequestBody{
		BotID:        req.BotID,
		SystemPrompt: req.SystemPrompt,
		Query:        req.Query,
		History:      history,
	})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx,
		http.MethodPost,
		fmt.Sprintf("%s/rag/stream", c.baseURL),
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		if len(respBody) > 0 {
			return nil, fmt.Errorf("RAG service error: %d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		}
		return nil, fmt.Errorf("RAG service error: %d", resp.StatusCode)
	}

	tokenCh := make(chan string, 100)

	go func() {
		defer close(tokenCh)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var event struct {
				Content string `json:"content"`
			}
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			select {
			case tokenCh <- event.Content:
			case <-ctx.Done():
				return
			}
		}

		if err := scanner.Err(); err != nil {
			select {
			case tokenCh <- "\n\n[stream parse error]\n":
			case <-ctx.Done():
			}
		}
	}()

	return tokenCh, nil
}
