package main

import (
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
	"github.com/stretchr/testify/require"
)

func TestStoriChallengeStack(t *testing.T) {
	// Create a new app for testing
	app := awscdk.NewApp(nil)

	// Create a new StoriChallengeStack with the app
	stack := NewStoriChallengeStack(app, "TestStack", nil)

	// Test if resources exist in the stack
	bucket := stack.Node().TryFindChild(jsii.String("storiChallenge-bucket"))
	require.NotNil(t, bucket, "S3 bucket not found in stack")

	rdsInstance := stack.Node().TryFindChild(jsii.String("StoriRdsInstance"))
	require.NotNil(t, rdsInstance, "RDS instance not found in stack")

	initLambda := stack.Node().TryFindChild(jsii.String("InitLambda"))
	require.NotNil(t, initLambda, "InitLambda not found in stack")

	sendSummaryLambda := stack.Node().TryFindChild(jsii.String("SendSummaryLambda"))
	require.NotNil(t, sendSummaryLambda, "SendSummaryLambda not found in stack")

	storeSummaryLambda := stack.Node().TryFindChild(jsii.String("StoreSummaryLambda"))
	require.NotNil(t, storeSummaryLambda, "StoreSummaryLambda not found in stack")

	processCsvLambda := stack.Node().TryFindChild(jsii.String("ProcessCsvLambda"))
	require.NotNil(t, processCsvLambda, "ProcessCsvLambda not found in stack")
}
