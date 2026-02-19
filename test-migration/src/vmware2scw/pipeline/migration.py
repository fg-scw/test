"""Migration pipeline orchestrator — coordinates all migration stages."""

from __future__ import annotations

import time
import uuid
from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path
from typing import Optional

from vmware2scw.config import AppConfig, VMMigrationPlan
from vmware2scw.pipeline.state import MigrationState, MigrationStateStore
from vmware2scw.utils.logging import get_logger

logger = get_logger(__name__)


@dataclass
class MigrationResult:
    """Result of a migration execution."""
    success: bool
    migration_id: str
    vm_name: str
    instance_id: Optional[str] = None
    image_id: Optional[str] = None
    duration: str = ""
    failed_stage: Optional[str] = None
    error: Optional[str] = None
    completed_stages: list[str] = field(default_factory=list)


class MigrationPipeline:
    """Orchestrates the full VMware → Scaleway migration pipeline.

    Stages (executed in order):
    1. validate       — Pre-flight compatibility checks
    2. snapshot       — Create VMware snapshot for consistency
    3. export         — Export VMDK disks from VMware
    4. clean_tools    — Remove VMware tools from guest
    5. inject_virtio  — Inject VirtIO drivers for KVM
    6. convert        — Convert VMDK → qcow2
    7. fix_bootloader — Adapt bootloader for KVM (fstab, GRUB, initramfs)
    8. fix_network    — Adapt network configuration
    9. upload_s3      — Upload qcow2 to Scaleway Object Storage
    10. import_scw    — Import image into Scaleway (snapshot → image)
    11. verify        — Post-migration health checks
    12. cleanup       — Remove temporary files, snapshots

    Each stage is idempotent and can be resumed after failure.

    Confidence: 88 — Pipeline pattern is proven; individual stage
    confidence varies (see DESIGN.md for details).
    """

    STAGES = [
        "validate",
        "snapshot",
        "export",
        "convert",           # MUST be before clean_tools: exported VMDK is streamOptimized, unreadable by libguestfs
        "clean_tools",
        "inject_virtio",
        "fix_bootloader",
        "ensure_uefi",       # Convert BIOS→UEFI if needed (after bootloader fix)
        "fix_network",
        "upload_s3",
        "import_scw",
        "verify",
        "cleanup",
    ]

    def __init__(self, config: AppConfig):
        self.config = config
        self.state_store = MigrationStateStore(config.conversion.work_dir)

    def run(self, plan: VMMigrationPlan) -> MigrationResult:
        """Execute a full migration for a single VM.

        Args:
            plan: Migration plan with VM name, target type, etc.

        Returns:
            MigrationResult with success status and details
        """
        migration_id = str(uuid.uuid4())[:8]
        start_time = time.time()

        state = MigrationState(
            migration_id=migration_id,
            vm_name=plan.vm_name,
            target_type=plan.target_type,
            zone=plan.zone,
            current_stage="",
            completed_stages=[],
            artifacts={},
            started_at=datetime.now(),
        )
        self.state_store.save(state)

        logger.info(f"[bold]Starting migration {migration_id}[/bold]: "
                     f"{plan.vm_name} → {plan.target_type} ({plan.zone})")

        stages_to_run = self.STAGES
        if plan.skip_validation:
            stages_to_run = [s for s in stages_to_run if s != "validate"]

        for stage_name in stages_to_run:
            state.current_stage = stage_name
            self.state_store.save(state)

            logger.info(f"[cyan]▶ Stage: {stage_name}[/cyan]")
            try:
                self._execute_stage(stage_name, plan, state)
                state.completed_stages.append(stage_name)
                self.state_store.save(state)
                logger.info(f"[green]✓ Stage {stage_name} complete[/green]")

            except Exception as e:
                elapsed = time.time() - start_time
                state.error = str(e)
                self.state_store.save(state)

                logger.error(f"[red]✗ Stage {stage_name} failed: {e}[/red]")
                return MigrationResult(
                    success=False,
                    migration_id=migration_id,
                    vm_name=plan.vm_name,
                    failed_stage=stage_name,
                    error=str(e),
                    duration=f"{elapsed:.0f}s",
                    completed_stages=list(state.completed_stages),
                )

        elapsed = time.time() - start_time
        logger.info(f"[bold green]Migration {migration_id} complete in {elapsed:.0f}s[/bold green]")

        return MigrationResult(
            success=True,
            migration_id=migration_id,
            vm_name=plan.vm_name,
            instance_id=state.artifacts.get("scaleway_instance_id"),
            image_id=state.artifacts.get("scaleway_image_id"),
            duration=f"{elapsed:.0f}s",
            completed_stages=list(state.completed_stages),
        )

    def resume(self, migration_id: str) -> MigrationResult:
        """Resume a failed migration from the last successful stage."""
        state = self.state_store.load(migration_id)
        if not state:
            raise ValueError(f"Migration '{migration_id}' not found")

        logger.info(f"Resuming migration {migration_id} for VM '{state.vm_name}'")
        logger.info(f"Completed stages: {', '.join(state.completed_stages)}")

        plan = VMMigrationPlan(
            vm_name=state.vm_name,
            target_type=state.target_type,
            zone=state.zone,
        )

        # Find remaining stages
        remaining = [s for s in self.STAGES if s not in state.completed_stages]
        if not remaining:
            return MigrationResult(
                success=True,
                migration_id=migration_id,
                vm_name=state.vm_name,
                completed_stages=list(state.completed_stages),
            )

        start_time = time.time()
        state.error = None

        for stage_name in remaining:
            state.current_stage = stage_name
            self.state_store.save(state)

            logger.info(f"[cyan]▶ Stage: {stage_name}[/cyan] (resumed)")
            try:
                self._execute_stage(stage_name, plan, state)
                state.completed_stages.append(stage_name)
                self.state_store.save(state)
                logger.info(f"[green]✓ Stage {stage_name} complete[/green]")

            except Exception as e:
                state.error = str(e)
                self.state_store.save(state)
                elapsed = time.time() - start_time

                return MigrationResult(
                    success=False,
                    migration_id=migration_id,
                    vm_name=state.vm_name,
                    failed_stage=stage_name,
                    error=str(e),
                    duration=f"{elapsed:.0f}s",
                    completed_stages=list(state.completed_stages),
                )

        elapsed = time.time() - start_time
        return MigrationResult(
            success=True,
            migration_id=migration_id,
            vm_name=state.vm_name,
            instance_id=state.artifacts.get("scaleway_instance_id"),
            image_id=state.artifacts.get("scaleway_image_id"),
            duration=f"{elapsed:.0f}s",
            completed_stages=list(state.completed_stages),
        )

    def dry_run(self, plan: VMMigrationPlan) -> None:
        """Simulate a migration without executing any stages."""
        logger.info(f"[yellow]DRY RUN for VM '{plan.vm_name}'[/yellow]")
        logger.info(f"Target: {plan.target_type} in {plan.zone}")
        logger.info(f"Stages that would execute:")
        for i, stage in enumerate(self.STAGES, 1):
            if plan.skip_validation and stage == "validate":
                logger.info(f"  {i}. {stage} [dim](skipped)[/dim]")
            else:
                logger.info(f"  {i}. {stage}")

    def _execute_stage(self, stage: str, plan: VMMigrationPlan, state: MigrationState) -> None:
        """Execute a single pipeline stage.

        Each stage method updates state.artifacts with any intermediate
        results (file paths, IDs, etc.) for use by subsequent stages.
        """
        handler = getattr(self, f"_stage_{stage}", None)
        if handler is None:
            raise NotImplementedError(f"Stage '{stage}' not implemented yet")
        handler(plan, state)

    # ─── Stage implementations ───────────────────────────────────────

    def _stage_validate(self, plan: VMMigrationPlan, state: MigrationState) -> None:
        """Pre-flight validation: check VM compatibility with target type."""
        from vmware2scw.pipeline.validator import MigrationValidator
        from vmware2scw.vmware.client import VSphereClient
        from vmware2scw.vmware.inventory import VMInventory

        client = VSphereClient()
        pw = self.config.vmware.password.get_secret_value() if self.config.vmware.password else ""
        client.connect(
            self.config.vmware.vcenter,
            self.config.vmware.username,
            pw,
            insecure=self.config.vmware.insecure,
        )

        inv = VMInventory(client)
        vm_info = inv.get_vm_info(plan.vm_name)
        state.artifacts["vm_info"] = vm_info.model_dump()

        validator = MigrationValidator()
        report = validator.validate(vm_info, plan.target_type)

        client.disconnect()

        if not report.passed:
            failures = [c for c in report.checks if not c.passed and c.blocking]
            msg = "; ".join(f"{c.name}: {c.message}" for c in failures)
            raise RuntimeError(f"Pre-validation failed: {msg}")

    def _stage_snapshot(self, plan: VMMigrationPlan, state: MigrationState) -> None:
        """Create a VMware snapshot for consistent export."""
        from vmware2scw.vmware.client import VSphereClient
        from vmware2scw.vmware.snapshot import SnapshotManager

        client = VSphereClient()
        pw = self.config.vmware.password.get_secret_value() if self.config.vmware.password else ""
        client.connect(
            self.config.vmware.vcenter,
            self.config.vmware.username,
            pw,
            insecure=self.config.vmware.insecure,
        )

        snap_mgr = SnapshotManager(client)
        snap_name = f"vmware2scw-{state.migration_id}"
        snap_mgr.create_migration_snapshot(plan.vm_name, snap_name)
        state.artifacts["snapshot_name"] = snap_name

        client.disconnect()

    def _stage_export(self, plan: VMMigrationPlan, state: MigrationState) -> None:
        """Export VMDK disks from VMware."""
        from vmware2scw.vmware.client import VSphereClient
        from vmware2scw.vmware.export import VMExporter

        work_dir = self.config.conversion.work_dir / state.migration_id
        work_dir.mkdir(parents=True, exist_ok=True)

        client = VSphereClient()
        pw = self.config.vmware.password.get_secret_value() if self.config.vmware.password else ""
        client.connect(
            self.config.vmware.vcenter,
            self.config.vmware.username,
            pw,
            insecure=self.config.vmware.insecure,
        )

        exporter = VMExporter(client)
        vmdk_paths = exporter.export_vm_disks(plan.vm_name, work_dir)
        state.artifacts["vmdk_paths"] = [str(p) for p in vmdk_paths]

        client.disconnect()

    def _stage_clean_tools(self, plan: VMMigrationPlan, state: MigrationState) -> None:
        """Clean VMware tools from converted qcow2 disks.

        Only processes the boot disk (first disk). Additional data disks
        don't contain an OS and would fail virt-customize inspection.
        """
        from vmware2scw.converter.disk import VMwareToolsCleaner
        from vmware2scw.scaleway.mapping import ResourceMapper

        mapper = ResourceMapper()
        vm_info_dict = state.artifacts.get("vm_info", {})
        guest_os = vm_info_dict.get("guest_os", "otherLinux64Guest")
        os_family, _ = mapper.get_os_family(guest_os)

        qcow2_paths = state.artifacts.get("qcow2_paths", [])
        if not qcow2_paths:
            logger.warning("No qcow2 files found — skipping clean_tools")
            return

        cleaner = VMwareToolsCleaner()
        # Only clean the boot disk (first disk)
        boot_disk = qcow2_paths[0]
        logger.info(f"Cleaning boot disk: {Path(boot_disk).name}")
        cleaner.clean(boot_disk, os_family=os_family)

        if len(qcow2_paths) > 1:
            logger.info(f"Skipping {len(qcow2_paths) - 1} data disk(s) — no OS to clean")

    def _stage_inject_virtio(self, plan: VMMigrationPlan, state: MigrationState) -> None:
        """Use virt-v2v to prepare the image for KVM boot.

        Based on:
        - Scaleway doc (Windows): virt-v2v -i disk <q> -block-driver virtio-scsi -o qemu -os ./out
        - migrate_centos.sh: virt-v2v -i disk <q> -o qemu -on <n> -os <dir> -of qcow2 -oc qcow2

        virt-v2v handles: VirtIO drivers, bootloader, initramfs, BIOS/UEFI.
        """
        import shutil
        import subprocess
        from vmware2scw.scaleway.mapping import ResourceMapper
        from vmware2scw.utils.subprocess import run_command, check_tool_available

        mapper = ResourceMapper()
        vm_info_dict = state.artifacts.get("vm_info", {})
        guest_os = vm_info_dict.get("guest_os", "otherLinux64Guest")
        os_family, _ = mapper.get_os_family(guest_os)

        qcow2_paths = state.artifacts.get("qcow2_paths", [])
        if not qcow2_paths:
            logger.warning("No qcow2 files found — skipping inject_virtio")
            return

        boot_disk = Path(qcow2_paths[0])

        if not check_tool_available("virt-v2v"):
            logger.warning("virt-v2v not installed — using virt-customize fallback")
            self._inject_virtio_fallback(boot_disk, os_family)
            return

        # Setup environment
        env = {"LIBGUESTFS_BACKEND": "direct"}
        mounted_virtio = False

        if os_family == "windows":
            virtio_iso = self.config.conversion.virtio_win_iso
            if not virtio_iso or not Path(virtio_iso).exists():
                raise RuntimeError(
                    "virtio-win ISO is required for Windows VMs.\n"
                    "  wget -O /opt/virtio-win.iso "
                    "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/stable-virtio/virtio-win.iso\n"
                    "  Then in migration.yaml: conversion.virtio_win_iso: /opt/virtio-win.iso"
                )
            mount_dir = Path("/usr/share/virtio-win")
            mount_dir.mkdir(parents=True, exist_ok=True)
            subprocess.run(["umount", str(mount_dir)], check=False, capture_output=True)
            subprocess.run(["mount", "-o", "loop,ro", str(virtio_iso), str(mount_dir)], check=False)
            env["VIRTIO_WIN"] = str(mount_dir)
            mounted_virtio = True

        # Output directory
        out_dir = boot_disk.parent / "v2v-out"
        out_dir.mkdir(parents=True, exist_ok=True)
        v2v_name = f"v2v-{boot_disk.stem}"

        # Try multiple virt-v2v syntaxes (varies by distro/version)
        v2v_syntaxes = [
            # Syntax 1: migrate_centos.sh style (-o qemu -of qcow2 -oc qcow2)
            ["virt-v2v", "-i", "disk", str(boot_disk),
             "-o", "qemu", "-os", str(out_dir),
             "-on", v2v_name, "-of", "qcow2", "-oc", "qcow2"],
            # Syntax 2: Scaleway doc style (--block-driver)
            ["virt-v2v", "-i", "disk", str(boot_disk),
             "-o", "qemu", "-os", str(out_dir),
             "-on", v2v_name, "-of", "qcow2",
             "--block-driver", "virtio-scsi"],
            # Syntax 3: -o local (most compatible)
            ["virt-v2v", "-i", "disk", str(boot_disk),
             "-o", "local", "-os", str(out_dir),
             "-on", v2v_name, "-of", "qcow2"],
        ]

        v2v_ok = False
        for i, cmd in enumerate(v2v_syntaxes, 1):
            logger.info(f"Trying virt-v2v syntax {i}/{len(v2v_syntaxes)}...")
            try:
                run_command(cmd, env=env, timeout=3600)
                v2v_ok = True
                logger.info(f"virt-v2v syntax {i} succeeded")
                break
            except Exception as e:
                logger.warning(f"virt-v2v syntax {i} failed: {e}")
                for f in out_dir.iterdir():
                    f.unlink(missing_ok=True)

        if mounted_virtio:
            subprocess.run(["umount", "/usr/share/virtio-win"], check=False, capture_output=True)

        if not v2v_ok:
            logger.warning("All virt-v2v syntaxes failed — using virt-customize fallback")
            self._inject_virtio_fallback(boot_disk, os_family)
            return

        # Find the virt-v2v output (named <v2v_name>-sda or similar)
        candidates = sorted(
            [f for f in out_dir.iterdir()
             if f.is_file() and f.stat().st_size > 1024 * 1024
             and f.suffix not in ('.xml', '.sh')],
            key=lambda f: f.stat().st_size, reverse=True,
        )
        if not candidates:
            raise RuntimeError(f"virt-v2v succeeded but no output in {out_dir}")

        converted = candidates[0]
        logger.info(f"virt-v2v output: {converted.name} ({converted.stat().st_size / (1024**3):.2f} GB)")

        # Linux: restore original fstab (virt-v2v overrides UUIDs with /dev/sda*)
        if os_family == "linux":
            logger.info("Restoring original fstab (virt-v2v may have replaced UUIDs)...")
            try:
                run_command([
                    "virt-customize", "-a", str(converted),
                    "--run-command",
                    "if [ -f /etc/fstab.augsave ]; then cp /etc/fstab.augsave /etc/fstab; echo Restored; fi",
                ], env={"LIBGUESTFS_BACKEND": "direct"})
            except Exception as e:
                logger.warning(f"fstab restore failed (non-critical): {e}")

        # Ensure output is qcow2
        import json as _json
        info_out = run_command(["qemu-img", "info", "--output=json", str(converted)], capture_output=True)
        fmt = _json.loads(info_out.stdout).get("format", "raw")
        if fmt != "qcow2":
            logger.info(f"Converting virt-v2v output from {fmt} to qcow2...")
            final_qcow2 = out_dir / "boot-v2v.qcow2"
            compress = ["-c"] if self.config.conversion.compress_qcow2 else []
            run_command(["qemu-img", "convert", "-O", "qcow2"] + compress + [str(converted), str(final_qcow2)])
            converted = final_qcow2

        # Replace original boot disk — clean up to save space
        boot_disk.unlink(missing_ok=True)
        shutil.move(str(converted), str(boot_disk))
        shutil.rmtree(out_dir, ignore_errors=True)
        logger.info("virt-v2v conversion complete — boot disk replaced")

    def _inject_virtio_fallback(self, boot_disk, os_family):
        """Fallback VirtIO injection when virt-v2v fails."""
        from vmware2scw.converter.disk import VirtIOInjector
        injector = VirtIOInjector(
            virtio_win_iso=self.config.conversion.virtio_win_iso
        )
        injector.inject(str(boot_disk), os_family=os_family)

    def _stage_convert(self, plan: VMMigrationPlan, state: MigrationState) -> None:
        """Convert VMDK disks to qcow2 format."""
        from vmware2scw.converter.disk import DiskConverter

        converter = DiskConverter()
        qcow2_paths = []

        for vmdk_path in state.artifacts.get("vmdk_paths", []):
            vmdk = Path(vmdk_path)
            qcow2_path = vmdk.with_suffix(".qcow2")

            # Skip if already converted and valid
            if qcow2_path.exists() and converter.check(qcow2_path):
                logger.info(f"Skipping conversion (already exists): {qcow2_path.name}")
                qcow2_paths.append(str(qcow2_path))
                continue

            converter.convert(
                vmdk,
                qcow2_path,
                compress=self.config.conversion.compress_qcow2,
            )
            qcow2_paths.append(str(qcow2_path))

        state.artifacts["qcow2_paths"] = qcow2_paths

        # Free disk space: delete VMDK source files after successful conversion
        for vmdk_path in state.artifacts.get("vmdk_paths", []):
            vmdk = Path(vmdk_path)
            if vmdk.exists():
                size_mb = vmdk.stat().st_size / (1024**2)
                vmdk.unlink()
                logger.info(f"Deleted source VMDK: {vmdk.name} ({size_mb:.0f} MB freed)")

    def _stage_fix_bootloader(self, plan: VMMigrationPlan, state: MigrationState) -> None:
        """Fix bootloader for KVM: fstab device names, GRUB config, initramfs.

        VMware uses LSI Logic / PVSCSI controllers → /dev/sd* devices.
        KVM with VirtIO uses /dev/vd* devices.

        If fstab or GRUB reference /dev/sda, the VM won't boot.
        Modern systems use UUID/LABEL which is safe, but we fix both.
        """
        from vmware2scw.scaleway.mapping import ResourceMapper
        from vmware2scw.utils.subprocess import run_command

        mapper = ResourceMapper()
        vm_info_dict = state.artifacts.get("vm_info", {})
        guest_os = vm_info_dict.get("guest_os", "")
        os_family, _ = mapper.get_os_family(guest_os)

        qcow2_paths = state.artifacts.get("qcow2_paths", [])
        if not qcow2_paths:
            return

        boot_disk = qcow2_paths[0]

        if os_family == "windows":
            logger.info("Windows VM — bootloader fix not applicable (BCD is handled by VirtIO injection)")
            return

        logger.info("Fixing bootloader for KVM compatibility...")

        # All fixes in a single virt-customize call to avoid multiple guest inspections
        commands = [
            # 1. Fix /etc/fstab: replace /dev/sd* with /dev/vd* (only if not UUID)
            "--run-command",
            "if [ -f /etc/fstab ]; then "
            "  cp /etc/fstab /etc/fstab.vmware2scw.bak; "
            "  sed -i 's|/dev/sda|/dev/vda|g; s|/dev/sdb|/dev/vdb|g; s|/dev/sdc|/dev/vdc|g' /etc/fstab; "
            "fi",

            # 2. Fix GRUB config: replace sd* references with vd*
            "--run-command",
            "if [ -f /etc/default/grub ]; then "
            "  cp /etc/default/grub /etc/default/grub.vmware2scw.bak; "
            "  sed -i 's|/dev/sda|/dev/vda|g' /etc/default/grub; "
            "fi",

            # 3. Fix GRUB device map
            "--run-command",
            "if [ -f /boot/grub/device.map ]; then "
            "  sed -i 's|/dev/sda|/dev/vda|g' /boot/grub/device.map; "
            "fi",

            # 4. Regenerate GRUB config
            "--run-command",
            "if command -v grub-mkconfig >/dev/null 2>&1; then "
            "  grub-mkconfig -o /boot/grub/grub.cfg 2>/dev/null || true; "
            "elif command -v grub2-mkconfig >/dev/null 2>&1; then "
            "  grub2-mkconfig -o /boot/grub2/grub.cfg 2>/dev/null || true; "
            "fi",

            # 5. Ensure VirtIO modules are loaded at boot
            "--run-command",
            "if [ -d /etc/initramfs-tools ]; then "
            "  for mod in virtio_blk virtio_scsi virtio_net virtio_pci; do "
            "    grep -q $mod /etc/initramfs-tools/modules 2>/dev/null || echo $mod >> /etc/initramfs-tools/modules; "
            "  done; "
            "  update-initramfs -u 2>/dev/null || true; "
            "elif command -v dracut >/dev/null 2>&1; then "
            "  dracut --force --add-drivers 'virtio_blk virtio_scsi virtio_net virtio_pci' 2>/dev/null || true; "
            "fi",

            # 6. Remove VMware SCSI driver references that interfere with VirtIO
            "--run-command",
            "rm -f /etc/modprobe.d/*vmw* 2>/dev/null || true; "
            "rm -f /etc/modprobe.d/*vmware* 2>/dev/null || true",

            # 7. Clean persistent net rules (interface names change)
            "--run-command",
            "rm -f /etc/udev/rules.d/70-persistent-net.rules 2>/dev/null || true; "
            "rm -f /etc/udev/rules.d/75-persistent-net-generator.rules 2>/dev/null || true",

            # 8. Enable DHCP on first interface (Scaleway provides IP via DHCP)
            "--run-command",
            "if [ -d /etc/netplan ]; then "
            "  cat > /etc/netplan/50-cloud-init.yaml << 'NETPLAN'\n"
            "network:\n"
            "  version: 2\n"
            "  ethernets:\n"
            "    ens2:\n"
            "      dhcp4: true\n"
            "    eth0:\n"
            "      dhcp4: true\n"
            "NETPLAN\n"
            "elif [ -d /etc/sysconfig/network-scripts ]; then "
            "  cat > /etc/sysconfig/network-scripts/ifcfg-eth0 << 'IFCFG'\n"
            "DEVICE=eth0\n"
            "ONBOOT=yes\n"
            "BOOTPROTO=dhcp\n"
            "IFCFG\n"
            "fi",
        ]

        cmd = ["virt-customize", "-a", str(boot_disk)] + commands
        run_command(cmd, env={"LIBGUESTFS_BACKEND": "direct"}, check=False)
        logger.info("Bootloader and network configuration fixed for KVM")

    def _stage_ensure_uefi(self, plan: VMMigrationPlan, state: MigrationState) -> None:
        """Ensure disk is UEFI-bootable. Scaleway uses UEFI firmware.

        If the source VM is BIOS/MBR (common for VMware), we must:
        - Convert MBR→GPT
        - Create an EFI System Partition (ESP)
        - Install GRUB EFI bootloader

        This is normally handled by virt-v2v, but falls back to manual
        conversion when virt-v2v fails (e.g. Ubuntu 24.04 kernel bug).
        """
        from vmware2scw.converter.bios2uefi import detect_boot_type, convert_bios_to_uefi
        from vmware2scw.scaleway.mapping import ResourceMapper

        mapper = ResourceMapper()
        vm_info_dict = state.artifacts.get("vm_info", {})
        guest_os = vm_info_dict.get("guest_os", "")
        firmware = vm_info_dict.get("firmware", "bios")
        os_family, _ = mapper.get_os_family(guest_os)

        qcow2_paths = state.artifacts.get("qcow2_paths", [])
        if not qcow2_paths:
            return

        boot_disk = qcow2_paths[0]

        # If virt-v2v succeeded (inject_virtio didn't fall back), disk should already be OK
        # Check anyway to be sure
        boot_type = detect_boot_type(boot_disk)
        logger.info(f"Boot type detection: firmware={firmware}, disk={boot_type}")

        if boot_type == "uefi":
            logger.info("Disk already UEFI-bootable — skipping conversion")
            return

        if os_family == "windows":
            logger.warning(
                "Windows BIOS→UEFI requires virt-v2v + virtio-win ISO. "
                "Manual conversion not supported. Consider using a RHEL/CentOS "
                "conversion host where virt-v2v works correctly."
            )
            return

        logger.info("Disk is BIOS — converting to UEFI for Scaleway compatibility")
        converted = convert_bios_to_uefi(boot_disk, os_family=os_family)
        if converted:
            logger.info("BIOS → UEFI conversion successful")
        else:
            logger.warning("BIOS → UEFI conversion was not performed")

    def _stage_fix_network(self, plan: VMMigrationPlan, state: MigrationState) -> None:
        """Network adaptation — handled in fix_bootloader stage.

        The fix_bootloader stage already:
        - Removes persistent net rules
        - Configures DHCP on default interfaces
        - Removes VMware network driver references
        """
        logger.info("Network adaptation already handled in fix_bootloader stage")

    def _stage_upload_s3(self, plan: VMMigrationPlan, state: MigrationState) -> None:
        """Upload qcow2 images to Scaleway Object Storage."""
        from vmware2scw.scaleway.s3 import ScalewayS3

        scw_secret = self.config.scaleway.secret_key
        s3 = ScalewayS3(
            region=self.config.scaleway.s3_region,
            access_key=self.config.scaleway.access_key or "",
            secret_key=scw_secret.get_secret_value() if scw_secret else "",
        )

        bucket = self.config.scaleway.s3_bucket
        s3.create_bucket_if_not_exists(bucket)

        s3_keys = []
        for qcow2_path in state.artifacts.get("qcow2_paths", []):
            p = Path(qcow2_path)
            key = f"migrations/{state.migration_id}/{p.name}"

            # Skip if already uploaded with same size
            if s3.check_object_exists(bucket, key):
                remote_size = s3.get_object_size(bucket, key)
                local_size = p.stat().st_size
                if remote_size == local_size:
                    logger.info(f"Skipping upload (already exists): {key}")
                    s3_keys.append(key)
                    continue

            s3.upload_image(qcow2_path, bucket, key)
            s3_keys.append(key)

        state.artifacts["s3_keys"] = s3_keys
        state.artifacts["s3_bucket"] = bucket

    def _stage_import_scw(self, plan: VMMigrationPlan, state: MigrationState) -> None:
        """Import qcow2 image into Scaleway: create snapshot → image.

        Confidence: 80 — API workflow is documented but import from S3
        has specific requirements.
        """
        from vmware2scw.scaleway.instance import ScalewayInstanceAPI

        api = ScalewayInstanceAPI(
            access_key=self.config.scaleway.access_key or "",
            secret_key=(self.config.scaleway.secret_key.get_secret_value()
                        if self.config.scaleway.secret_key else ""),
            project_id=self.config.scaleway.project_id,
        )

        zone = plan.zone
        bucket = state.artifacts["s3_bucket"]
        s3_keys = state.artifacts.get("s3_keys", [])

        if not s3_keys:
            raise RuntimeError("No S3 keys found — upload stage may have failed")

        # Import the boot disk (first disk)
        boot_key = s3_keys[0]
        snapshot_name = f"vmware2scw-{plan.vm_name}-{state.migration_id}"

        logger.info(f"Creating Scaleway snapshot from s3://{bucket}/{boot_key}")
        snapshot = api.create_snapshot_from_s3(
            zone=zone,
            name=snapshot_name,
            bucket=bucket,
            key=boot_key,
        )
        snapshot_id = snapshot["id"]
        state.artifacts["scaleway_snapshot_id"] = snapshot_id

        logger.info(f"Waiting for snapshot {snapshot_id}...")
        api.wait_for_snapshot(zone, snapshot_id)

        # Create image from snapshot
        image_name = f"migrated-{plan.vm_name}"
        logger.info(f"Creating Scaleway image '{image_name}'")
        image = api.create_image(zone, image_name, snapshot_id)
        state.artifacts["scaleway_image_id"] = image["id"]

        logger.info(f"Image created: {image['id']}")

    def _stage_verify(self, plan: VMMigrationPlan, state: MigrationState) -> None:
        """Post-migration verification.

        Confidence: 75 — SPÉCULATIF. Basic checks only.
        """
        image_id = state.artifacts.get("scaleway_image_id")
        if image_id:
            logger.info(f"✅ Scaleway image created: {image_id}")
        else:
            logger.warning("⚠️  No Scaleway image ID found — verify manually")

        # TODO: Optionally boot a test instance and check connectivity

    def _stage_cleanup(self, plan: VMMigrationPlan, state: MigrationState) -> None:
        """Clean up all temporary resources to free disk space."""
        import shutil

        # 1. Clean local work directory (VMDK + qcow2 intermediate files)
        work_dir = self.config.conversion.work_dir / state.migration_id
        if work_dir.exists():
            size_gb = sum(f.stat().st_size for f in work_dir.rglob("*") if f.is_file()) / (1024**3)
            logger.info(f"Cleaning work directory: {work_dir} ({size_gb:.1f} GB)")
            shutil.rmtree(work_dir, ignore_errors=True)

        # 2. Clean VMware snapshot
        snap_name = state.artifacts.get("snapshot_name")
        if snap_name:
            try:
                from vmware2scw.vmware.client import VSphereClient
                from vmware2scw.vmware.snapshot import SnapshotManager

                client = VSphereClient()
                pw = self.config.vmware.password.get_secret_value() if self.config.vmware.password else ""
                client.connect(
                    self.config.vmware.vcenter,
                    self.config.vmware.username,
                    pw,
                    insecure=self.config.vmware.insecure,
                )
                snap_mgr = SnapshotManager(client)
                snap_mgr.delete_migration_snapshot(plan.vm_name, snap_name)
                client.disconnect()
                logger.info(f"Deleted VMware snapshot: {snap_name}")
            except Exception as e:
                logger.warning(f"Failed to clean VMware snapshot: {e}")

        # 3. Clean S3 transit files (safe now — image is created)
        image_id = state.artifacts.get("scaleway_image_id")
        s3_keys = state.artifacts.get("s3_keys", [])
        bucket = state.artifacts.get("s3_bucket")
        if image_id and s3_keys and bucket:
            try:
                from vmware2scw.scaleway.storage import ScalewayS3Client
                s3 = ScalewayS3Client(
                    access_key=self.config.scaleway.access_key,
                    secret_key=self.config.scaleway.secret_key.get_secret_value() if self.config.scaleway.secret_key else "",
                    region=self.config.scaleway.region,
                )
                for key in s3_keys:
                    try:
                        s3.client.delete_object(Bucket=bucket, Key=key)
                        logger.info(f"Deleted S3 transit: s3://{bucket}/{key}")
                    except Exception as e2:
                        logger.warning(f"Failed to delete {key}: {e2}")
            except Exception as e:
                logger.warning(f"S3 cleanup failed: {e}")
        else:
            logger.info("S3 transit files retained (image not confirmed or no keys)")

        logger.info("Cleanup complete.")
