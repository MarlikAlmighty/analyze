FROM golang:1.18-alpine AS builder

ENV CGO_ENABLED 0
ENV TZ=Europe/Moscow

RUN apk update && apk upgrade && apk add --no-cache chromium

RUN echo @edge http://nl.alpinelinux.org/alpine/edge/community >> /etc/apk/repositories \
    && echo @edge http://nl.alpinelinux.org/alpine/edge/main >> /etc/apk/repositories \
    && apk add --no-cache \
    harfbuzz@edge \
    nss@edge \
    freetype@edge \
    ttf-freefont@edge \
    && rm -rf /var/cache/* \
    && mkdir /var/cache/apk

WORKDIR /go/src/analyze

COPY . .

RUN go mod tidy && go build -o /go/src/analyze/app /go/src/analyze/cmd/main.go

FROM gruebel/upx:latest as upx
COPY --from=builder /go/src/analyze/app /app
RUN upx --best --lzma -o /analyze /app

FROM scratch

COPY --from=upx /app /app

ENV BOT_TOKEN=""
ENV CHANNEL=""
ENV RZN_URL=""
ENV YA_URL=""
ENV REDIS_URL="redis://127.0.0.1:6379"

EXPOSE 3000
CMD ["/app"]
