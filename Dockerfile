# Step 1: Modules caching
FROM golang:1.24-alpine3.21 AS modules

COPY go.mod go.sum /modules/

WORKDIR /modules

RUN go mod download

# Step 2: Builder
FROM golang:1.24-alpine3.21 AS builder

ARG TARGETARCH

RUN apk add --no-cache ca-certificates make tzdata

COPY --from=modules /go/pkg /go/pkg
COPY . /app

WORKDIR /app

RUN make generate
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o engine ./cmd/app/main.go

# Step 3: Final
FROM scratch

COPY --from=builder /app/engine /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

CMD ["/app"]
