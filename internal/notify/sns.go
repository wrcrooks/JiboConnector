package notify

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"github.com/wrcrooks/JiboConnector/internal/jiboapi"
)

// SNSNotifier delivers photo notifications by SMS via Amazon SNS. It does
// not send email — see SESNotifier for that — so Deliver is a no-op for
// contacts that only have an email address configured.
type SNSNotifier struct {
	client *sns.Client
}

// NewSNSNotifier loads AWS credentials via the SDK's standard chain (env
// vars, shared config/credentials files, or an IAM role), same as
// NewSESNotifier — this package deliberately does not invent its own
// credential handling.
func NewSNSNotifier(ctx context.Context, region string) (*SNSNotifier, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	return &SNSNotifier{client: sns.NewFromConfig(awsCfg)}, nil
}

func (n *SNSNotifier) Deliver(ctx context.Context, media jiboapi.Media, contact jiboapi.PhotoContact) error {
	if contact.PhoneNumber == "" {
		return nil
	}

	// SNS requires E.164 (a leading "+" and country code). The portal API
	// that stores PhotoNotificationContactRecord.PhoneNumber doesn't
	// validate the format, so a malformed number is a real, expected
	// failure mode here — surface it as an error (the poll loop already
	// logs Deliver errors) rather than silently dropping the notification.
	if !strings.HasPrefix(contact.PhoneNumber, "+") {
		return fmt.Errorf("phone number %q is not in E.164 format (must start with +)", contact.PhoneNumber)
	}

	message := fmt.Sprintf("Jibo took a new photo! View it here: %s", media.URL)
	_, err := n.client.Publish(ctx, &sns.PublishInput{
		PhoneNumber: aws.String(contact.PhoneNumber),
		Message:     aws.String(message),
	})
	if err != nil {
		return fmt.Errorf("sending SNS SMS to %s: %w", contact.PhoneNumber, err)
	}
	return nil
}
