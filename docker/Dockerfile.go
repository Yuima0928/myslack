FROM golang:1.24 AS build
WORKDIR /src
COPY backend/go/go.mod backend/go/go.sum ./
RUN go mod download
COPY backend/go .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/api ./cmd/api

FROM gcr.io/distroless/base-debian12
WORKDIR /
EXPOSE 8000
COPY --from=build /app/api /api
ENTRYPOINT ["/api"]
