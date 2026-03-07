# Copyright 2026 The Cocomhub Authors. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.21-alpine AS build

ADD ./ cocom

RUN cd cocom \
    && go env -w GO111MODULE="on" \
    && go env -w GOPROXY="https://goproxy.cn,direct" \
    && go build -o cocom \
    && cp cocom /usr/local/bin

FROM alpine:3.18.4 AS runtinue
COPY --from=build /usr/local/bin/cocom /usr/local/bin/cocom
CMD ["cocom", "server", "--configPath", "/etc/cocom"]
