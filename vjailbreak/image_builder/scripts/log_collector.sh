#!/bin/bash
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

LOG_DIR="/var/log/pf9"
HOSTNAME=$(hostname -s 2>/dev/null || hostname)
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
OUTPUT_DIR="/tmp/vjb-support-bundle-${HOSTNAME}-${TIMESTAMP}"
NAMESPACE="migration-system"
DATE_FILTER=""
VM_FILTER=""
INCLUDE_DEBUG=false

print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Collect vJailbreak logs and system information for support bundle.

OPTIONS:
    --vms <vm1,vm2,vm3>     Filter logs for specific VMs (comma-separated)
    --vms-file <file.csv>   Read VM names from CSV file (one VM per line or comma-separated)
    --date <timestamp>      Filter logs from specific date (Unix timestamp or YYYY-MM-DD)
    --debug                 Include debug logs
    --output-dir <path>     Custom output directory (default: /tmp/vjb-support-bundle-<hostname>-<timestamp>)
    --namespace <name>      Kubernetes namespace (default: migration-system)
    -h, --help              Show this help message

EXAMPLES:
    $0 --vms migration-gk-vm-10,migration-gk-vm-20
    $0 --vms-file vms.csv
    $0 --date 2025-10-27
    $0 --date 1730000000 --debug
    $0 --vms vm1,vm2 --date 2025-10-27 --debug

CSV FILE FORMAT:
    vms.csv can contain VM names in any of these formats:
    - One VM per line
    - Comma-separated on a single line
    - Mix of both

EOF
    exit 1
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --vms)
            VM_FILTER="$2"
            shift 2
            ;;
        --vms-file)
            if [ ! -f "$2" ]; then
                print_error "VM file not found: $2"
                exit 1
            fi
            VM_FILTER=$(cat "$2" | tr '\n' ',' | tr -s ',' | sed 's/,$//')
            print_info "Loaded VMs from file: $2"
            shift 2
            ;;
        --date)
            DATE_FILTER="$2"
            shift 2
            ;;
        --debug)
            INCLUDE_DEBUG=true
            shift
            ;;
        --output-dir)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        *)
            print_error "Unknown option: $1"
            usage
            ;;
    esac
done

print_info "Creating support bundle directory: $OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"/{logs,versions,configs,system-info}

collect_controller_logs() {
    print_info "Collecting controller logs"
    if command -v kubectl &> /dev/null; then
	local pods=$(sudo kubectl get pods -n "$NAMESPACE" -o name | grep migration-controller | cut -d'/' -f2)
        if [ -n "$pods" ]; then
            for pod in $pods; do
                print_info "Collecting logs from pod: $pod"
                sudo kubectl logs -n "$NAMESPACE" "$pod" > "$OUTPUT_DIR/logs/controller-${pod}.log" 2>&1 || print_warn "Failed to get logs from $pod"
                sudo kubectl logs -n "$NAMESPACE" "$pod" --previous > "$OUTPUT_DIR/logs/controller-${pod}-previous.log" 2>&1 || true
            done
        else
            print_warn "No controller pods found in namespace: $NAMESPACE"
        fi
    else
        print_warn "kubectl not found, skipping controller logs"
    fi
}

