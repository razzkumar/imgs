# syntax=docker/dockerfile:1

ARG GO_VERSION=1.22
FROM golang:${GO_VERSION} AS build
WORKDIR /src

# RUN --mount=type=cache,target=/go/pkg/mod/ \
#   --mount=type=bind,source=go.sum,target=go.sum \
#   --mount=type=bind,source=go.mod,target=go.mod \
#   go mod download -x

RUN --mount=type=cache,target=/go/pkg/mod/ \
  --mount=type=bind,target=. \
  CGO_ENABLED=0 go build -o /bin/server .

FROM alpine:latest AS final

WORKDIR /app
RUN --mount=type=cache,target=/var/cache/apk \
  apk --update add \
  ca-certificates \
  tzdata \
  && \
  update-ca-certificates && \
  mkdir assets

COPY --from=build /bin/server /app

EXPOSE 8080

ENTRYPOINT [ "/app/server" ]
