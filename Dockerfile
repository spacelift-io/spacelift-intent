# Copyright 2025 Spacelift, Inc. and contributors
# SPDX-License-Identifier: Apache-2.0

FROM alpine

RUN apk add --no-cache ca-certificates

COPY spacelift-intent /usr/local/bin/spacelift-intent
ENTRYPOINT ["/usr/local/bin/spacelift-intent"]