collect_migration_logs() {
    print_info "Collecting migration logs from $LOG_DIR"
    if [ ! -d "$LOG_DIR" ]; then
        print_warn "Log directory not found: $LOG_DIR"
        return
    fi
    local find_date_filter=""
    if [ -n "$DATE_FILTER" ]; then
        if [[ "$DATE_FILTER" =~ ^[0-9]+$ ]]; then
            find_date_filter="-newermt @$DATE_FILTER"
        else
            find_date_filter="-newermt $DATE_FILTER"
        fi
    fi

    if [ -n "$VM_FILTER" ]; then
        IFS=',' read -ra VMS <<< "$VM_FILTER"
        for vm in "${VMS[@]}"; do
            vm=$(echo "$vm" | xargs)
            print_info "Collecting logs for VM: $vm"
            if [ -f "$LOG_DIR/migration-${vm}.log" ]; then
                cp "$LOG_DIR/migration-${vm}.log" "$OUTPUT_DIR/logs/" 2>/dev/null || true
            fi
            find "$LOG_DIR" -maxdepth 1 -type f -name "*${vm}*.log" $find_date_filter -exec cp {} "$OUTPUT_DIR/logs/" \; 2>/dev/null || true
            if [ -d "$LOG_DIR/migration-${vm}" ]; then
                print_info "  - Copying directory: migration-${vm}/"
                cp -r "$LOG_DIR/migration-${vm}" "$OUTPUT_DIR/logs/" 2>/dev/null || true
            fi
            find "$LOG_DIR" -maxdepth 1 -type d -name "*${vm}*" $find_date_filter -exec cp -r {} "$OUTPUT_DIR/logs/" \; 2>/dev/null || true
            local log_count=$(find "$OUTPUT_DIR/logs" -type f -name "*${vm}*.log" 2>/dev/null | wc -l)
            print_info "  - Collected $log_count log file(s) for $vm"
        done
    else
        print_info "Collecting all migration logs"
        find "$LOG_DIR" -maxdepth 1 -type f -name "migration-*.log" $find_date_filter -exec cp {} "$OUTPUT_DIR/logs/" \; 2>/dev/null || true
        for dir in "$LOG_DIR"/migration-*/; do
            if [ -d "$dir" ]; then
                local dirname=$(basename "$dir")
                print_info "  - Copying directory: $dirname/"
                cp -r "$dir" "$OUTPUT_DIR/logs/" 2>/dev/null || true
            fi
        done
        local file_count=$(find "$OUTPUT_DIR/logs" -type f -name "*.log" 2>/dev/null | wc -l)
        local dir_count=$(find "$OUTPUT_DIR/logs" -mindepth 1 -type d 2>/dev/null | wc -l)
        print_info "  - Collected $file_count log file(s) in $dir_count directory/directories"
    fi
    if [ ! "$(ls -A $OUTPUT_DIR/logs 2>/dev/null)" ]; then
        print_warn "No migration logs found matching the criteria"
    fi
}
collect_version_info() {
    print_info "Collecting version information"
    local version_file="$OUTPUT_DIR/versions/version-info.txt"
    (
    set +e  
    {
        echo "================================================================"
        echo "vJailbreak Version Information"
        echo "Collected: $(date)"
        echo "================================================================"
        echo ""

        echo "=== System Information ==="
        echo "OS: $(cat /etc/os-release 2>/dev/null | grep "^PRETTY_NAME=" | cut -d'"' -f2 || echo "Unknown")"
        echo "Kernel: $(uname -r 2>/dev/null || echo "Unknown")"
        echo "Architecture: $(uname -m 2>/dev/null || echo "Unknown")"
        echo ""

        echo "=== Kubernetes Version ==="
        if command -v kubectl &> /dev/null; then
            local k8s_version="Could not retrieve"
            if sudo kubectl version --short &>/dev/null; then
                k8s_version=$(sudo kubectl version --short 2>/dev/null | grep "Server Version" | awk '{print $3}')
            elif sudo kubectl version -o json &>/dev/null; then
                k8s_version=$(sudo kubectl version -o json 2>/dev/null | grep '"gitVersion"' | head -1 | cut -d'"' -f4)
            fi
            echo "Version: $k8s_version"
        else
            echo "kubectl not available"
        fi
        echo ""

        echo "=== Controller Version ==="
        if command -v kubectl &> /dev/null; then
            local controller_image=""
            for deployment_name in migration-controller migration-controller-manager controller; do
                controller_image=$(sudo kubectl get deployment -n "$NAMESPACE" "$deployment_name" -o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null)
                if [ -n "$controller_image" ]; then
                    echo "Deployment: $deployment_name"
                    break
                fi
            done

            if [ -n "$controller_image" ]; then
                echo "Image: $controller_image"
                local version=$(echo "$controller_image" | awk -F':' '{print $2}')
                [ -z "$version" ] && version="latest"
                echo "Version: $version"
            else
                echo "Could not retrieve controller version (checked: migration-controller, migration-controller-manager, controller)"
            fi
        else
            echo "kubectl not available"
        fi
        echo ""

        echo "=== v2v-helper Image ==="
        if command -v kubectl &> /dev/null; then
            local helper_pod=$(sudo kubectl get pods -n "$NAMESPACE" --sort-by=.metadata.creationTimestamp 2>/dev/null | grep "^v2v-helper-" | tail -1 | awk '{print $1}')
            if [ -n "$helper_pod" ]; then
                echo "Pod: $helper_pod"
                local helper_image=$(sudo kubectl get pod -n "$NAMESPACE" "$helper_pod" -o jsonpath='{.spec.containers[0].image}' 2>/dev/null)
                echo "Image: $helper_image"
                local version=$(echo "$helper_image" | awk -F':' '{print $2}')
                [ -z "$version" ] && version="latest"
                echo "Version: $version"
            else
                echo "No v2v-helper pods found"
            fi
        else
            echo "kubectl not available"
        fi
        echo ""
        echo "=== Tool Versions (from migration logs) ==="
        if [ -d "$LOG_DIR" ]; then
            local recent_log=$(find "$LOG_DIR" -mindepth 2 -type f -name "migration.*.log" -printf '%T@ %p\n' 2>/dev/null | sort -rn | head -1 | cut -d' ' -f2-)

            if [ -z "$recent_log" ]; then
                recent_log=$(find "$LOG_DIR" -type f -name "migration*.log" -printf '%T@ %p\n' 2>/dev/null | sort -rn | head -1 | cut -d' ' -f2-)
            fi

            if [ -n "$recent_log" ]; then
                local rel_path=$(echo "$recent_log" | sed "s|^$LOG_DIR/||")
                echo "Source: $rel_path"
                echo ""

                local has_nbdkit_debug=$(grep -c "nbdkit: debug:" "$recent_log" 2>/dev/null || echo "0")

                if [ "$has_nbdkit_debug" -gt 0 ]; then
                    echo "nbdkit Version:"
                    local nbdkit_line=$(grep "nbdkit: debug: nbdkit [0-9]" "$recent_log" 2>/dev/null | head -1)
                    if [ -n "$nbdkit_line" ]; then
                        echo "  $(echo "$nbdkit_line" | sed 's/.*nbdkit: debug: //')"
                    else
                        echo "  Not found in logs"
                    fi
                    echo ""

                    echo "VDDK Library:"
                    local vddk_line=$(grep "vddk: config key=libdir" "$recent_log" 2>/dev/null | head -1)
                    if [ -n "$vddk_line" ]; then
                        local vddk_path=$(echo "$vddk_line" | sed 's/.*value=//')
                        echo "  Path: $vddk_path"
                        if echo "$vddk_path" | grep -q "vmware-vix-disklib"; then
                            local vddk_version=$(echo "$vddk_path" | sed -n 's/.*disklib-distrib-\([0-9.]*\).*/\1/p')
                            if [ -n "$vddk_version" ]; then
                                echo "  Version: $vddk_version"
                            else
                                echo "  Version: (path-based, version may be embedded in directory name)"
                            fi
                        fi
                    else
                        echo "  Not found in logs"
                    fi
                    echo ""

                    echo "libvixDiskLib.so:"
                    local libvix_line=$(grep "libvixDiskLib" "$recent_log" 2>/dev/null | head -1)
                    if [ -n "$libvix_line" ]; then
                        echo "  $(echo "$libvix_line" | sed 's/^[[:space:]]*nbdkit: [^:]*: //' | sed 's/^[[:space:]]*//')"
                    else
                        echo "  Not mentioned in logs"
                    fi
                    echo ""

                    echo "nbdcopy/libnbd:"
                    local nbdcopy_cmd=$(grep "COMMAND:.*nbdcopy" "$recent_log" 2>/dev/null | head -1)
                    if [ -n "$nbdcopy_cmd" ]; then
                        echo "  nbdcopy is being used"
                        local nbdcopy_version=$(grep -E "nbdcopy.*[0-9]+\.[0-9]+" "$recent_log" 2>/dev/null | head -1 | grep -o '[0-9]\+\.[0-9]\+\(\.[0-9]\+\)\?' | head -1)
                        if [ -n "$nbdcopy_version" ]; then
                            echo "  Version: $nbdcopy_version"
                        fi
                    else
                        echo "  Not found in logs"
                    fi
                    echo ""
                else
                    echo "Note: Selected log file is a high-level migration log without detailed tool version information."
                    echo "Detailed version info is typically in subdirectory logs (migration-<vm>/migration.*.log)."
                    echo ""
                fi
            else
                echo "No migration logs found in $LOG_DIR"
            fi
        else
            echo "Log directory not found: $LOG_DIR"
        fi
        echo "================================================================"
    } > "$version_file"
    )
    print_info "Version information saved to: $version_file"
}

