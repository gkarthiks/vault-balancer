FROM golang:1.14-alpine AS build_base
ARG BUILD_VERSION
RUN apk add --no-cache git
WORKDIR /tmp/vault-balancer

COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -ldflags "-X main.BuildVersion=$BUILD_VERSION" -o ./out/vault-balancer .


FROM alpine:3.12
RUN apk add ca-certificates
COPY --from=build_base /tmp/vault-balancer/out/vault-balancer /app/vault-balancer

#EXPOSE 8080

CMD ["/app/vault-balancer"]
