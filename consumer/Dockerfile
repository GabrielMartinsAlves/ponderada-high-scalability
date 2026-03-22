FROM golang:1.24-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY consumer/ ./consumer/

RUN cd consumer && go build -o telemetry-consumer .

CMD ["./consumer/telemetry-consumer"]