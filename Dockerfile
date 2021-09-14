FROM golang:1.17.1-alpine AS builder

COPY . /app

WORKDIR /app

RUN go mod download

RUN CGO_ENABLED=0 go build -o /app/ldtester cmd/ldtester/*.go

FROM scratch

COPY --from=builder /app/ldtester /app/ldtester

COPY --from=builder /app/config.yaml /app/config.yaml

WORKDIR /app

EXPOSE 8000

CMD ["./ldtester"]

