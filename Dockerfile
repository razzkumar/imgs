# syntax=docker/dockerfile:1

ARG GO_VERSION=1.22
FROM golang:${GO_VERSION} AS build
WORKDIR /src

COPY . .

RUN go mod download -x
RUN CGO_ENABLED=0 go build -o /bin/server .


FROM alpine:latest AS final

WORKDIR /app
RUN apk --update add \
  ca-certificates \
  tzdata \
  && \
  update-ca-certificates && \
  mkdir assets

COPY --from=build /bin/server /app

EXPOSE 8080

ENTRYPOINT [ "/app/server" ]
