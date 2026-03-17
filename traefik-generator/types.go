package main

const (
	REST APIType = "rest"
	HTTP APIType = "http"
)

type APIType string

type ApiInfo struct {
	Name  string
	ApiID string
	Stage string
	Type  APIType
}

type TraefikConfig struct {
	HTTP HTTPConfig `yaml:"http"`
}

type HTTPConfig struct {
	Routers     map[string]Router     `yaml:"routers"`
	Services    map[string]Service    `yaml:"services"`
	Middlewares map[string]Middleware `yaml:"middlewares"`
}

type Router struct {
	Rule        string   `yaml:"rule"`
	Service     string   `yaml:"service"`
	EntryPoints []string `yaml:"entryPoints"`
	Middlewares []string `yaml:"middlewares,omitempty"`
}

type Service struct {
	LoadBalancer LoadBalancer `yaml:"loadBalancer"`
}

type LoadBalancer struct {
	Servers []Server `yaml:"servers"`
}

type Server struct {
	URL string `yaml:"url"`
}

type Middleware struct {
	ReplacePathRegex ReplacePathRegex `yaml:"replacePathRegex"`
}

type ReplacePathRegex struct {
	Regex       string `yaml:"regex"`
	Replacement string `yaml:"replacement"`
}
