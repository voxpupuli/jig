# SPDX-License-Identifier: GPL-3.0-or-later

# Build stage: compile a static, CGO-free binary for the target platform.
FROM golang:1-alpine AS build

WORKDIR /src

# Download modules first so this layer is cached across source-only changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# TARGETOS/TARGETARCH are provided by buildx for multi-arch builds.
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /jig

# Runtime stage: minimal alpine with CA certificates for HTTPS git remotes.
FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=build /jig /usr/local/bin/jig
COPY LICENSE /LICENSE

ENTRYPOINT ["jig"]
