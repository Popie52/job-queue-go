FROM golang:1.25.5-alpine

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o jobqueue ./cmd/jobqueue

EXPOSE 8080

CMD ["./jobqueue"]