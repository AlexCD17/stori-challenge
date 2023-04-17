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

// DefaultUserID  change user id by 'cdk.json/context/defaultUserID'.
func DefaultUserID(scope constructs.Construct) string {
	userID := "653480115121"

	ctxValue := scope.Node().TryGetContext(jsii.String("defaultUserID"))
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
