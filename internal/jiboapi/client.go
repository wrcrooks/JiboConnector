// Package jiboapi is a thin client for jibo-api's /api/connector/* endpoints:
// listing recently captured, person-tagged media, and looking up photo
// notification contacts for a recognized person. Authenticated with a
// shared-secret bearer token (see jibo-api's ConnectorEndpoints.cs).
package jiboapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Media mirrors the fields jibo-api's /api/connector/media returns (see
// BuildConnectorMediaPayload in ConnectorEndpoints.cs).
type Media struct {
	Path        string `json:"path"`
	URL         string `json:"url"`
	PersonID    string `json:"personId"`
	CreatedUnix int64  `json:"createdUnixMs"`
}

// PhotoContact mirrors jibo-api's PhotoNotificationContactRecord as returned
// by /api/connector/loop-members/{personId}/photo-contacts.
type PhotoContact struct {
	ID          string `json:"id"`
	PersonID    string `json:"personId"`
	Name        string `json:"name"`
	PhoneNumber string `json:"phoneNumber"`
	Email       string `json:"email"`
}

// ListRecentPersonTaggedMedia returns person-tagged media created strictly
// after sinceUnixMs (pass 0 for "everything"). jibo-api's endpoint already
// filters out media with no PersonId set.
func (c *Client) ListRecentPersonTaggedMedia(ctx context.Context, sinceUnixMs int64) ([]Media, error) {
	endpoint := c.baseURL + "/api/connector/media"
	if sinceUnixMs > 0 {
		endpoint += "?sinceUnixMs=" + url.QueryEscape(strconv.FormatInt(sinceUnixMs, 10))
	}

	var decoded struct {
		Media []Media `json:"media"`
	}
	if err := c.getJSON(ctx, endpoint, &decoded); err != nil {
		return nil, fmt.Errorf("listing recent person-tagged media: %w", err)
	}
	return decoded.Media, nil
}

// PhotoContactsForPerson returns the notification contacts configured for a
// recognized person (a jibo-api LoopMemberRecord id).
func (c *Client) PhotoContactsForPerson(ctx context.Context, personID string) ([]PhotoContact, error) {
	endpoint := c.baseURL + "/api/connector/loop-members/" + url.PathEscape(personID) + "/photo-contacts"

	var decoded struct {
		Contacts []PhotoContact `json:"contacts"`
	}
	if err := c.getJSON(ctx, endpoint, &decoded); err != nil {
		return nil, fmt.Errorf("looking up photo contacts for person %q: %w", personID, err)
	}
	return decoded.Contacts, nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("calling %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned status %d", endpoint, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding response from %s: %w", endpoint, err)
	}
	return nil
}
