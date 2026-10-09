FROM golang:1.26-alpine AS builder

WORKDIR /app

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -ldflags "-s -w" -o traily-backend ./cmd/main.go

FROM golang:1.26-alpine AS dev

WORKDIR /app

RUN go install github.com/air-verse/air@v1.61.7

COPY go.mod go.sum ./

RUN go mod download

EXPOSE 8080

ENTRYPOINT [ "air", "-c", ".air.toml" ]

FROM gcr.io/distroless/static-debian12

WORKDIR /app

COPY --from=builder /app/traily-backend .

EXPOSE 8080

ENTRYPOINT [ "./traily-backend" ]
