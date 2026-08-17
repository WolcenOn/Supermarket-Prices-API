FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/prices-api ./cmd/api

FROM alpine:3.21
RUN adduser -D -H appuser
USER appuser
COPY --from=build /out/prices-api /usr/local/bin/prices-api
EXPOSE 8080
CMD ["prices-api"]
