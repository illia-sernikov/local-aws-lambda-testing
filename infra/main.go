package main

import (
	"fmt"
	"os"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/apigateway"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/lambda"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "")
		artifactPath := cfg.Get("artifactPath")
		if artifactPath == "" {
			artifactPath = "../build/lambda.zip"
		}

		openapiSpec, err := os.ReadFile("../docs/openapi.yaml")
		if err != nil {
			return fmt.Errorf("read OpenAPI spec: %w", err)
		}

		provider, err := aws.NewProvider(ctx, "localstack", &aws.ProviderArgs{
			Region:                    pulumi.String("us-east-1"),
			AccessKey:                 pulumi.String("test"),
			SecretKey:                 pulumi.String("test"),
			SkipCredentialsValidation: pulumi.Bool(true),
			SkipRequestingAccountId:   pulumi.Bool(true),
			SkipMetadataApiCheck:      pulumi.Bool(true),
			S3UsePathStyle:            pulumi.Bool(true),
			Endpoints: aws.ProviderEndpointArray{
				aws.ProviderEndpointArgs{Apigateway: pulumi.String("http://localhost:4566")},
				aws.ProviderEndpointArgs{Iam: pulumi.String("http://localhost:4566")},
				aws.ProviderEndpointArgs{Lambda: pulumi.String("http://localhost:4566")},
				aws.ProviderEndpointArgs{Logs: pulumi.String("http://localhost:4566")},
				aws.ProviderEndpointArgs{Sts: pulumi.String("http://localhost:4566")},
			},
		})
		if err != nil {
			return err
		}

		role, err := iam.NewRole(ctx, "lambda-exec-role", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Action": "sts:AssumeRole",
						"Principal": {"Service": "lambda.amazonaws.com"},
						"Effect": "Allow"
					}
				]
			}`),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		_, err = iam.NewRolePolicyAttachment(ctx, "lambda-basic-exec", &iam.RolePolicyAttachmentArgs{
			Role:      role.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		apigwCloudWatchRole, err := iam.NewRole(ctx, "apigateway-cloudwatch-role", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Action": "sts:AssumeRole",
						"Principal": {"Service": "apigateway.amazonaws.com"},
						"Effect": "Allow"
					}
				]
			}`),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		_, err = iam.NewRolePolicyAttachment(ctx, "apigateway-push-cwlogs", &iam.RolePolicyAttachmentArgs{
			Role:      apigwCloudWatchRole.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs"),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		_, err = apigateway.NewAccount(ctx, "apigateway-account", &apigateway.AccountArgs{
			CloudwatchRoleArn: apigwCloudWatchRole.Arn,
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		fn, err := lambda.NewFunction(ctx, "handler", &lambda.FunctionArgs{
			Name:    pulumi.String("handler"),
			Role:    role.Arn,
			Runtime: pulumi.String("provided.al2023"),
			Handler: pulumi.String("bootstrap"),
			Code:    pulumi.NewFileArchive(artifactPath),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		restAPI, err := apigateway.NewRestApi(ctx, "lambda-api", &apigateway.RestApiArgs{
			Body: pulumi.String(openapiSpec),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "apigateway-invoke", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  fn.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:us-east-1:000000000000:%s/*/*", restAPI.ID()),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		deployment, err := apigateway.NewDeployment(ctx, "api-deployment", &apigateway.DeploymentArgs{
			RestApi: restAPI.ID(),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		apigwAccessLogs, err := cloudwatch.NewLogGroup(ctx, "apigateway-access-logs", &cloudwatch.LogGroupArgs{
			Name: pulumi.Sprintf("/aws/apigateway/%s/%s/access", restAPI.ID(), pulumi.String("dev")),
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		stage, err := apigateway.NewStage(ctx, "dev-stage", &apigateway.StageArgs{
			RestApi:    restAPI.ID(),
			Deployment: deployment.ID(),
			StageName:  pulumi.String("dev"),
			AccessLogSettings: apigateway.StageAccessLogSettingsArgs{
				DestinationArn: apigwAccessLogs.Arn,
				Format:         pulumi.String("{\"requestId\":\"$context.requestId\",\"ip\":\"$context.identity.sourceIp\",\"requestTime\":\"$context.requestTime\",\"httpMethod\":\"$context.httpMethod\",\"routeKey\":\"$context.resourcePath\",\"status\":\"$context.status\",\"responseLength\":\"$context.responseLength\",\"integrationError\":\"$context.integrationErrorMessage\"}"),
			},
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		_, err = apigateway.NewMethodSettings(ctx, "api-method-settings", &apigateway.MethodSettingsArgs{
			RestApi:    restAPI.ID(),
			StageName:  stage.StageName,
			MethodPath: pulumi.String("*/*"),
			Settings: apigateway.MethodSettingsSettingsArgs{
				LoggingLevel:     pulumi.String("INFO"),
				DataTraceEnabled: pulumi.Bool(true),
				MetricsEnabled:   pulumi.Bool(true),
			},
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		ctx.Export("apiEndpoint", pulumi.Sprintf("http://localhost:4566/_aws/execute-api/%s/%s", restAPI.ID(), stage.StageName))
		ctx.Export("healthcheckUrl", pulumi.Sprintf("http://localhost:4566/_aws/execute-api/%s/%s/healthcheck", restAPI.ID(), stage.StageName))
		ctx.Export("calculateUrl", pulumi.Sprintf("http://localhost:4566/_aws/execute-api/%s/%s/calculate", restAPI.ID(), stage.StageName))
		ctx.Export("apiGatewayExecutionLogGroup", pulumi.Sprintf("API-Gateway-Execution-Logs_%s/%s", restAPI.ID(), stage.StageName))
		ctx.Export("apiGatewayAccessLogGroup", apigwAccessLogs.Name)
		ctx.Export("lambdaName", fn.Name)

		fmt.Println("Pulumi deployment configured for LocalStack")
		return nil
	})
}
