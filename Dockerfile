# build
FROM golang:1.8-alpine3.6 as builder

ADD . /go/src/fmbot
WORKDIR /go/src/fmbot

RUN apk update && \
    apk add --no-cache git && \
    go get -v fmbot && \
    GOOS=linux go build -v -o /go/bin/fmbot .

# run
FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /go/bin/fmbot /opt/fmbot
RUN ["chmod", "+x", "/opt/fmbot"]

ENTRYPOINT ["/opt/fmbot"]
