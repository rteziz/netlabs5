FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY protocol ./protocol
COPY cmd/server ./cmd/server
RUN CGO_ENABLED=0 go build -trimpath -o /out/server ./cmd/server

FROM alpine:3.22
RUN apk add --no-cache iproute2
WORKDIR /app
COPY --from=build /out/server /usr/local/bin/server
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh
EXPOSE 9000
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["9000"]
