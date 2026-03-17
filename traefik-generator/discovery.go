package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	apigw "github.com/aws/aws-sdk-go-v2/service/apigateway"
)

func DiscoverAPIs() ([]ApiInfo, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithBaseEndpoint("http://localstack:4566"),
	)
	if err != nil {
		return nil, err
	}

	restClient := apigw.NewFromConfig(cfg)

	apis := []ApiInfo{}
	errs := []string{}

	rest, err := restClient.GetRestApis(context.TODO(), &apigw.GetRestApisInput{})
	if err != nil {
		errs = append(errs, fmt.Sprintf("apigateway get-rest-apis failed: %v", err))
	} else {
		for _, api := range rest.Items {

			stages, stageErr := restClient.GetStages(
				context.TODO(),
				&apigw.GetStagesInput{
					RestApiId: api.Id,
				},
			)
			if stageErr != nil {
				log.Printf("skip REST API %s (%s): get-stages failed: %v", aws.ToString(api.Name), aws.ToString(api.Id), stageErr)
				continue
			}

			for _, s := range stages.Item {
				apis = append(apis, ApiInfo{
					Name:  aws.ToString(api.Name),
					ApiID: aws.ToString(api.Id),
					Stage: aws.ToString(s.StageName),
					Type:  REST,
				})
			}
		}
	}

	if len(apis) == 0 && len(errs) > 0 {
		err := strings.Join(errs, "; ")
		return nil, errors.New(err)
	}

	return apis, nil
}
