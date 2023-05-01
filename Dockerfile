FROM golang:1.19.0 AS builder
COPY . /opt
WORKDIR /opt
RUN env GOOS=linux GOARCH=amd64 go build

FROM gcr.io/distroless/base-debian11
WORKDIR /opt
COPY --from=builder /opt/kafka-topic-creator /opt/kafka-topic-creator
ENTRYPOINT ["/opt/kafka-topic-creator"]
