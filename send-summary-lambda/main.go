package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	sesTypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
)

type SummaryData struct {
	DebitTotal  float64 `json:"debit_total"`
	CreditTotal float64 `json:"credit_total"`
}

func readLogoFromS3(bucket, key string) (string, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to load configuration: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := s3Client.GetObject(context.Background(), input)
	if err != nil {
		return "", fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer result.Body.Close()

	buf := new(strings.Builder)
	encoder := base64.NewEncoder(base64.StdEncoding, buf)
	_, err = io.Copy(encoder, result.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read logo image: %w", err)
	}
	encoder.Close()

	return buf.String(), nil
}

func readEmailTemplateFromS3(bucket, key string) (string, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to load configuration: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := s3Client.GetObject(context.Background(), input)
	if err != nil {
		return "", fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer result.Body.Close()

	buf := new(strings.Builder)
	_, err = io.Copy(buf, result.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read email template: %w", err)
	}

	return buf.String(), nil
}

func getBody(templateBucket, templateKey string, summary *SummaryData) (bytes.Buffer, error) {
	templateStr, err := readEmailTemplateFromS3(templateBucket, templateKey)
	if err != nil {
		return bytes.Buffer{}, fmt.Errorf("failed to read email template from S3: %w", err)
	}

	emailTemplate, err := template.New("email").Parse(templateStr)
	if err != nil {
		return bytes.Buffer{}, fmt.Errorf("failed to parse email template: %w", err)
	}

	logoURL := "https://blog.storicard.com/wp-content/uploads/2019/07/stori.logo-horizontal-03.png"

	data := struct {
		LogoURL     string
		DebitTotal  string
		CreditTotal string
	}{
		LogoURL:     logoURL,
		DebitTotal:  strconv.FormatFloat(summary.DebitTotal, 'f', 2, 64),
		CreditTotal: strconv.FormatFloat(summary.CreditTotal, 'f', 2, 64),
	}

	// Execute the template with the data
	var emailBody bytes.Buffer
	if err = emailTemplate.Execute(&emailBody, data); err != nil {
		return bytes.Buffer{}, fmt.Errorf("failed to execute email template: %v", err)
	}

	return emailBody, nil
}

func sendEmail(emailBody bytes.Buffer, sender, recipient string) error {

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	sesClient := ses.NewFromConfig(cfg)
	input := &ses.SendEmailInput{
		Source: aws.String(sender),
		Destination: &sesTypes.Destination{
			ToAddresses: []string{recipient},
		},
		Message: &sesTypes.Message{
			Subject: &sesTypes.Content{
				Data: aws.String("Transaction Summary"),
			},
			Body: &sesTypes.Body{
				Html: &sesTypes.Content{
					Data: aws.String(emailBody.String()),
				},
			},
		},
	}

	_, err = sesClient.SendEmail(context.Background(), input)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil

}

func storeEmailOutput(bucketName, objectKey, emailBody string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to load SDK config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	input := &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   strings.NewReader(emailBody),
	}

	_, err = s3Client.PutObject(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to store email output: %w", err)
	}

	return nil
}

func handleRequest(ctx context.Context, summaryData *SummaryData) error {

	bucketName := os.Getenv("BUCKET_NAME")
	templateKey := os.Getenv("TEMPLATE_KEY")

	emailBody, err := getBody(bucketName, templateKey, summaryData)
	if err != nil {
		log.Printf("unable to get email body: %v", err)
		return err
	}

	currentTime := time.Now()
	currentTime.Format("02-01-2006")

	err = storeEmailOutput(bucketName, fmt.Sprintf("output/email-%s.html", currentTime.String()), emailBody.String())
	if err != nil {
		log.Printf("failed to store email output: %v", err)
		return err
	}

	useSES := os.Getenv("USE_SES")
	if useSES == "true" {
		sender := os.Getenv("SENDER")
		recipient := os.Getenv("RECIPIENT")
		err := sendEmail(emailBody, sender, recipient)
		if err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}

	}

	return nil

}

func main() {
	lambda.Start(handleRequest)
}