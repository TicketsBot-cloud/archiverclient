package archiverclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type ProxyRetriever struct {
	client   *http.Client
	endpoint string
}

var _ Retriever = (*ProxyRetriever)(nil)

func NewProxyRetriever(endpoint string) *ProxyRetriever {
	return &ProxyRetriever{
		client: &http.Client{
			Timeout: time.Second * 3,
		},
		endpoint: endpoint,
	}
}

func NewProxyRetrieverWithClient(client *http.Client, endpoint string) *ProxyRetriever {
	return &ProxyRetriever{
		client:   client,
		endpoint: endpoint,
	}
}

type proxyErrorResponse struct {
	Message string `json:"message"`
}

func (r *ProxyRetriever) GetTicket(ctx context.Context, guildId uint64, ticketId int) ([]byte, error) {
	uri, err := url.Parse(r.endpoint)
	if err != nil {
		return nil, err
	}

	query := uri.Query()
	query.Set("guild", fmt.Sprintf("%d", guildId))
	query.Set("id", fmt.Sprintf("%d", ticketId))
	uri.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusOK {
		return body, nil
	} else if res.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	} else {
		var decoded proxyErrorResponse
		if err := json.Unmarshal(body, &decoded); err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("error from proxy: %s", decoded.Message)
	}
}

func (r *ProxyRetriever) StoreTicket(ctx context.Context, guildId uint64, ticketId int, data []byte) error {
	uri, err := url.Parse(r.endpoint)
	if err != nil {
		return err
	}

	query := uri.Query()
	query.Set("guild", fmt.Sprintf("%d", guildId))
	query.Set("id", fmt.Sprintf("%d", ticketId))
	uri.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri.String(), bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := r.client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		var decoded proxyErrorResponse
		if err := json.NewDecoder(res.Body).Decode(&decoded); err != nil {
			return err
		}

		return errors.New(decoded.Message)
	}

	return nil
}

func (r *ProxyRetriever) DeleteTicket(ctx context.Context, guildId uint64, ticketId int) error {
	uri, err := url.Parse(r.endpoint)
	if err != nil {
		return err
	}

	query := uri.Query()
	query.Set("guild", fmt.Sprintf("%d", guildId))
	query.Set("id", fmt.Sprintf("%d", ticketId))
	uri.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, uri.String(), nil)
	if err != nil {
		return err
	}

	res, err := r.client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return ErrNotFound
	} else if res.StatusCode != http.StatusNoContent {
		var decoded proxyErrorResponse
		if err := json.NewDecoder(res.Body).Decode(&decoded); err != nil {
			return err
		}

		return errors.New(decoded.Message)
	}

	return nil
}

func (r *ProxyRetriever) PurgeGuild(ctx context.Context, guildId uint64) error {
	uri, err := url.Parse(r.endpoint)
	if err != nil {
		return err
	}

	uri.Path = fmt.Sprintf("/guild/%d", guildId)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, uri.String(), nil)
	if err != nil {
		return err
	}

	res, err := r.client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusAccepted {
		var decoded proxyErrorResponse
		if err := json.NewDecoder(res.Body).Decode(&decoded); err != nil {
			return err
		}

		return errors.New(decoded.Message)
	}

	return nil
}

func (r *ProxyRetriever) PurgeStatus(ctx context.Context, guildId uint64) (PurgeStatus, error) {
	uri, err := url.Parse(r.endpoint)
	if err != nil {
		return PurgeStatus{}, err
	}

	uri.Path = fmt.Sprintf("/guild/status/%d", guildId)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri.String(), nil)
	if err != nil {
		return PurgeStatus{}, err
	}

	res, err := r.client.Do(req)
	if err != nil {
		return PurgeStatus{}, err
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		var status PurgeStatus
		if err := json.NewDecoder(res.Body).Decode(&status); err != nil {
			return PurgeStatus{}, err
		}

		return status, nil
	} else if res.StatusCode == http.StatusNotFound {
		return PurgeStatus{}, ErrOperationNotFound
	} else {
		var response proxyErrorResponse
		if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
			return PurgeStatus{}, err
		}

		return PurgeStatus{}, errors.New(response.Message)
	}
}
