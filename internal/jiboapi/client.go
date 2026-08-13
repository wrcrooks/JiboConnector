// Package jiboapi is a thin client for the pieces of jibo-api's protocol this
// worker needs: listing recently captured, person-tagged media, and looking
// up photo notification contacts for a recognized person.
package jiboapi

import (
	"context"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Media mirrors the fields of jibo-api's MediaRecord that this worker cares
// about (see MapMedia in JiboCloudProtocolService.cs).
type Media struct {
	Path     string `json:"path"`
	URL      string `json:"url"`
	PersonID string `json:"personId"`
}

// PhotoContact mirrors jibo-api's PhotoNotificationContactRecord.
type PhotoContact struct {
	ID          string `json:"id"`
	PersonID    string `json:"personId"`
	Name        string `json:"name"`
	PhoneNumber string `json:"phoneNumber"`
	Email       string `json:"email"`
}

// ListRecentPersonTaggedMedia is not yet implemented: jibo-api's Media_20160725
// List operation does not currently support filtering by "has a PersonId set"
// or "created since"; that filter needs to be added to jibo-api (or this
// worker needs to track a high-water mark itself) before this can be wired up.
func (c *Client) ListRecentPersonTaggedMedia(ctx context.Context) ([]Media, error) {
	panic("not implemented")
}

// PhotoContactsForPerson is not yet implemented: it will call
// GET /api/portal/loop-members/{personId}/photo-contacts once this worker's
// authentication to jibo-api's portal API is designed.
func (c *Client) PhotoContactsForPerson(ctx context.Context, personID string) ([]PhotoContact, error) {
	panic("not implemented")
}
