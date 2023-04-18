package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"io"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	lmbda "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"strconv"
)

type SummaryData struct {
	TotalBalance        float64
	TransactionsByMonth map[string]int
	AvgCreditsByMonth   map[string]float64
	AvgDebitsByMonth    map[string]float64
	DebitTotal          float64
	CreditTotal         float64
}

// readCsvFromS3 reads a CSV file from S3 and returns its contents as a string.
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

// processCsvData processes the CSV data and returns a SummaryData struct containing the total debit and credit amounts.
func processCsvData(csvData string) (SummaryData, error) {
	var debitTotal float64
	var creditTotal float64
	monthTransactions := make(map[string]int)
	monthCredits := make(map[string]float64)
	monthDebits := make(map[string]float64)

	reader := csv.NewReader(strings.NewReader(csvData))
	// Read and ignore the header line
	if _, err := reader.Read(); err != nil {
		return SummaryData{}, fmt.Errorf("failed to read header line: %w", err)
	}

	// Process each record in the CSV file
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return SummaryData{}, fmt.Errorf("failed to read record: %w", err)
		}

		// Check that the record has the required columns
		if len(record) < 4 {
			return SummaryData{}, fmt.Errorf("record has missing columns: %v", record)
		}

		// Get the transaction type (debit or credit)
		typ := strings.ToLower(record[1])
		if typ != "debit" && typ != "credit" {
			return SummaryData{}, fmt.Errorf("invalid transaction type: %s", typ)
		}

		// Get the transaction amount
		amount, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			return SummaryData{}, fmt.Errorf("failed to parse amount: %w", err)
		}

		month := record[3][:7] // Extract year-month from the date
		monthTransactions[month]++
		if typ == "credit" {
			creditTotal += amount
			monthCredits[month] += amount
		} else if typ == "debit" {
			debitTotal += amount
			monthDebits[month] += amount
		}

	}

	return SummaryData{
		DebitTotal:          debitTotal,
		CreditTotal:         creditTotal,
		TotalBalance:        creditTotal + debitTotal,
		TransactionsByMonth: monthTransactions,
		AvgCreditsByMonth:   monthCredits,
		AvgDebitsByMonth:    monthDebits,
	}, nil
}

// Invokes lambdas for next steps
func invokeLambda(ctx context.Context, summaryData *SummaryData, lambdaName string) error {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("Failed to load SDK configuration: %v", err)
	}

	lambdaClient := lambda.NewFromConfig(cfg)

	data, err := json.Marshal(summaryData)
	if err != nil {
		return fmt.Errorf("failed to marshal summary data: %v", err)
	}

	input := &lambda.InvokeInput{
		FunctionName: aws.String(lambdaName),
		Payload:      data,
	}

	_, err = lambdaClient.Invoke(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to invoke %s: %v", lambdaName, err)
	}

	return nil
}

// This function is the main entry point for the Lambda function. It takes in an S3 event, reads and processes
// the corresponding CSV file, and invokes two separate Lambda functions with the resulting summary data.
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

		// Store records
		err = invokeLambda(ctx, &summary, os.Getenv("STORE_ARN"))
		if err != nil {
			return err
		}

		// Send email
		err = invokeLambda(ctx, &summary, os.Getenv("SEND_ARN"))
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	lmbda.Start(handler)
}
