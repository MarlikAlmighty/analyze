FROM golang:1.21-alpine3.18 AS builder

ENV TZ=Europe/Moscow

WORKDIR /go/src/analyze

COPY . .

RUN go mod tidy && go build -o /go/src/analyze/app /go/src/analyze/cmd/main.go

FROM gruebel/upx:latest as upx
COPY --from=builder /go/src/analyze/app /app
RUN upx --best --lzma -o /analyze /app

#FROM scratch
FROM golang:1.21-alpine3.18

COPY --from=upx /app /app

RUN echo @edge http://nl.alpinelinux.org/alpine/edge/community >> /etc/apk/repositories \
    && echo @edge http://nl.alpinelinux.org/alpine/edge/main >> /etc/apk/repositories \
    && apk update && apk upgrade \
    && apk add --no-cache ca-certificates && update-ca-certificates \
    && apk add --no-cache chromium chromium-chromedriver \
    && rm -rf /var/cache/* \
    && mkdir /var/cache/apk

ENV RZN_URL=""
ENV YA_URL=""
ENV BOT_TOKEN=""
ENV MAIN_CHANNEL="-100***"
ENV MODERATOR_CHANNEL="-955***"

CMD ["/app"]
