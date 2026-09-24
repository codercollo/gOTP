# GoTP — static build. The dashboard is embedded in the binary (web/web.go),
# so the final image is just the binary. Distroless, non-root, tiny.

FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
# Static, stripped, version burned in.
RUN CGO_ENABLED=0 go build -ldflags='-s -w' -o /bin/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /bin/api /api
EXPOSE 4000
ENV PORT=4000
ENTRYPOINT ["/api"]
