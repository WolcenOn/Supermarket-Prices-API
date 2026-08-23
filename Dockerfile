FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/prices-api ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/prices-migrate ./cmd/migrate \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/import-prices ./cmd/import-prices \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/dia-product-probe ./cmd/dia-product-probe \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/enrich-dia-nutrition ./cmd/enrich-dia-nutrition \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/propose-canonical-alias ./cmd/propose-canonical-alias

FROM alpine:3.21
RUN adduser -D -H appuser
USER appuser
COPY --from=build /out/prices-api /usr/local/bin/prices-api
COPY --from=build /out/prices-migrate /usr/local/bin/prices-migrate
COPY --from=build /out/import-prices /usr/local/bin/import-prices
COPY --from=build /out/dia-product-probe /usr/local/bin/dia-product-probe
COPY --from=build /out/enrich-dia-nutrition /usr/local/bin/enrich-dia-nutrition
COPY --from=build /out/propose-canonical-alias /usr/local/bin/propose-canonical-alias
EXPOSE 8080
CMD ["prices-api"]
