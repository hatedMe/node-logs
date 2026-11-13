# 构建 Go server/worker 双二进制；compose 中以 command 区分进程。
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/server ./cmd/server \
 && CGO_ENABLED=0 go build -trimpath -o /out/worker ./cmd/worker

FROM alpine:3.21
RUN adduser -D -u 10001 appuser
COPY --from=build /out/server /out/worker /usr/local/bin/
USER appuser
EXPOSE 8080
CMD ["server"]
