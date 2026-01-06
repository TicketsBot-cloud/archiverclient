package archiverclient

import (
	"context"
	"errors"
)

type PurgingClient struct {
	*ArchiverClient
	retriever *ProxyRetriever
}

type PurgeStatus struct {
	Status  Status            `json:"status"`
	Removed []string          `json:"removed"`
	Failed  []string          `json:"failed"`
	Errors  map[string]string `json:"errors"`
}

type Status string

const (
	StatusInProgress Status = "in_progress"
	StatusComplete   Status = "complete"
	StatusFailed     Status = "failed"
	StatusTimeout    Status = "timeout"
)

var ErrOperationNotFound = errors.New("operation not found")

func NewPurgingClient(retriever *ProxyRetriever, encryptionKey []byte) *PurgingClient {
	return &PurgingClient{
		ArchiverClient: NewArchiverClient(retriever, encryptionKey),
		retriever:      retriever,
	}
}

func (c *PurgingClient) PurgeGuild(ctx context.Context, guildId uint64) error {
	return c.retriever.PurgeGuild(ctx, guildId)
}

func (c *PurgingClient) PurgeStatus(ctx context.Context, guildId uint64) (PurgeStatus, error) {
	return c.retriever.PurgeStatus(ctx, guildId)
}
