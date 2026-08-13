package notify

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"

	"github.com/wrcrooks/JiboConnector/internal/jiboapi"
)

// SESNotifier delivers photo notifications by email via Amazon SES. It does
// not send SMS — no SMS provider has been chosen yet (see README) — so
// Deliver is a no-op for contacts that only have a phone number configured.
type SESNotifier struct {
	client      *sesv2.Client
	fromAddress string
}

// NewSESNotifier loads AWS credentials via the SDK's standard chain (env
// vars, shared config/credentials files, or an IAM role) — this package
// deliberately does not invent its own credential handling.
func NewSESNotifier(ctx context.Context, region, fromAddress string) (*SESNotifier, error) {
	if fromAddress == "" {
		return nil, fmt.Errorf("fromAddress is required")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	return &SESNotifier{
		client:      sesv2.NewFromConfig(awsCfg),
		fromAddress: fromAddress,
	}, nil
}

func (n *SESNotifier) Deliver(ctx context.Context, media jiboapi.Media, contact jiboapi.PhotoContact) error {
	if contact.Email == "" {
		return nil
	}

	// jibo-api's connector endpoints only expose the recognized person's id
	// (a LoopMemberRecord.Id), not a display name, so the subject stays
	// generic rather than fabricating one.
	subject := "Jibo took a new photo!"
	textBody := fmt.Sprintf("Jibo took a photo. View it here: %s", media.URL)
	htmlBody := fmt.Sprintf(
		`<p>Jibo took a photo.</p><p><a href="%s">View the photo</a></p>`,
		media.URL)

	_, err := n.client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(n.fromAddress),
		Destination: &types.Destination{
			ToAddresses: []string{contact.Email},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(subject)},
				Body: &types.Body{
					Text: &types.Content{Data: aws.String(textBody)},
					Html: &types.Content{Data: aws.String(htmlBody)},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("sending SES email to %s: %w", contact.Email, err)
	}
	return nil
}
