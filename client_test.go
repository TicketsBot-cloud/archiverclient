package archiverclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TicketsBot-cloud/logarchiver/pkg/model"
	v2 "github.com/TicketsBot-cloud/logarchiver/pkg/model/v2"
)

func TestProxyDeleteTicket(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		body         string
		wantErr      string
		wantNotFound bool
	}{
		{name: "deleted", status: http.StatusNoContent},
		{name: "missing", status: http.StatusNotFound, body: `{"message":"ticket not found"}`, wantErr: ErrNotFound.Error(), wantNotFound: true},
		{name: "storage error", status: http.StatusInternalServerError, body: `{"message":"The specified bucket does not exist."}`, wantErr: "The specified bucket does not exist."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete || r.URL.Query().Get("guild") != "123" || r.URL.Query().Get("id") != "7" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			err := NewProxyRetriever(srv.URL).DeleteTicket(context.Background(), 123, 7)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("got error %v, want %q", err, tt.wantErr)
			}
			if errors.Is(err, ErrNotFound) != tt.wantNotFound || IsNotFound(err) != tt.wantNotFound {
				t.Fatalf("not found = %v/%v, want %v", errors.Is(err, ErrNotFound), IsNotFound(err), tt.wantNotFound)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{ErrNotFound, true},
		{fmt.Errorf("failed to retrieve transcript: %w", ErrNotFound), true},
		{errors.New("error from proxy: The specified key does not exist."), true},
		{errors.New("NoSuchKey: no such key"), true},
		{errors.New("The specified bucket does not exist."), false},
		{errors.New(`ERROR: relation "tickets" does not exist (SQLSTATE 42P01)`), false},
		{errors.New("context deadline exceeded"), false},
	}

	for _, tt := range tests {
		if got := IsNotFound(tt.err); got != tt.want {
			t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

type memRetriever map[string][]byte

func (m memRetriever) GetTicket(_ context.Context, guildId uint64, ticketId int) ([]byte, error) {
	data, ok := m[fmt.Sprintf("%d/%d", guildId, ticketId)]
	if !ok {
		return nil, ErrNotFound
	}
	return data, nil
}

func (m memRetriever) StoreTicket(_ context.Context, guildId uint64, ticketId int, data []byte) error {
	m[fmt.Sprintf("%d/%d", guildId, ticketId)] = data
	return nil
}

func (m memRetriever) DeleteTicket(_ context.Context, guildId uint64, ticketId int) error {
	delete(m, fmt.Sprintf("%d/%d", guildId, ticketId))
	return nil
}

func TestStoreTranscriptRoundTrip(t *testing.T) {
	store := memRetriever{}
	client := NewArchiverClient(store, bytes.Repeat([]byte{1}, 32))

	in := v2.Transcript{
		Version:  model.V2,
		Messages: []v2.Message{{Id: 1, AuthorId: 0, Content: "[removed]"}},
	}
	if err := client.StoreTranscript(context.Background(), 123, 7, in); err != nil {
		t.Fatal(err)
	}

	out, err := client.Get(context.Background(), 123, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Messages) != 1 || out.Messages[0].Content != "[removed]" || out.Messages[0].Id != 1 {
		t.Fatalf("round trip lost data: %+v", out.Messages)
	}
}
