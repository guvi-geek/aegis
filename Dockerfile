FROM public.ecr.aws/docker/library/golang:1.24-bullseye AS builder

ARG VERSION=dev

WORKDIR /go/src/app
COPY . .

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates

RUN go build -o build/aegis -ldflags=-X=main.version=${VERSION} ./cmd


FROM public.ecr.aws/docker/library/debian:bullseye-slim

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

WORKDIR /app
COPY --from=builder /go/src/app/build/aegis /app/aegis

EXPOSE 8080
ENTRYPOINT ["/app/aegis"]
