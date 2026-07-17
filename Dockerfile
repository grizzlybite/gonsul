FROM golang:1.26.5-alpine AS build
ARG VERSION=dev
ARG BUILD_DATE=unknown

RUN apk --no-cache add build-base git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build VERSION=${VERSION} BUILD_DATE=${BUILD_DATE}

FROM alpine:3.22

RUN apk --no-cache add ca-certificates && \
    adduser -D -H -s /sbin/nologin gonsul
COPY --from=build /src/bin/gonsul /usr/bin/gonsul

USER gonsul
ENTRYPOINT ["/usr/bin/gonsul"]
