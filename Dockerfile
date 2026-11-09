# syntax=docker/dockerfile:1

FROM golang:1.27.3-alpine3.24 AS build

RUN apk add --no-cache gcc musl-dev
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd ./cmd
COPY configs ./configs
COPY internal ./internal
COPY pkg ./pkg
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/tbot ./cmd/app

FROM alpine:3.24.4
RUN apk add --no-cache ca-certificates
COPY --from=build /out/tbot /bin/tbot
ENTRYPOINT ["/bin/tbot"]
