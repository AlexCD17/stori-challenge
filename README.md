# Stori Challenge

This is a cdk app written in go that deploys a series of aws resources. 

When a CSV file is uploaded to the app s3 bucket it will trigger a lambda that parse the CSV file and do the transactions Summary, next a second lambda its called to store the summary data into a postgres RDS instance, finally a third lambda is called to generate the email and store it in the s3 bucket and optionally sends an email using SES if it was configured.


## How to

 * You need to export `CDK_DEFAULT_ACCOUNT` with your account id and  `CDK_DEFAULT_REGION` with you preferred region.
 * Notice that this app was written with the use of `IAM Identity Center` in mind, and you'll need to configure a sso session using AWS CLI.
 * Edit the `cdk.json` file with the appropriate values for your deployment. You can change params such as DBUser, DBPass, DBName, EnableSES, SenderEmail, RecipientEmail, and StackName.
 * Binaries for the lambdas are already included in the repo, if you want to modify it you should compile for linux and X64 architecture. In the lambda folder: `GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o main .`
 * Install the required dependencies: `go mod tidy`
 * `cdk deploy` will deploy this stack to your previously configured AWS Account.
 * If you want to test its functionality you can use the sample CSV under the Resources folder and upload it using AWS CLI: `aws s3 cp sample.csv s3://<name-of-your-bucket>/input/ ` note that you should get the name of the bucket from the AWS console since CF adds a UUID to the name.
 * The app will output an email html file to the output folder in the bucket, but it can send the email using SES, unfortunately there's no way to register emails on the CDK deployment, so you will need to do it manually and change the config accordingly.

