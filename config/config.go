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

// DBUser   change user id by 'cdk.json/context/dbUser'.
func DBUser(scope constructs.Construct) string {
	userID := "adminStori"

	ctxValue := scope.Node().TryGetContext(jsii.String("dbUser"))
	if v, ok := ctxValue.(string); ok {
		userID = v
	}

	return userID
}

// DBPass   change user id by 'cdk.json/context/dbPass'.
func DBPass(scope constructs.Construct) string {
	userID := "adminPass"

	ctxValue := scope.Node().TryGetContext(jsii.String("dbPass"))
	if v, ok := ctxValue.(string); ok {
		userID = v
	}

	return userID
}

// SenderEmail change user id by 'cdk.json/context/senderEmail'.
func SenderEmail(scope constructs.Construct) string {
	userID := "someemail@email.com"

	ctxValue := scope.Node().TryGetContext(jsii.String("senderEmail"))
	if v, ok := ctxValue.(string); ok {
		userID = v
	}

	return userID
}

// RecipientEmail   change user id by 'cdk.json/context/recipientEmail'.
func RecipientEmail(scope constructs.Construct) string {
	userID := "recipient@some.com"

	ctxValue := scope.Node().TryGetContext(jsii.String("recipientEmail"))
	if v, ok := ctxValue.(string); ok {
		userID = v
	}

	return userID
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
