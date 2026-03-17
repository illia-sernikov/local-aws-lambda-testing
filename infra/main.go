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

const (
	defaultArtifactPath = "../build/lambda.zip"
	openAPISpecPath     = "../docs/openapi.yaml"
	region              = "us-east-1"
	localstackURL       = "http://localhost:4566"
	stageName           = "dev"
)

type apiGatewayLoggingResources struct {
	accessLogGroupName pulumi.StringOutput
}

func providerOptions(provider *aws.Provider) []pulumi.ResourceOption {
	return []pulumi.ResourceOption{pulumi.Provider(provider)}
}

func assumeRolePolicy(service string) pulumi.StringInput {
	return pulumi.String(fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Action": "sts:AssumeRole",
				"Principal": {"Service": %q},
				"Effect": "Allow"
			}
		]
	}`, service))
}

func mustConfigOrDefault(cfg *config.Config, key, fallback string) string {
	value := cfg.Get(key)
	if value == "" {
		return fallback
	}
	return value
}

func newLocalstackProvider(ctx *pulumi.Context) (*aws.Provider, error) {
	return aws.NewProvider(ctx, "localstack", &aws.ProviderArgs{
		Region:                    pulumi.String(region),
		AccessKey:                 pulumi.String("test"),
		SecretKey:                 pulumi.String("test"),
		SkipCredentialsValidation: pulumi.Bool(true),
		SkipRequestingAccountId:   pulumi.Bool(true),
		SkipMetadataApiCheck:      pulumi.Bool(true),
		S3UsePathStyle:            pulumi.Bool(true),
		Endpoints: aws.ProviderEndpointArray{
			aws.ProviderEndpointArgs{Apigateway: pulumi.String(localstackURL)},
			aws.ProviderEndpointArgs{Iam: pulumi.String(localstackURL)},
			aws.ProviderEndpointArgs{Lambda: pulumi.String(localstackURL)},
			aws.ProviderEndpointArgs{Logs: pulumi.String(localstackURL)},
			aws.ProviderEndpointArgs{Sts: pulumi.String(localstackURL)},
		},
	})
}

func configureAPIGatewayExecutionLogging(
	ctx *pulumi.Context,
	providerOpts []pulumi.ResourceOption,
	restAPI *apigateway.RestApi,
	stage *apigateway.Stage,
) error {
	apigwCloudWatchRole, err := iam.NewRole(ctx, "apigateway-cloudwatch-role", &iam.RoleArgs{
		AssumeRolePolicy: assumeRolePolicy("apigateway.amazonaws.com"),
	}, providerOpts...)
	if err != nil {
		return err
	}

	_, err = iam.NewRolePolicyAttachment(ctx, "apigateway-push-cwlogs", &iam.RolePolicyAttachmentArgs{
		Role:      apigwCloudWatchRole.Name,
		PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs"),
	}, providerOpts...)
	if err != nil {
		return err
	}

	_, err = apigateway.NewAccount(ctx, "apigateway-account", &apigateway.AccountArgs{
		CloudwatchRoleArn: apigwCloudWatchRole.Arn,
	}, providerOpts...)
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
	}, providerOpts...)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "")
		artifactPath := mustConfigOrDefault(cfg, "artifactPath", defaultArtifactPath)
		enableExecutionLogging := cfg.GetBool("enableExecutionLogging")

		openapiSpec, err := os.ReadFile(openAPISpecPath)
		if err != nil {
			return fmt.Errorf("read OpenAPI spec: %w", err)
		}

		provider, err := newLocalstackProvider(ctx)
		if err != nil {
			return err
		}
		providerOpts := providerOptions(provider)

		role, err := iam.NewRole(ctx, "lambda-exec-role", &iam.RoleArgs{
			AssumeRolePolicy: assumeRolePolicy("lambda.amazonaws.com"),
		}, providerOpts...)
		if err != nil {
			return err
		}

		_, err = iam.NewRolePolicyAttachment(ctx, "lambda-basic-exec", &iam.RolePolicyAttachmentArgs{
			Role:      role.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"),
		}, providerOpts...)
		if err != nil {
			return err
		}

		fn, err := lambda.NewFunction(ctx, "handler", &lambda.FunctionArgs{
			Name:    pulumi.String("handler"),
			Role:    role.Arn,
			Runtime: pulumi.String("provided.al2023"),
			Handler: pulumi.String("bootstrap"),
			Code:    pulumi.NewFileArchive(artifactPath),
		}, providerOpts...)
		if err != nil {
			return err
		}

		restAPI, err := apigateway.NewRestApi(ctx, "lambda-api", &apigateway.RestApiArgs{
			Body: pulumi.String(string(openapiSpec)),
		}, providerOpts...)
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "apigateway-invoke", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  fn.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:000000000000:%s/*/*", region, restAPI.ID()),
		}, providerOpts...)
		if err != nil {
			return err
		}

		deployment, err := apigateway.NewDeployment(ctx, "api-deployment", &apigateway.DeploymentArgs{
			RestApi: restAPI.ID(),
		}, providerOpts...)
		if err != nil {
			return err
		}

		stageArgs := &apigateway.StageArgs{
			RestApi:    restAPI.ID(),
			Deployment: deployment.ID(),
			StageName:  pulumi.String(stageName),
		}

		var loggingResources *apiGatewayLoggingResources
		if enableExecutionLogging {
			apigwAccessLogs, err := cloudwatch.NewLogGroup(ctx, "apigateway-access-logs", &cloudwatch.LogGroupArgs{
				Name: pulumi.Sprintf("/aws/apigateway/%s/%s/access", restAPI.ID(), stageName),
			}, providerOpts...)
			if err != nil {
				return err
			}

			stageArgs.AccessLogSettings = apigateway.StageAccessLogSettingsArgs{
				DestinationArn: apigwAccessLogs.Arn,
				Format:         pulumi.String("{\"requestId\":\"$context.requestId\",\"ip\":\"$context.identity.sourceIp\",\"requestTime\":\"$context.requestTime\",\"httpMethod\":\"$context.httpMethod\",\"routeKey\":\"$context.resourcePath\",\"status\":\"$context.status\",\"responseLength\":\"$context.responseLength\",\"integrationError\":\"$context.integrationErrorMessage\"}"),
			}

			loggingResources = &apiGatewayLoggingResources{accessLogGroupName: apigwAccessLogs.Name}
		}

		stage, err := apigateway.NewStage(ctx, "dev-stage", stageArgs, providerOpts...)
		if err != nil {
			return err
		}

		if enableExecutionLogging {
			if err := configureAPIGatewayExecutionLogging(ctx, providerOpts, restAPI, stage); err != nil {
				return err
			}
		}

		ctx.Export("apiEndpoint", pulumi.Sprintf("%s/_aws/execute-api/%s/%s", localstackURL, restAPI.ID(), stage.StageName))
		ctx.Export("healthcheckUrl", pulumi.Sprintf("%s/_aws/execute-api/%s/%s/healthcheck", localstackURL, restAPI.ID(), stage.StageName))
		ctx.Export("calculateUrl", pulumi.Sprintf("%s/_aws/execute-api/%s/%s/calculate", localstackURL, restAPI.ID(), stage.StageName))
		if enableExecutionLogging {
			ctx.Export("apiGatewayExecutionLogGroup", pulumi.Sprintf("API-Gateway-Execution-Logs_%s/%s", restAPI.ID(), stage.StageName))
			ctx.Export("apiGatewayAccessLogGroup", loggingResources.accessLogGroupName)
		}
		ctx.Export("lambdaName", fn.Name)

		fmt.Println("Pulumi deployment configured for LocalStack")
		return nil
	})
}
