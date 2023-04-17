package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsrds"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3assets"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3notifications"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssecretsmanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/customresources"

	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"stori-challenge/config"
)

type StoriChallengeStackProps struct {
	awscdk.StackProps
}

func NewStoriChallengeStack(scope constructs.Construct, id string, props *StoriChallengeStackProps) awscdk.Stack {
	bucketName := jsii.String("storiChallenge-bucket")
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// Create a VPC for the RDS instance and Lambda functions
	vpc := awsec2.NewVpc(stack, jsii.String("StoriVPC"), &awsec2.VpcProps{
		MaxAzs: jsii.Number(2),
	})

	rdsSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String("RdsSecurityGroup"), &awsec2.SecurityGroupProps{
		Vpc: vpc,
	})

	// Create an S3 bucket
	bucket := awss3.NewBucket(stack, bucketName, &awss3.BucketProps{
		Versioned: jsii.Bool(false),
	})

	// Secret for db details
	rdsSecret := awssecretsmanager.NewSecret(stack, jsii.String("StoriRdsInstanceSecret"), &awssecretsmanager.SecretProps{
		SecretObjectValue: &map[string]awscdk.SecretValue{
			"username": awscdk.SecretValue_UnsafePlainText(jsii.String("adminStori")),
			"password": awscdk.SecretValue_UnsafePlainText(jsii.String("adminPass")),
		},
	})

	// Create an RDS PostgreSQL instance
	awsrds.NewDatabaseInstance(stack, jsii.String("StoriRdsInstance"), &awsrds.DatabaseInstanceProps{
		Engine: awsrds.DatabaseInstanceEngine_Postgres(&awsrds.PostgresInstanceEngineProps{
			Version: awsrds.PostgresEngineVersion_VER_15_2(),
		}),
		InstanceType: awsec2.NewInstanceType(jsii.String("t3.micro")),
		Vpc:          vpc,
		Credentials:  awsrds.Credentials_FromSecret(rdsSecret, jsii.String("adminStori")),
		SecurityGroups: &[]awsec2.ISecurityGroup{
			rdsSecurityGroup,
		},
		DatabaseName:       jsii.String(config.DBName(stack)),
		PubliclyAccessible: jsii.Bool(true),
		DeletionProtection: jsii.Bool(false),
		RemovalPolicy:      awscdk.RemovalPolicy_DESTROY,
	})

	// Create the init-lambda function
	initLambda := awslambda.NewFunction(stack, jsii.String("InitLambda"), &awslambda.FunctionProps{
		Runtime: awslambda.Runtime_GO_1_X(),
		Code:    awslambda.Code_FromAsset(jsii.String("init-lambda"), nil),
		Handler: jsii.String("main"),
		Environment: &map[string]*string{
			"SECRET_ARN":  rdsSecret.SecretArn(),
			"BUCKET_NAME": bucket.BucketName(),
		},
		Vpc:               vpc,
		AllowPublicSubnet: jsii.Bool(true),
	})

	// Create the process-csv-lambda function
	sendSummaryLambda := awslambda.NewFunction(stack, jsii.String("SendSummaryLambda"), &awslambda.FunctionProps{
		Runtime: awslambda.Runtime_GO_1_X(),
		Code:    awslambda.Code_FromAsset(jsii.String("send-summary-lambda"), nil),
		Handler: jsii.String("main"),
		Environment: &map[string]*string{
			"BUCKET_NAME":  bucket.BucketName(),
			"TEMPLATE_KEY": jsii.String("email_template.html"),
			"USE_SES":      jsii.String(config.EnableSES(stack)),
			"SENDER":       jsii.String("alex.contredel@gmail.com"),
			"RECIPIENT":    jsii.String("alexcondel17@gmail.com"),
		},
		Vpc: vpc,
	})

	// Create store-summary-lambda
	storeSummaryLambda := awslambda.NewFunction(stack, jsii.String("StoreSummaryLambda"), &awslambda.FunctionProps{
		Runtime:      awslambda.Runtime_GO_1_X(),
		Code:         awslambda.Code_FromAsset(jsii.String("store-summary-lambda"), &awss3assets.AssetOptions{}),
		Handler:      jsii.String("main"),
		Timeout:      awscdk.Duration_Seconds(jsii.Number(30)),
		LogRetention: awslogs.RetentionDays_ONE_WEEK,
		Environment: &map[string]*string{
			"SECRET_ARN": rdsSecret.SecretArn(),
		},
		Vpc: vpc,
	})

	// Create the process-csv-lambda function
	processCsvLambda := awslambda.NewFunction(stack, jsii.String("ProcessCsvLambda"), &awslambda.FunctionProps{
		Runtime: awslambda.Runtime_GO_1_X(),
		Code:    awslambda.Code_FromAsset(jsii.String("process-csv-lambda"), nil),
		Handler: jsii.String("main"),
		Environment: &map[string]*string{
			"SEND_ARN":  sendSummaryLambda.FunctionArn(),
			"STORE_ARN": storeSummaryLambda.FunctionArn(),
		},
		AllowPublicSubnet: jsii.Bool(true),
		Vpc:               vpc,
	})

	initLambda.Connections().AllowTo(rdsSecurityGroup, awsec2.Port_Tcp(jsii.Number(5432)), jsii.String("Allow Lambda to access RDS instance"))
	storeSummaryLambda.Connections().AllowTo(rdsSecurityGroup, awsec2.Port_Tcp(jsii.Number(5432)), jsii.String("Allow Lambda to access RDS instance"))

	// Attach the IAM policy to the init-lambda function's execution role
	bucket.GrantReadWrite(initLambda, "*")

	bucket.GrantRead(processCsvLambda, "*")

	bucket.GrantReadWrite(sendSummaryLambda, "*")

	rdsSecret.GrantRead(initLambda, nil)

	rdsSecret.GrantRead(storeSummaryLambda, nil)

	// Attach the IAM policy to the process-csv-lambda function's execution role
	bucket.GrantPut(initLambda, "*")

	// Grant permission for process-csv-lambda to invoke store-summary-lambda
	storeSummaryLambda.GrantInvoke(processCsvLambda)

	// Grant permission for process-csv-lambda to invoke send-summary-lambda
	sendSummaryLambda.GrantInvoke(processCsvLambda)
	//processCsvLambda.GrantInvoke(sendSummaryLambda)

	// Configure the security group to allow connections between Lambda functions and RDS instance
	lambdaSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String("LambdaSecurityGroup"), &awsec2.SecurityGroupProps{
		Vpc: vpc,
	})
	processCsvLambda.Connections().AddSecurityGroup(lambdaSecurityGroup)
	sendSummaryLambda.Connections().AddSecurityGroup(lambdaSecurityGroup)
	storeSummaryLambda.Connections().AddSecurityGroup(lambdaSecurityGroup)
	initLambda.Connections().AddSecurityGroup(lambdaSecurityGroup)

	lambdaSecurityGroup.AddIngressRule(awsec2.Peer_SecurityGroupId(rdsSecurityGroup.SecurityGroupId(), stack.Account()),
		awsec2.Port_Tcp(jsii.Number(5432)), jsii.String("Allow Lambda to access RDS instance"), jsii.Bool(false))

	// Grant send email permission to the Lambda function
	sendSummaryLambda.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   &[]*string{jsii.String("ses:SendEmail"), jsii.String("ses:SendRawEmail")},
		Resources: &[]*string{jsii.String("*")},
	}))

	// Create the custom resource to trigger the init Lambda
	initTrigger := customresources.NewAwsCustomResource(stack, jsii.String("InitLambdaTrigger"), &customresources.AwsCustomResourceProps{
		OnCreate: &customresources.AwsSdkCall{
			Service: jsii.String("Lambda"),
			Action:  jsii.String("invoke"),
			Parameters: map[string]interface{}{
				"FunctionName": initLambda.FunctionArn(),
			},
			PhysicalResourceId: customresources.PhysicalResourceId_Of(jsii.String("init-lambda")),
		},
		Policy: customresources.AwsCustomResourcePolicy_FromStatements(&[]awsiam.PolicyStatement{awsiam.NewPolicyStatement(
			&awsiam.PolicyStatementProps{
				Actions:   &[]*string{jsii.String("lambda:InvokeFunction")},
				Resources: &[]*string{initLambda.FunctionArn()},
			})}),
	})

	// Add rds instance secret as custom resource dependency
	initTrigger.Node().AddDependency(rdsSecret)

	// Configure the S3 bucket to trigger the process CSV Lambda when a file is uploaded
	bucket.AddEventNotification(
		awss3.EventType_OBJECT_CREATED_PUT,
		awss3notifications.NewLambdaDestination(processCsvLambda),
		&awss3.NotificationKeyFilter{
			Prefix: jsii.String("input/"),
		},
	)

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	NewStoriChallengeStack(app, config.StackName(app), &StoriChallengeStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	// If unspecified, this stack will be "environment-agnostic".
	// Account/Region-dependent features and context lookups will not work, but a
	// single synthesized template can be deployed anywhere.
	//---------------------------------------------------------------------------
	return &awscdk.Environment{
		Account: jsii.String("653480115121"), // Replace with your AWS account ID
		Region:  jsii.String("us-east-1"),    // Replace with your desired AWS region
	}

	// Uncomment if you know exactly what account and region you want to deploy
	// the stack to. This is the recommendation for production stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String("123456789012"),
	//  Region:  jsii.String("us-east-1"),
	// }

	// Uncomment to specialize this stack for the AWS Account and Region that are
	// implied by the current CLI configuration. This is recommended for dev
	// stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
	//  Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	// }
}
