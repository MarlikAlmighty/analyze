FROM golang:1.14-alpine3.12 AS builder

WORKDIR /go/src/analyze-it

COPY . .

RUN go build -o /go/src/analyze-it/app /go/src/analyze-it/cmd/main.go

FROM alpine:3.12

COPY --from=builder /go/src/analyze-it/app /

CMD ["/app"]
