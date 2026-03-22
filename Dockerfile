FROM golang:1-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o telemetry-system .

EXPOSE 8080

CMD ["./telemetry-system"]