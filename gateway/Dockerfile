# syntax=docker/dockerfile:1

FROM golang:1.26.10-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/gateway ./cmd/gateway

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/gateway /app/gateway
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/gateway"]
