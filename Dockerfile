# syntax=docker/dockerfile:1

FROM golang:1.26.3-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/gh-usecase ./cmd/gh-usecase

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/gh-usecase /gh-usecase

USER nonroot:nonroot
ENTRYPOINT ["/gh-usecase"]
