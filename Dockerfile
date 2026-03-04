FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOARCH=$(go env GOARCH) go build -ldflags="-X main.version=${VERSION}" -o /bin/flaggy ./cmd/flaggy

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/flaggy /usr/local/bin/

EXPOSE 8080

ENTRYPOINT ["flaggy"]
CMD ["serve"]
