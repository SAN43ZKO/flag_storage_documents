# flag_storage_documents/Dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata libreoffice openjdk11-jre font-noto
COPY --from=builder /server /server
COPY migrations /migrations
RUN mkdir -p /app/uploads && chmod 777 /app/uploads
VOLUME /app/uploads
EXPOSE 8082
CMD ["/server"]
