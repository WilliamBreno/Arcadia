FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/api /out/migrate ./
COPY migrations ./migrations
COPY entrypoint.sh ./
RUN tr -d '' < entrypoint.sh > entrypoint.tmp && mv entrypoint.tmp entrypoint.sh && chmod +x entrypoint.sh
USER app
ENV TZ=America/Sao_Paulo
EXPOSE 8080
ENTRYPOINT ["./entrypoint.sh"]
