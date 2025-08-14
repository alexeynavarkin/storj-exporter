.PHONY: docker-push

docker-push:
	docker buildx build \
        --platform linux/amd64,linux/arm64 \
        -t alexnav/storj-exporter:v0.0.1 \
        -t alexnav/storj-exporter:latest \
        --push .
