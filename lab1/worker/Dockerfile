FROM golang:latest

WORKDIR /go-docker

COPY . .
RUN go build -o worker .

CMD ["./worker"]
