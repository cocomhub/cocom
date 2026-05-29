# Copyright 2026 The Cocomhub Authors. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

# Build stage
FROM golang:1.26-alpine AS build
ARG GOPROXY=https://goproxy.cn,direct

WORKDIR /src
COPY go.mod go.sum ./
RUN apk add --no-cache git
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /usr/local/bin/cocom -trimpath \
    -ldflags="-s -w -X 'github.com/cocomhub/cocom/pkg/version.Version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)'"

# Runtime stage
FROM alpine:3.21 AS runtime

RUN apk add --no-cache wget
RUN addgroup -S cocom && adduser -S cocom -G cocom

COPY --from=build /usr/local/bin/cocom /usr/local/bin/cocom

RUN mkdir -p /data/cocom/data/gallery /data/cocom/data/archive /data/cocom/data/archive-temp && \
    chown -R cocom:cocom /data/cocom

USER cocom
EXPOSE 15456
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:15456/healthz || exit 1

CMD ["cocom", "server", "-p", "15456"]
