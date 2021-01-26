FROM golang:alpine

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

COPY ./deploy.go .

RUN go build -o upload ./deploy.go

WORKDIR /dist

RUN cp /build/upload .

EXPOSE 6666

CMD ["/dist/upload"]