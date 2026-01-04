FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /tempo-federated-querier ./cmd/tempo-federated-querier

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /tempo-federated-querier /tempo-federated-querier

EXPOSE 3200

ENTRYPOINT ["/tempo-federated-querier"]
