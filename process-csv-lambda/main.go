package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	lmbda "github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"strconv"
)

type SummaryData struct {
	DebitTotal  float64 `json:"debit_total"`
	CreditTotal float64 `json:"credit_total"`
}

func readCsvFromS3(bucket, key string) (string, error) {
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
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(result.Body)

	buf := new(strings.Builder)
	_, err = io.Copy(buf, result.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read object body: %w", err)
	}

	return buf.String(), nil
}

func processCsvData(csvData string) (SummaryData, error) {
	summary := map[string]float64{
		"debit":  0,
		"credit": 0,
	}

	reader := csv.NewReader(strings.NewReader(csvData))
	// Read and ignore the header line
	if _, err := reader.Read(); err != nil {
		return SummaryData{}, fmt.Errorf("failed to read header line: %w", err)
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return SummaryData{}, fmt.Errorf("failed to read record: %w", err)
		}

		if len(record) < 4 {
			return SummaryData{}, fmt.Errorf("record has missing columns: %v", record)
		}

		typ := strings.ToLower(record[1])
		if typ != "debit" && typ != "credit" {
			return SummaryData{}, fmt.Errorf("invalid transaction type: %s", typ)
		}

		amount, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			return SummaryData{}, fmt.Errorf("failed to parse amount: %w", err)
		}

		summary[typ] += amount
	}

	return SummaryData{
		DebitTotal:  summary["debit"],
		CreditTotal: summary["credit"],
	}, nil
}

func invokeLambda(ctx context.Context, summaryData *SummaryData, lambdaName string) error {
	lambdaClient := lmbda.New(lmbda.Options{Region: "us-east-1"})

	data, err := json.Marshal(summaryData)
	if err != nil {
		return fmt.Errorf("failed to marshal summary data: %v", err)
	}

	input := &lmbda.InvokeInput{
		FunctionName:   aws.String(lambdaName),
		InvocationType: types.InvocationType("Event"),
		Payload:        data,
	}

	_, err = lambdaClient.Invoke(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to invoke %s: %v", lambdaName, err)
	}

	return nil
}

func handler(ctx context.Context, s3Event events.S3Event) error {

	for _, record := range s3Event.Records {
		s3Entity := record.S3
		bucket := s3Entity.Bucket.Name
		key := s3Entity.Object.Key

		csvData, err := readCsvFromS3(bucket, key)
		if err != nil {
			return fmt.Errorf("failed to read CSV from S3: %w", err)
		}

		summary, err := processCsvData(csvData)
		if err != nil {
			return fmt.Errorf("failed to process CSV data: %w", err)
		}

		err = invokeLambda(ctx, &summary, os.Getenv("SEND_ARN"))
		if err != nil {
			return err
		}

		err = invokeLambda(ctx, &summary, os.Getenv("STORE_ARN"))
		if err != nil {
			return err
		}

	}

	return nil
}

func main() {
	lambda.Start(handler)
}
