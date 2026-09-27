# The image of a bos API, assembled from a binary that was built beforehand:
#   CGO_ENABLED=0 GOOS=linux GOARCH=<the docker daemon's> go build -o api .
# so building the image needs its base image and nothing else (no module downloads).
#
# Build context: a folder that holds that binary as the file `api`.
# The API listens on 8080 inside the container; the bundle publishes it on a host port of its own.
FROM alpine:3.20

RUN adduser -D -u 10001 app
WORKDIR /app
COPY --chmod=755 api /app/api

ENV PORT=8080
USER app
EXPOSE 8080

ENTRYPOINT ["/app/api"]
