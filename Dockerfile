FROM golang:alpine

COPY . /go/src/github.com/fogger/generator

WORKDIR /go/src/github.com/fogger/generator
RUN go build -o /usr/local/bin/generator ./cmd/generator

EXPOSE 3000

CMD ["/usr/local/bin/generator"]
