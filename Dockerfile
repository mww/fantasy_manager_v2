FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.23-alpine AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app/
ADD . .
RUN go mod download
# Skip running go test because most of the tests use testcontainers which
# don't run in the docker build step.
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o fantasy_manager_v2

FROM --platform=${TARGETPLATFORM:-linux/amd64} scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
WORKDIR /app/
COPY --from=builder /app/fantasy_manager_v2 /app/fantasy_manager_v2

EXPOSE 3000
ENTRYPOINT [ "/app/fantasy_manager_v2" ]