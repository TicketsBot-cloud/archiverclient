package archiverclient

import (
	"context"
	"errors"

	"github.com/TicketsBot-cloud/logarchiver/pkg/s3client"
)

type S3Retriever struct {
	client *s3client.S3Client
}

var _ Retriever = (*S3Retriever)(nil)

func NewS3Retriever(client *s3client.S3Client) *S3Retriever {
	return &S3Retriever{client: client}
}

func (r *S3Retriever) GetTicket(ctx context.Context, guildId uint64, ticketId int) ([]byte, error) {
	res, err := r.client.GetTicket(ctx, guildId, ticketId)
	if err != nil && errors.Is(err, s3client.ErrTicketNotFound) {
		return nil, ErrNotFound
	}

	return res, err
}

func (r *S3Retriever) StoreTicket(ctx context.Context, guildId uint64, ticketId int, data []byte) error {
	return r.client.StoreTicket(ctx, guildId, ticketId, data)
}

func (r *S3Retriever) DeleteTicket(ctx context.Context, guildId uint64, ticketId int) error {
	return r.client.DeleteTicket(ctx, guildId, ticketId)
}
