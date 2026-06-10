FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates
WORKDIR /app

COPY ./stock-event-generator-service /app/stock-event-generator-service
COPY ./common-go-lib /app/common-go-lib
WORKDIR /app/stock-event-generator-service
RUN go mod tidy
RUN go build -o /bin/stock-event-generator-service .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/stock-event-generator-service /bin/stock-event-generator-service
COPY --from=builder /app/stock-event-generator-service/config /app/stock-event-generator-service/config
WORKDIR /app/stock-event-generator-service
ENTRYPOINT ["/bin/stock-event-generator-service"]
