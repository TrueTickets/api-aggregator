#######################################################################
#                    API Aggregator Dockerfile                        #
#######################################################################
# Stages:
#   build
#   └── production

#######################################################################

FROM golang:1.23.11-bookworm AS build

WORKDIR /go/src/app/svc/api-aggregator
COPY go.mod go.sum /go/src/app/
COPY . /go/src/app/svc/api-aggregator
RUN go mod download

ARG SERVICE_VERSION=dev
ARG COMMIT_HASH
ENV VERSION=${SERVICE_VERSION}-${COMMIT_HASH}

RUN go build -o ../../api-aggregator -ldflags \
    "-X github.com/TrueTickets/api-aggregator/internal/build.ServiceVersion=$VERSION" \
    ./cmd/api-aggregator

#######################################################################
FROM alpine:3 AS runtime

# We're not going to pin a specific version of ca-certificates here
# hadolint ignore=DL3018
RUN set -ex; \
    apk add --no-cache ca-certificates curl libc6-compat && \
    addgroup -g 1001 -S api-aggregator && \
    adduser -u 1001 -S api-aggregator -G api-aggregator \
            --home /opt/api-aggregator

WORKDIR /opt/api-aggregator
COPY --chown=api-aggregator:api-aggregator \
     --from=build \
     /go/src/app/api-aggregator \
     ./

# Copy default configuration
COPY --chown=api-aggregator:api-aggregator \
     config.yaml ./

EXPOSE 8080
USER api-aggregator

CMD ["/opt/api-aggregator/api-aggregator"]
