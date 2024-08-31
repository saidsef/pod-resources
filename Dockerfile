# Build
FROM golang:1.23 AS build
WORKDIR /app
ENV CGO_ENABLED=0 GOOS=linux
COPY ./ ./
RUN go build -v -ldflags "-s -w" -trimpath -buildvcs -compiler gc -o ./pod-resources ./resources/resources.go

# Application
FROM scratch

LABEL org.opencontainers.image.title="Pod Resources"
LABEL org.opencontainers.image.description="Kubernetes Container Resources"
LABEL org.opencontainers.image.source="https://github.com/saidsef/pod-resources.git"
LABEL com.docker.extension.publisher-url="https://github.com/saidsef/pod-resources.git"
LABEL com.docker.extension.categories="kubernetes,resources"

COPY --from=build /app/pod-resources /
CMD ["/pod-resources"]