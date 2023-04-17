package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	_ "github.com/lib/pq"
)

// Add the initializeDB function
func initializeDB(ctx context.Context, secretName string) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	smClient := secretsmanager.NewFromConfig(cfg)
	smOutput, err := smClient.GetSecretValue(context.Background(), &secretsmanager.GetSecretValueInput{SecretId: aws.String(secretName)})
	if err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}

	var dbParams map[string]interface{}

	err = json.Unmarshal([]byte(*smOutput.SecretString), &dbParams)
	if err != nil {
		return err
	}

	connStr := fmt.Sprintf(
		"host=%s port=%d dbname=postgres user=%s password=%s sslmode=require",
		dbParams["host"].(string), int(dbParams["port"].(float64)), dbParams["username"].(string), dbParams["password"].(string))

	fmt.Printf("%v", connStr)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %v", err)
	}

	defer db.Close()

	err = db.Ping()
	if err != nil {
		return err
	}

	createTableQuery := `CREATE TABLE IF NOT EXISTS summary_records (
		id SERIAL PRIMARY KEY,
		debit_total NUMERIC(15, 2),
		credit_total NUMERIC(15, 2),
		created_at DATE
	);`

	_, err = db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create summary_records table: %w", err)
	}

	return nil
}

func uploadEmailTemplate(bucket string, key string) error {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	emailTemplate := `
	<!DOCTYPE html>
<html>
<head>
    <title>Transaction Summary</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 0;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .logo {
            max-width: 150px;
            margin-bottom: 20px;
        }
        .summary {
            border-collapse: collapse;
            width: 100%;
            margin-bottom: 20px;
        }
        .summary th, .summary td {
            border: 1px solid #dddddd;
            text-align: left;
            padding: 8px;
        }
        .summary th {
            background-color: #f2f2f2;
        }
    </style>
</head>
<body>
    <div class="container">
        <img class="logo" src="{{.LogoURL}}" alt="Logo">
        <h1>Transaction Summary</h1>
        <table class="summary">
            <thead>
                <tr>
                    <th>Transaction Type</th>
                    <th>Total Amount</th>
                </tr>
            </thead>
            <tbody>
                <tr>
                    <td>Debit</td>
                    <td>{{.DebitTotal}}</td>
                </tr>
                <tr>
                    <td>Credit</td>
                    <td>{{.CreditTotal}}</td>
                </tr>
            </tbody>
        </table>
    </div>
</body>
</html>
`

	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   strings.NewReader(emailTemplate),
	}

	_, err = s3Client.PutObject(context.Background(), input)
	if err != nil {
		return fmt.Errorf("failed to upload email template: %w", err)
	}

	return nil
}

func handleRequest(ctx context.Context, s3Event events.S3Event) error {

	// Initialize the database with the summary table
	dbSecretName := os.Getenv("SECRET_ARN")
	err := initializeDB(ctx, dbSecretName)
	if err != nil {
		return fmt.Errorf("failed to initialize the database: %w", err)
	}

	bucket := os.Getenv("BUCKET_NAME")
	key := "email_template.html"

	err = uploadEmailTemplate(bucket, key)
	if err != nil {
		return fmt.Errorf("failed to upload email template: %w", err)
	}

	return nil
}

func main() {
	lambda.Start(handleRequest)
}
