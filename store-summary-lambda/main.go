package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/lib/pq"
)

type SummaryData struct {
	DebitTotal  float64 `json:"debit_total"`
	CreditTotal float64 `json:"credit_total"`
}

func main() {
	lambda.Start(handler)
}

// handler function that stores summary data into a PostgreSQL database using a secret retrieved from AWS Secrets Manager.
func handler(ctx context.Context, summaryData *SummaryData) error {
	secretName := os.Getenv("SECRET_ARN")            // retrieve the name of the secret from an environment variable
	err := storeSummaryData(summaryData, secretName) // call function to store the summary data using the secret
	if err != nil {
		return fmt.Errorf("failed to store summary data: %v", err)
	}

	return nil // return nil to indicate success
}

func storeSummaryData(summaryData *SummaryData, secretName string) error {
	var dbParams map[string]interface{}
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	smClient := secretsmanager.NewFromConfig(cfg)
	smOutput, err := smClient.GetSecretValue(context.Background(), &secretsmanager.GetSecretValueInput{SecretId: aws.String(secretName)})
	if err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}

	err = json.Unmarshal([]byte(*smOutput.SecretString), &dbParams)
	if err != nil {
		return err
	}

	connStr := fmt.Sprintf(
		"host=%s port=%d dbname=postgres user=%s password=%s sslmode=require", // construct the connection string for the database
		dbParams["host"].(string), int(dbParams["port"].(float64)), dbParams["username"].(string), dbParams["password"].(string))

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %v", err)
	}
	defer db.Close()

	query := `
		INSERT INTO summary_records (debit_total, credit_total, created_at)
		VALUES ($1, $2, $3)
	`

	date := time.Now().Format("02-01-2006")

	res, err := db.Exec(query, summaryData.DebitTotal, summaryData.CreditTotal, date) // execute the SQL query to insert the summary data into the database
	if err != nil {
		return fmt.Errorf("failed to insert summary data into the database: %v", err)
	}

	recIns, _ := res.RowsAffected()

	fmt.Printf("Successfully inserted to db: %v rows affected", recIns)

	return nil
}