collect_v2v_helper_info() {
    print_info "Collecting v2v-helper pod information"
    set +e
    if command -v  kubectl &> /dev/null; then
        local pod=$(sudo kubectl get pods -n "$NAMESPACE" --sort-by=.metadata.creationTimestamp 2>/dev/null | grep "^v2v-helper-" | tail -1 | awk '{print $1}')
        if [ -n "$pod" ]; then
            print_info "Collecting info from v2v-helper pod: $pod"
            sudo kubectl describe pod -n "$NAMESPACE" "$pod" > "$OUTPUT_DIR/system-info/v2v-helper-pod-describe.txt" 2>&1 || true
        else
            print_warn "No v2v-helper pods found"
        fi
    fi
    set -e
}

collect_configmaps() {
    print_info "Collecting vJB ConfigMaps"
    set +e
    if command -v kubectl &> /dev/null; then
        local important_configs="pf9-env version-config vjailbreak-settings"
        local config_file="$OUTPUT_DIR/configs/configmaps.yaml"

        echo "# vJailbreak ConfigMaps (filtered for relevance)" > "$config_file"
        echo "# Excluded: firstboot-config-* (redundant defaults), kube-root-ca.crt (k8s system config)" >> "$config_file"
        echo "---" >> "$config_file"

        for config in $important_configs; do
            if sudo kubectl get configmap -n "$NAMESPACE" "$config" &>/dev/null; then
                print_info "  - Collecting configmap: $config"
                sudo kubectl get configmap -n "$NAMESPACE" "$config" -o yaml >> "$config_file" 2>&1
                echo "---" >> "$config_file"
            fi
        done

        # collect one sample migration-config to show the structure
        local sample_migration_config=$(sudo kubectl get configmap -n "$NAMESPACE" 2>/dev/null | grep "migration-config-" | head -1 | awk '{print $1}')
        if [ -n "$sample_migration_config" ]; then
            print_info "  - Collecting sample migration config: $sample_migration_config"
            echo "# Sample migration-config (one representative example)" >> "$config_file"
            sudo kubectl get configmap -n "$NAMESPACE" "$sample_migration_config" -o yaml >> "$config_file" 2>&1
            echo "---" >> "$config_file"
        fi
    fi
    set -e 
}

