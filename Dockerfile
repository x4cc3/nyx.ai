FROM golang:1.26.2-bookworm AS build

ARG SERVICE=api
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
	-trimpath \
	-ldflags "-s -w -X nyx/internal/version.Version=${VERSION} -X nyx/internal/version.Commit=${COMMIT} -X nyx/internal/version.BuildDate=${BUILD_DATE}" \
	-o /out/nyx \
	./cmd/${SERVICE}

FROM debian:bookworm-slim

ARG SERVICE=api

RUN set -eux; \
	packages="ca-certificates chromium fonts-liberation tini tzdata wget"; \
	if [ "$SERVICE" != "migrate" ]; then \
		packages="$packages docker.io"; \
	fi; \
	apt-get update; \
	apt-get install -y --no-install-recommends $packages; \
	rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /out/nyx /usr/local/bin/nyx

RUN useradd -r -u 1000 -s /sbin/nologin nyx
USER nyx

ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/nyx"]
