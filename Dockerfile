# Example Dockerfile for a Go application (this is just a placeholder)
FROM scratch
COPY go-p2p /
ENTRYPOINT ["/go-p2p"]
