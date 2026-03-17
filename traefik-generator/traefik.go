package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v4"
)

func BuildConfig(apis []ApiInfo) TraefikConfig {
	cfg := TraefikConfig{
		HTTP: HTTPConfig{
			Routers:     map[string]Router{},
			Services:    map[string]Service{},
			Middlewares: map[string]Middleware{},
		},
	}

	for _, api := range apis {

		routerName := api.Name + "-router"
		serviceName := api.Name + "-service"
		middlewareName := api.Name + "-path"

		host := fmt.Sprintf(
			"%s.api.localhost.localstack",
			api.Name,
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
	tmpPath := filepath.Join(filepath.Dir(outputPath), ".apis.yaml.tmp")

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
