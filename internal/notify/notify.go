// Package notify defines how JiboConnector delivers a photo to a contact.
// No concrete provider (Twilio, SendGrid, plain SMTP, ...) is chosen yet;
// NoopNotifier exists only so the rest of the worker has something to build
// and run against in the meantime.
package notify

import (
	"context"
	"log/slog"

	"github.com/wrcrooks/JiboConnector/internal/jiboapi"
)

type Notifier interface {
	// Deliver sends the given media (a photo of the recognized person) to a
	// single contact, by whichever channel(s) the contact has configured.
	Deliver(ctx context.Context, media jiboapi.Media, contact jiboapi.PhotoContact) error
}

// NoopNotifier logs what it would have sent instead of actually sending
// anything. Placeholder until a real SMS/email provider is chosen.
type NoopNotifier struct {
	Logger *slog.Logger
}

func (n NoopNotifier) Deliver(_ context.Context, media jiboapi.Media, contact jiboapi.PhotoContact) error {
	n.Logger.Info("would deliver photo",
		"mediaPath", media.Path,
		"personId", media.PersonID,
		"contactName", contact.Name,
		"hasPhone", contact.PhoneNumber != "",
		"hasEmail", contact.Email != "")
	return nil
}
