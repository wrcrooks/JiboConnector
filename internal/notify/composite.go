package notify

import (
	"context"
	"errors"

	"github.com/wrcrooks/JiboConnector/internal/jiboapi"
)

// CompositeNotifier fans a single Deliver call out to every configured
// channel, so a contact with both an email and a phone number configured is
// notified on both. Each channel is responsible for no-op'ing on a contact
// that doesn't have the field it needs (see SESNotifier/SNSNotifier).
type CompositeNotifier struct {
	channels []Notifier
}

func NewComposite(channels ...Notifier) CompositeNotifier {
	return CompositeNotifier{channels: channels}
}

func (c CompositeNotifier) Deliver(ctx context.Context, media jiboapi.Media, contact jiboapi.PhotoContact) error {
	var errs []error
	for _, channel := range c.channels {
		if err := channel.Deliver(ctx, media, contact); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
