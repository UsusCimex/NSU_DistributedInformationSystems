FROM golang:latest

WORKDIR /go-docker

COPY . .
RUN go build -o manager .

CMD ["./manager"]
