package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/config"
)

type Handler struct {
	logger *slog.Logger
}

func New() Handler {
	return Handler{logger: slog.Default()}
}

func (h Handler) Handle(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	path := strings.TrimSuffix(req.Path, "/")
	h.logger.InfoContext(ctx, "received request", "method", req.HTTPMethod, "path", path)

	switch {
	case req.HTTPMethod == "GET" && path == "/healthcheck":
		return healthResponse()
	case req.HTTPMethod == "POST" && path == "/calculate":
		return h.calculateResponse(ctx, req)
	default:
		h.logger.WarnContext(ctx, "route not found", "method", req.HTTPMethod, "path", path)
		return jsonResponse(404, errorResponse{Error: "route not found"})
	}
}

type healthPayload struct {
	Status string `json:"status"`
}

func healthResponse() (events.APIGatewayProxyResponse, error) {
	return jsonResponse(200, healthPayload{Status: "ok"})
}

type calculateRequest struct {
	A         float64 `json:"a"`
	B         float64 `json:"b"`
	Operation string  `json:"operation"`
}

type calculatePayload struct {
	Result float64 `json:"result"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h Handler) calculateResponse(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	_, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to load AWS config", "error", err)
		return jsonResponse(500, errorResponse{Error: fmt.Sprintf("failed to load AWS config: %v", err)})
	}

	var payload calculateRequest
	if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
		h.logger.WarnContext(ctx, "invalid JSON body", "error", err)
		return jsonResponse(400, errorResponse{Error: "invalid JSON body"})
	}

	result, err := calculate(payload)
	if err != nil {
		h.logger.WarnContext(ctx, "calculation failed", "operation", payload.Operation, "error", err)
		return jsonResponse(400, errorResponse{Error: err.Error()})
	}

	h.logger.InfoContext(ctx, "calculation succeeded", "operation", payload.Operation, "a", payload.A, "b", payload.B, "result", result)

	return jsonResponse(200, calculatePayload{Result: result})
}

func calculate(req calculateRequest) (float64, error) {
	switch req.Operation {
	case "add":
		return req.A + req.B, nil
	case "subtract":
		return req.A - req.B, nil
	case "multiply":
		return req.A * req.B, nil
	case "divide":
		if req.B == 0 {
			return 0, errors.New("cannot divide by zero")
		}
		return req.A / req.B, nil
	default:
		return 0, errors.New("unsupported operation; use add, subtract, multiply, or divide")
	}
}

func jsonResponse(status int, payload any) (events.APIGatewayProxyResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: "{\"error\":\"failed to serialize response\"}"}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(body),
	}, nil
}
