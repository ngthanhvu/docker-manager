FROM golang:1.26.1-alpine

WORKDIR /app

RUN apk add --no-cache git
RUN go install github.com/air-verse/air@v1.63.0

COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]
