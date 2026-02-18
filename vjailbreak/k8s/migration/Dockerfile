# Build the manager binary
FROM golang:1.24 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG V2V_IMG

WORKDIR /workspace
COPY manager /workspace/manager

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
COPY pkg/scripts pkg/scripts
USER 65532:65532

ENTRYPOINT ["/manager"]
