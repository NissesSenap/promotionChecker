FROM golang:1.15-buster as builder

WORKDIR /app

COPY . .

RUN go build -o testbin/promotionTestServer main.go

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
  summary="promotionTestServer is a fake artifactory server" \
  description="promotionTestServer is a fake artifactory server that only responds to a VERY limited set of requests and is used for testing in this project" \
  name="promotionTestServer"

USER 1001

WORKDIR /app

COPY --from=builder /app/testbin/promotionTestServer .

CMD ["./promotionTestServer"]
