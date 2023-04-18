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
	total_balance NUMERIC(15, 2),
	transactions_by_month JSONB,
	avg_credits_by_month JSONB,
	avg_debits_by_month JSONB,
	created_at VARCHAR);`

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
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Summary Email</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            background-color: #f7f8f8;
            color: #333;
            padding: 1rem;
            max-width: 600px;
            margin: auto;
        }
        h1 {
            font-size: 1.5rem;
            margin-bottom: 1rem;
        }
        h2 {
            font-size: 1.25rem;
            margin-top: 2rem;
            margin-bottom: 1rem;
        }
        p {
            margin-bottom: 1rem;
        }
        img {
            display: block;
            max-width: 100%;
            height: auto;
            margin-bottom: 1rem;
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th,
        td {
            padding: 0.5rem;
            text-align: left;
            border: 1px solid #ccc;
        }
        th {
            background-color: #222;
            color: #fff;
        }
    </style>
</head>
<body>
    <img src="{{.LogoURL}}" alt="Logo">
    <h1>Account Summary</h1>
    <p>Total Balance: {{.TotalBalance}}</p>
    <h2>Transaction Summary</h2>
    <table>
        <thead>
            <tr>
                <th>Month</th>
                <th>Transactions</th>
                <th>Average Credit</th>
                <th>Average Debit</th>
            </tr>
        </thead>
        <tbody>
            {{range $month, $transactions := .TransactionsByMonth}}
            <tr>
                <td>{{$month}}</td>
                <td>{{$transactions}}</td>
                <td>{{index $.AvgCreditsByMonth $month}}</td>
                <td>{{index $.AvgDebitsByMonth $month}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    <h2>Total Credits and Debits</h2>
    <p>Total Credits: {{.CreditTotal}}</p>
    <p>Total Debits: {{.DebitTotal}}</p>
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
