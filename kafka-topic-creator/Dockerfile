FROM golang:1.22.7 AS builder
COPY src/main/go /opt
WORKDIR /opt

RUN go test -coverprofile coverage.out && go tool cover -func=coverage.out
RUN env GOOS=linux GOARCH=amd64 go build

FROM gcr.io/distroless/base-debian12
WORKDIR /opt
COPY --from=builder /opt/kafka-topic-creator /opt/kafka-topic-creator
ENTRYPOINT ["/opt/kafka-topic-creator"]
