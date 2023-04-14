FROM golang:1.20 AS builder

WORKDIR /go/src/github.com/yourname/yourproject
COPY . .
RUN go build -o /go/bin/yourproject
CMD ["/go/bin/yourproject"]
