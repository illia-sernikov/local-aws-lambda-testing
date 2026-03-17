package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v4"
)

var pulumiAutoNameSuffix = regexp.MustCompile(`-[0-9a-f]{7,}$`)

func staticName(name, fallback string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		trimmed = fallback
	}

	return pulumiAutoNameSuffix.ReplaceAllString(trimmed, "")
}

func BuildConfig(apis []ApiInfo) TraefikConfig {
	cfg := TraefikConfig{
		HTTP: HTTPConfig{
			Routers:     map[string]Router{},
			Services:    map[string]Service{},
			Middlewares: map[string]Middleware{},
		},
	}

	for _, api := range apis {
		stableName := staticName(api.Name, api.ApiID)

		routerName := stableName + "-router"
		serviceName := stableName + "-service"
		middlewareName := stableName + "-path"

		host := fmt.Sprintf(
			"%s.api.localhost.localstack",
			stableName,
		)

		cfg.HTTP.Routers[routerName] = Router{
			Rule:        fmt.Sprintf("Host(`%s`)", host),
			Service:     serviceName,
			EntryPoints: []string{"web"},
			Middlewares: []string{middlewareName},
		}

		cfg.HTTP.Services[serviceName] = Service{
			LoadBalancer: LoadBalancer{
				Servers: []Server{
					{URL: "http://localstack:4566"},
				},
			},
		}

		path := ""

		if api.Type == REST {
			path = fmt.Sprintf(
				"/_aws/execute-api/%s/%s/$1",
				api.ApiID,
				api.Stage,
			)
		} else {
			path = fmt.Sprintf(
				"/%s/$1",
				api.Stage,
			)
		}

		cfg.HTTP.Middlewares[middlewareName] = Middleware{
			ReplacePathRegex: ReplacePathRegex{
				Regex:       "^/(.*)",
				Replacement: path,
			},
		}
	}

	return cfg
}

func WriteConfig(cfg TraefikConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	outputPath := "/traefik/dynamic/apis.yaml"
	outputDir := filepath.Dir(outputPath)
	tmpPath := filepath.Join(outputDir, ".apis.yaml.tmp")

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}

	log.Printf("wrote temporary config %s (%d bytes)", tmpPath, len(data))

	if err := os.Rename(tmpPath, outputPath); err != nil {
		return err
	}

	log.Printf("activated config %s", outputPath)
	return nil
}
