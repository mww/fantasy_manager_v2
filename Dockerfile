FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.22-alpine AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app/
ADD . .
RUN go mod download
# For now ignore the tests in ./db because they use testcontainers and that
# wasn't working with a simple `go test ./...`
RUN CGO_ENABLED=0 go test ./controller/... ./model/... ./sleeper/... ./web/...
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o fantasy_manager_v2

FROM --platform=${TARGETPLATFORM:-linux/amd64} scratch
WORKDIR /app/
COPY --from=builder /app/fantasy_manager_v2 /app/fantasy_manager_v2

EXPOSE 3000
ENTRYPOINT [ "/app/fantasy_manager_v2" ]