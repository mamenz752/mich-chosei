FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o bot main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/bot .
# タイムゾーンデータの追加
RUN apk add --no-cache tzdata
ENV TZ=Asia/Tokyo

CMD ["./bot"]