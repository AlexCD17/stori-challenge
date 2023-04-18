package config

import (
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"strconv"
)

// EnableSES set SES by 'cdk.json/context/enableSES'.
func EnableSES(scope constructs.Construct) string {
	enableSes := false

	ctxValue := scope.Node().TryGetContext(jsii.String("enableSES"))
	if v, ok := ctxValue.(bool); ok {
		enableSes = v
	}

	return strconv.FormatBool(enableSes)
}

// StackName change stack name by 'cdk.json/context/stackName'.
func StackName(scope constructs.Construct) string {
	stackName := "StoriChallengeStack"

	ctxValue := scope.Node().TryGetContext(jsii.String("stackName"))
	if v, ok := ctxValue.(string); ok {
		stackName = v
	}

	return stackName
}

// DBUser DBUser  change user id by 'cdk.json/context/dbUser'.
func DBUser(scope constructs.Construct) string {
	userID := "adminStori"

	ctxValue := scope.Node().TryGetContext(jsii.String("dbUser"))
	if v, ok := ctxValue.(string); ok {
		userID = v
	}

	return userID
}

// DBPass DBUser  change user id by 'cdk.json/context/dbPass'.
func DBPass(scope constructs.Construct) string {
	userID := "adminPass"

	ctxValue := scope.Node().TryGetContext(jsii.String("dbPass"))
	if v, ok := ctxValue.(string); ok {
		userID = v
	}

	return userID
}

// DefaultRegion change region by 'cdk.json/context/defaultRegion'.
func DefaultRegion(scope constructs.Construct) string {
	defaultRegion := "us-east-1"

	ctxValue := scope.Node().TryGetContext(jsii.String("defaultRegion"))
	if v, ok := ctxValue.(string); ok {
		defaultRegion = v
	}

	return defaultRegion
}

// DBName change region by 'cdk.json/context/dbName'.
func DBName(scope constructs.Construct) string {
	dbName := "postgres"

	ctxValue := scope.Node().TryGetContext(jsii.String("dbName"))
	if v, ok := ctxValue.(string); ok {
		dbName = v
	}

	return dbName
}
