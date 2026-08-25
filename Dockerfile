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
    && CGO_ENABLED=0 GOOS=linux go build -o /out/propose-canonical-alias ./cmd/propose-canonical-alias \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/review-canonical-alias ./cmd/review-canonical-alias \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/validate-canonical-curation ./cmd/validate-canonical-curation \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/curate-canonical-alias ./cmd/curate-canonical-alias \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/generate-canonical-matches ./cmd/generate-canonical-matches \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/discover-dia-taxonomy ./cmd/discover-dia-taxonomy

FROM alpine:3.21
RUN adduser -D -H appuser
USER appuser
COPY --from=build /out/prices-api /usr/local/bin/prices-api
COPY --from=build /out/prices-migrate /usr/local/bin/prices-migrate
COPY --from=build /out/import-prices /usr/local/bin/import-prices
COPY --from=build /out/dia-product-probe /usr/local/bin/dia-product-probe
COPY --from=build /out/enrich-dia-nutrition /usr/local/bin/enrich-dia-nutrition
COPY --from=build /out/propose-canonical-alias /usr/local/bin/propose-canonical-alias
COPY --from=build /out/review-canonical-alias /usr/local/bin/review-canonical-alias
COPY --from=build /out/validate-canonical-curation /usr/local/bin/validate-canonical-curation
COPY --from=build /out/curate-canonical-alias /usr/local/bin/curate-canonical-alias
COPY --from=build /out/generate-canonical-matches /usr/local/bin/generate-canonical-matches
COPY --from=build /out/discover-dia-taxonomy /usr/local/bin/discover-dia-taxonomy
EXPOSE 8080
CMD ["prices-api"]
