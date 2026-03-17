FROM golang:1.24-alpine3.23 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/generator .

FROM alpine:3.23 AS generator

WORKDIR /app
COPY --from=builder /out/generator /app/generator

CMD ["/app/generator"]
