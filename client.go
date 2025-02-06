package archiverclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/TicketsBot/common/encryption"
	"github.com/TicketsBot/logarchiver/pkg/model"
	v1 "github.com/TicketsBot/logarchiver/pkg/model/v1"
	v2 "github.com/TicketsBot/logarchiver/pkg/model/v2"
	"github.com/rxdn/gdl/objects/channel"
	"github.com/rxdn/gdl/objects/channel/message"
	"github.com/rxdn/gdl/objects/guild"
	"github.com/rxdn/gdl/objects/user"
)

type ArchiverClient struct {
	retriever Retriever
	key       []byte
}

var ErrNotFound = errors.New("Transcript not found")

func NewArchiverClient(retriever Retriever, encryptionKey []byte) *ArchiverClient {
	return &ArchiverClient{
		retriever: retriever,
		key:       encryptionKey,
	}
}

func (c *ArchiverClient) Get(ctx context.Context, guildId uint64, ticketId int) (v2.Transcript, error) {
	body, err := c.retriever.GetTicket(ctx, guildId, ticketId)
	if err != nil {
		return v2.Transcript{}, err
	}

	body, err = encryption.Decompress(body)
	if err != nil {
		return v2.Transcript{}, err
	}

	body, err = encryption.Decrypt(c.key, body)
	if err != nil {
		return v2.Transcript{}, err
	}

	version := model.GetVersion(body)
	switch version {
	case model.V1:
		var messages []message.Message
		if err := json.Unmarshal(body, &messages); err != nil {
			return v2.Transcript{}, err
		}

		return v1.ConvertToV2(messages), nil
	case model.V2:
		var transcript v2.Transcript
		if err := json.Unmarshal(body, &transcript); err != nil {
			return v2.Transcript{}, err
		}

		return transcript, nil
	default:
		return v2.Transcript{}, fmt.Errorf("unknown version %d", version)
	}
}

func (c *ArchiverClient) Store(ctx context.Context, guildId uint64, ticketId int, messages []message.Message) error {
	transcript := v2.NewTranscript(messages, v2.NoopRetriever[user.User], v2.NoopRetriever[channel.Channel], v2.NoopRetriever[guild.Role])

	data, err := json.Marshal(transcript)
	if err != nil {
		return err
	}

	data, err = encryption.Encrypt(c.key, data)
	if err != nil {
		return err
	}

	data = encryption.Compress(data)

	return c.retriever.StoreTicket(ctx, guildId, ticketId, data)
}

func (c *ArchiverClient) ImportTranscript(ctx context.Context, guildId uint64, ticketId int, data []byte) error {
	data, err := encryption.Encrypt(c.key, data)
	if err != nil {
		return err
	}

	data = encryption.Compress(data)
	return c.retriever.StoreTicket(ctx, guildId, ticketId, data)
}
