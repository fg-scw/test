# Stage 1: Build environment
# Using Fedora 42 which includes Go 1.24 required by our go.mod
FROM fedora:42 AS builder

# Build arguments for multi-platform support and versioning
ARG TARGETOS
ARG TARGETARCH
ARG RELEASE_VERSION
ENV RELEASE_VERSION=$RELEASE_VERSION

# Install build dependencies
# - golang: Required for building the application (Go 1.24.1 from Fedora 42)
# - libnbd-devel: Development files for NBD (Network Block Device) support
RUN dnf install -y golang libnbd-devel

# Set up build workspace
WORKDIR /workspace

# Copy pre-built binary from local build
# Note: The binary is built by 'make -C v2v-helper build' before docker build
# This is part of the VM migration toolchain for converting between different platforms
COPY v2v-helper/manager manager

# Stage 2: Runtime environment
# Using same Fedora version for consistency and required runtime dependencies
FROM fedora:42

WORKDIR /tmp/rpms
COPY v2v-helper/nbdkit/*.rpm /tmp/rpms/

# Install runtime dependencies with size optimizations:
# - --nodocs: Skip documentation files
# - --setopt=install_weak_deps=False: Skip weak dependencies
# - Clean up aggressively after install
RUN dnf install -y \
    --nodocs \
    --setopt=install_weak_deps=False \
    ./libnbd-1.22.2-1.fc42.x86_64.rpm \
    ./nbdkit-server-1.42.4-1.fc42.x86_64.rpm \
    ./nbdkit-basic-plugins-1.42.4-1.fc42.x86_64.rpm \
    ./nbdkit-basic-filters-1.42.4-1.fc42.x86_64.rpm \
    ./nbdkit-1.42.4-1.fc42.x86_64.rpm \
    ./nbdkit-vddk-plugin-1.42.4-1.fc42.x86_64.rpm \
    ./nbdkit-ssh-plugin-1.42.4-1.fc42.x86_64.rpm \
    ./nbdkit-curl-plugin-1.42.4-1.fc42.x86_64.rpm \
    ./nbdkit-python-plugin-1.42.4-1.fc42.x86_64.rpm \
    ./nbdkit-nbd-plugin-1.42.4-1.fc42.x86_64.rpm \
    ./virt-v2v-2.7.17-1.fc42.x86_64.rpm \
    ./nbdkit-selinux-1.42.4-1.fc42.noarch.rpm \
    guestfs-tools \
    netplan && \
    dnf clean all && \
    rm -rf /var/cache/dnf /var/cache/yum /tmp/rpms/ && \
    rm -rf /usr/share/doc /usr/share/man /usr/share/info /usr/share/locale && \
    find /var/log -type f -delete

# Version lock to prevent accidental upgrades
# Install versionlock plugin without docs to save space
RUN dnf install -y --nodocs --setopt=install_weak_deps=False python3-dnf-plugin-versionlock && \
    dnf versionlock add nbdkit-1.42.4-1.fc42 nbdkit-vddk-plugin-1.42.4-1.fc42 \
    libnbd-1.22.2-1.fc42 nbdkit-server-1.42.4-1.fc42 nbdkit-basic-plugins-1.42.4-1.fc42 \ 
    nbdkit-basic-filters-1.42.4-1.fc42 nbdkit-curl-plugin-1.42.4-1.fc42 \
    nbdkit-ssh-plugin-1.42.4-1.fc42 nbdkit-python-plugin-1.42.4-1.fc42 \
    nbdkit-nbd-plugin-1.42.4-1.fc42 virt-v2v-2.7.17-1.fc42.x86_64 nbdkit-selinux-1.42.4-1.fc42.noarch && \
    dnf clean all && \
    rm -rf /var/cache/dnf /var/cache/yum

# Copy the binary from builder stage to the runtime environment
COPY --from=builder /workspace/manager /home/fedora/manager

# Copy scripts directory for migration utilities
COPY scripts/generate-mount-persistence.sh /home/fedora/generate-mount-persistence.sh
COPY scripts/get-bootable-partition.sh /home/fedora/get-bootable-partition.sh
COPY scripts/generate-udev-mapping.sh /home/fedora/generate-udev-mapping.sh
COPY scripts/firstboot/windows/NIC-Recovery /home/fedora/NIC-Recovery
COPY scripts/firstboot/store /home/fedora/store
COPY scripts/firstboot/windows/Firstboot-Init-Windows.bat /home/fedora/Firstboot-Init-Windows.sh