collect_system_info() {
    print_info "Collecting system information..."
    set +e
    {
        echo "=== System Information ==="
        echo "Hostname: $(hostname)"
        echo "Date: $(date)"
        echo "Uptime: $(uptime)"
        echo ""
        echo "=== Disk Usage ==="
        df -h
        echo ""
        echo "=== Memory Usage ==="
        free -h
        echo ""
        echo "=== Kubernetes Cluster Info ==="
        if command -v kubectl &> /dev/null; then
            sudo kubectl cluster-info 2>/dev/null || echo "Could not retrieve cluster info"
            echo ""
            sudo kubectl get nodes -o wide 2>/dev/null || echo "Could not retrieve node info"
        fi
        echo ""
        echo "=== vJB Namespace Resources ==="
        if command -v  kubectl &> /dev/null; then
            sudo kubectl get all -n "$NAMESPACE" 2>/dev/null || echo "Could not retrieve namespace resources"
        fi
        echo ""
    } > "$OUTPUT_DIR/system-info/system-info.txt"
    set -e 
}

collect_debug_logs() {
    if [ "$INCLUDE_DEBUG" = true ]; then
        print_info "Collecting debug logs..."
        if command -v kubectl &> /dev/null; then
            local pods=$(sudo kubectl get pods -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || echo "")
            for pod in $pods; do
                sudo kubectl logs -n "$NAMESPACE" "$pod" --all-containers=true > "$OUTPUT_DIR/logs/debug-${pod}-all-containers.log" 2>&1 || true
            done
        fi
        if command -v journalctl &> /dev/null; then
            journalctl -u kubelet --since "24 hours ago" > "$OUTPUT_DIR/logs/journalctl-kubelet.log" 2>&1 || true
        fi
    fi
}

create_tarball() {
    local tarball="${OUTPUT_DIR}.tar.gz"
    print_info "Creating support bundle tarball: $tarball"
    tar -czf "$tarball" -C "$(dirname $OUTPUT_DIR)" "$(basename $OUTPUT_DIR)" 2>&1
    if [ $? -eq 0 ]; then
        print_info "Support bundle created successfully: $tarball"
        print_info "Bundle size: $(du -h $tarball | cut -f1)"
        rm -rf "$OUTPUT_DIR"
        echo ""
        echo "================================================================"
        echo "Support bundle location: $tarball"
        echo "================================================================"
    else
        print_error "Failed to create tarball"
        exit 1
    fi
}

main() {
    print_info "Starting vJailbreak log collection"
    echo ""
    print_info "Configuration:"
    echo "  Output directory: $OUTPUT_DIR"
    echo "  Namespace: $NAMESPACE"
    [ -n "$VM_FILTER" ] && echo "  VM filter: $VM_FILTER"
    [ -n "$DATE_FILTER" ] && echo "  Date filter: $DATE_FILTER"
    [ "$INCLUDE_DEBUG" = true ] && echo "  Debug logs: enabled"
    echo ""
    collect_migration_logs
    collect_controller_logs
    collect_version_info
    collect_v2v_helper_info
    collect_configmaps
    collect_system_info
    collect_debug_logs
    create_tarball
    print_info "Log collection completed successfully!"
}

main
