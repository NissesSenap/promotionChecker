FROM golang:1.15-buster as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o bin/promotionChecker main.go

FROM debian:buster-slim

ARG VERSION=0.0.1
ARG BUILD_DATE=2020-11-1

LABEL \
  org.opencontainers.image.created="$BUILD_DATE" \
  org.opencontainers.image.authors="edvin.norling@redhat.com" \
  org.opencontainers.image.homepage="https://github.com/NissesSenap/promotionChecker" \
  org.opencontainers.image.documentation="https://github.com/NissesSenap/promotionChecker" \
  org.opencontainers.image.source="https://github.com/NissesSenap/promotionChecker" \
  org.opencontainers.image.version="$VERSION" \
  org.opencontainers.image.vendor="GitHub" \
  org.opencontainers.image.licenses="MIT" \
  summary="promotionChecker is a pre-webhook tool for artifactory" \
  description="promotionChecker is a pre-webhook tool for artifactory that polls artifactory according to a pre-defined sync window" \
  name="gh"

USER 1001

WORKDIR /app

COPY --from=builder /app/bin/promotionChecker .

CMD ["./promotionChecker"]
