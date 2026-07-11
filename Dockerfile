# syntax=docker/dockerfile:1.6

# --- build stage ---
FROM golang:1.25-alpine AS build
WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build -trimpath -ldflags="-s -w" -o /out/billing-svc ./

# --- runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=build /out/billing-svc /app/billing-svc

USER nonroot:nonroot
ENTRYPOINT ["/app/billing-svc"]
