# syntax=docker/dockerfile:1
FROM golang:1.26 AS base
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

FROM base AS service-build
ARG TARGET
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/${TARGET}

FROM base AS e2e-build
RUN CGO_ENABLED=0 GOOS=linux go test -c -o /out/e2e ./test/e2e

FROM scratch AS e2e
COPY --from=e2e-build /out/e2e /e2e
ENTRYPOINT ["/e2e", "-test.v"]

FROM scratch AS service
COPY --from=service-build /out/app /app
ENTRYPOINT ["/app"]
