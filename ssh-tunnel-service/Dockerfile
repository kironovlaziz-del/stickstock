FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/tunnel-service .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/tunnel-service /app/tunnel-service
EXPOSE 8081
CMD ["/app/tunnel-service"]
