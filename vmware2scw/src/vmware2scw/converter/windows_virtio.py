r"""Inject VirtIO drivers into Windows guest offline.

Ensures all three critical VirtIO drivers are present and registered:
  - viostor.sys  -- block storage (boot-critical)
  - vioscsi.sys  -- SCSI controller
  - netkvm.sys   -- network adapter (required for DHCP/connectivity)

Strategy (layered, belt-and-suspenders):

  Layer 1 - .sys files in System32\drivers\ + Services registry
    Minimum for boot. virt-v2v does this for viostor only.

  Layer 2 - Full driver dirs in C:\Drivers\ + pnputil firstboot script
    On first Windows boot, pnputil /add-driver /install registers drivers
    properly in the DriverStore and DriverDatabase. This is the ONLY
    reliable way to get netkvm working on Windows Server 2019/2022.
    (CriticalDeviceDatabase was removed in modern Windows.)

  Layer 3 - DHCP forcing (registry offline + firstboot script)
    VMware VMs often have static IPs; Scaleway requires DHCP.

Reference: Scaleway official migration doc uses pre-installation of VirtIO
drivers inside VMware before export. This module handles the case where
that was not done.
"""

import logging
import os
import re
import subprocess
import tempfile
from pathlib import Path

logger = logging.getLogger(__name__)

GUESTFS_ENV = {**os.environ, "LIBGUESTFS_BACKEND": "direct"}

# --- Driver definitions ---

DRIVER_DEFS = {
    "viostor": {
        "Group": "SCSI miniport",
        "ImagePath": "system32\\drivers\\viostor.sys",
        "Start": 0, "Type": 1, "ErrorControl": 1, "Tag": 0x40,
        "iso_dir": "viostor",
    },
    "vioscsi": {
        "Group": "SCSI miniport",
        "ImagePath": "system32\\drivers\\vioscsi.sys",
        "Start": 0, "Type": 1, "ErrorControl": 1, "Tag": 0x41,
        "iso_dir": "vioscsi",
    },
    "netkvm": {
        "Group": "NDIS",
        "ImagePath": "system32\\drivers\\netkvm.sys",
        "Start": 0, "Type": 1, "ErrorControl": 1,
        "iso_dir": "NetKVM",
    },
}

# Search order for driver subdirs in virtio-win ISO (newest compatible first)
OS_SUBDIRS = ["2k22/amd64", "2k19/amd64", "2k16/amd64", "w11/amd64", "w10/amd64"]


# --- Helpers ---

def _run(cmd, check=True, **kw):
    logger.debug("  $ %s", " ".join(str(c) for c in cmd[:8]))
    r = subprocess.run(cmd, capture_output=True, text=True, env=GUESTFS_ENV, **kw)
    if check and r.returncode != 0:
        raise RuntimeError(r.stderr.strip()[-500:])
    return r


def _str_to_reg_expand_sz(s):
    """Encode string as REG_EXPAND_SZ hex(2): for .reg files."""
    raw = s.encode("utf-16-le") + b"\x00\x00"
    return "hex(2):" + ",".join(f"{b:02x}" for b in raw)


def _str_to_reg_multi_sz(strings):
    """Encode list of strings as REG_MULTI_SZ hex(7): for .reg files."""
    raw = b""
    for s in strings:
        raw += s.encode("utf-16-le") + b"\x00\x00"
    raw += b"\x00\x00"
    return "hex(7):" + ",".join(f"{b:02x}" for b in raw)


# --- Prerequisite check ---

def ensure_prerequisites():
    """Ensure required host tools are installed."""
    missing = []
    for tool, pkg in [("hivexregedit", "libwin-hivex-perl"), ("guestfish", "libguestfs-tools")]:
        if subprocess.run(["which", tool], capture_output=True).returncode != 0:
            missing.append(pkg)
    if missing:
        logger.info(f"Installing prerequisites: {', '.join(missing)}")
        subprocess.run(["apt-get", "install", "-y", "-qq"] + missing,
                        check=False, capture_output=True)


# --- Driver extraction ---

def _find_driver_dir(mount_dir, drv_name, iso_dir):
    """Find the best matching driver directory in the ISO for a given driver."""
    for subdir in OS_SUBDIRS:
        candidate = mount_dir / iso_dir / subdir
        sys_file = candidate / f"{drv_name}.sys"
        if sys_file.exists():
            return candidate, subdir
    # Broad fallback
    for f in mount_dir.rglob(f"{drv_name}.sys"):
        if "amd64" in str(f).lower():
            return f.parent, str(f.parent.relative_to(mount_dir))
    return None, None


def _extract_drivers_from_iso(iso_path, work_dir):
    """Extract VirtIO driver directories from the ISO.

    Returns dict of {driver_name: {"sys": Path, "dir": Path}}
    where "dir" contains the full driver package (.sys, .inf, .cat, etc.)
    """
    mount_dir = work_dir / "virtio-iso"
    mount_dir.mkdir(parents=True, exist_ok=True)

    try:
        _run(["mount", "-o", "loop,ro", iso_path, str(mount_dir)])
    except RuntimeError as e:
        raise RuntimeError(f"Cannot mount virtio-win ISO {iso_path}: {e}")

    drivers = {}
    try:
        for drv_name, drv_def in DRIVER_DEFS.items():
            iso_dir = drv_def["iso_dir"]
            src_dir, subdir = _find_driver_dir(mount_dir, drv_name, iso_dir)
            if src_dir is None:
                logger.warning(f"  {drv_name}.sys not found in ISO!")
                continue

            # Copy .sys file
            sys_src = src_dir / f"{drv_name}.sys"
            sys_dest = work_dir / f"{drv_name}.sys"
            subprocess.run(["cp", str(sys_src), str(sys_dest)], check=True)

            # Copy entire driver directory (for DriverStore / pnputil)
            dir_dest = work_dir / f"drv_{drv_name}"
            dir_dest.mkdir(parents=True, exist_ok=True)
            subprocess.run(["cp", "-r"] + [str(f) for f in src_dir.iterdir()] + [str(dir_dest)],
                           check=True)

            drivers[drv_name] = {"sys": sys_dest, "dir": dir_dest}
            logger.info(f"  Extracted {drv_name} from {iso_dir}/{subdir} "
                         f"({len(list(dir_dest.iterdir()))} files)")
    finally:
        _run(["umount", str(mount_dir)], check=False)

    return drivers


# --- Check existing drivers ---

def _check_existing_drivers(qcow2_path):
    """Return set of driver names whose .sys file exists in the guest."""
    present = set()
    for drv_name in DRIVER_DEFS:
        r = _run(
            ["guestfish", "--ro", "-a", qcow2_path, "-i", "--",
             "is-file", f"/Windows/System32/drivers/{drv_name}.sys"],
            check=False,
        )
        if r.returncode == 0 and "true" in r.stdout.lower():
            present.add(drv_name)
    return present


# --- Layer 1: Registry Services ---

def _build_driver_reg(drivers_to_register):
    """Build .reg content for driver Services entries only.

    On Windows Server 2019/2022, CriticalDeviceDatabase no longer exists.
    We only register Services entries here. The actual driver<->hardware
    binding will be done by pnputil in the firstboot script (Layer 2).
    """
    lines = ["Windows Registry Editor Version 5.00", ""]

    for cs in ["ControlSet001"]:
        for drv_name in sorted(drivers_to_register):
            drv = DRIVER_DEFS[drv_name]
            base = f"HKEY_LOCAL_MACHINE\\SYSTEM\\{cs}\\Services\\{drv_name}"

            lines.append(f"[{base}]")
            lines.append(f'"Group"="{drv["Group"]}"')
            lines.append(f'"ImagePath"={_str_to_reg_expand_sz(drv["ImagePath"])}')
            lines.append(f'"ErrorControl"=dword:{drv["ErrorControl"]:08x}')
            lines.append(f'"Start"=dword:{drv["Start"]:08x}')
            lines.append(f'"Type"=dword:{drv["Type"]:08x}')
            if "Tag" in drv:
                lines.append(f'"Tag"=dword:{drv["Tag"]:08x}')
            lines.append("")

            # Parent key MUST exist before child for hivexregedit
            lines.append(f"[{base}\\Parameters]")
            lines.append("")

            lines.append(f"[{base}\\Parameters\\PnpInterface]")
            lines.append('"5"=dword:00000001')
            lines.append("")

            lines.append(f"[{base}\\Enum]")
            lines.append('"Count"=dword:00000000')
            lines.append('"NextInstance"=dword:00000000')
            lines.append("")

    return "\n".join(lines)


def _build_dhcp_reg(interface_guids):
    """Build .reg content to force DHCP on all network interfaces."""
    lines = ["Windows Registry Editor Version 5.00", ""]

    for guid in interface_guids:
        base = (f"HKEY_LOCAL_MACHINE\\SYSTEM\\ControlSet001\\Services"
                f"\\Tcpip\\Parameters\\Interfaces\\{guid}")
        lines.append(f"[{base}]")
        lines.append('"EnableDHCP"=dword:00000001')
        lines.append(f'"IPAddress"={_str_to_reg_multi_sz(["0.0.0.0"])}')
        lines.append(f'"SubnetMask"={_str_to_reg_multi_sz(["0.0.0.0"])}')
        lines.append(f'"DefaultGateway"={_str_to_reg_multi_sz([])}')
        lines.append('"NameServer"=""')
        lines.append("")

    return "\n".join(lines)


def _merge_reg_file(qcow2_path, reg_file, work_dir):
    """Merge a .reg file into the guest SYSTEM hive."""
    # Try virt-win-reg first
    r = _run(["virt-win-reg", "--merge", qcow2_path, str(reg_file)], check=False)
    if r.returncode == 0:
        logger.info("  Registry merged via virt-win-reg")
        return

    logger.info("  virt-win-reg failed, falling back to hivexregedit...")

    hive_local = work_dir / "SYSTEM.hive"
    hive_local.unlink(missing_ok=True)

    _run(["guestfish", "-a", qcow2_path, "-i", "--",
          "download", "/Windows/System32/config/SYSTEM", str(hive_local)])

    r2 = _run(["hivexregedit", "--merge", str(hive_local),
               "--prefix", "HKEY_LOCAL_MACHINE\\SYSTEM", str(reg_file)], check=False)

    if r2.returncode != 0:
        # Split into per-section and merge one by one
        logger.warning("  Bulk merge failed, trying section-by-section...")
        reg_text = Path(reg_file).read_text(encoding="utf-8")
        sections = re.split(r'\n(?=\[)', reg_text)

        for i, section in enumerate(sections):
            section = section.strip()
            if not section or not section.startswith("["):
                continue
            part_file = work_dir / f"reg_part_{i:03d}.reg"
            part_file.write_text(
                f"Windows Registry Editor Version 5.00\n\n{section}\n",
                encoding="utf-8")
            r3 = _run(["hivexregedit", "--merge", str(hive_local),
                        "--prefix", "HKEY_LOCAL_MACHINE\\SYSTEM", str(part_file)],
                       check=False)
            if r3.returncode != 0:
                logger.warning(f"  Section {i} failed: {r3.stderr.strip()[:100]}")

    _run(["guestfish", "-a", qcow2_path, "-i", "--",
          "upload", str(hive_local), "/Windows/System32/config/SYSTEM"])
    logger.info("  Registry merged via hivexregedit")


def _get_interface_guids(qcow2_path):
    """Read network interface GUIDs from guest SYSTEM hive."""
    r = _run(
        ["virt-win-reg", qcow2_path,
         "HKLM\\SYSTEM\\ControlSet001\\Services\\Tcpip\\Parameters\\Interfaces"],
        check=False,
    )
    if r.returncode == 0:
        return list(set(re.findall(r'\{[0-9a-fA-F-]+\}', r.stdout)))

    # Fallback via hivexsh
    logger.info("  Trying hivexsh fallback for interface GUIDs...")
    work_dir = Path(tempfile.mkdtemp(prefix="guids-"))
    hive_local = work_dir / "SYSTEM.hive"
    try:
        _run(["guestfish", "--ro", "-a", qcow2_path, "-i", "--",
              "download", "/Windows/System32/config/SYSTEM", str(hive_local)])
        r2 = subprocess.run(
            ["hivexsh", str(hive_local)],
            input="cd \\ControlSet001\\Services\\Tcpip\\Parameters\\Interfaces\nls\n",
            capture_output=True, text=True,
        )
        return list(set(re.findall(r'\{[0-9a-fA-F-]+\}', r2.stdout)))
    except Exception:
        return []


# --- Layer 2: DriverStore + pnputil firstboot ---

PNPUTIL_FIRSTBOOT_SCRIPT = r"""@echo off
echo [%date% %time%] === VirtIO driver installation via pnputil === >> C:\virtio-install.log

REM Install ALL drivers from C:\Drivers staging directory
for /d %%D in (C:\Drivers\*) do (
    echo [%date% %time%] Installing drivers from %%D... >> C:\virtio-install.log
    for %%F in (%%D\*.inf) do (
        echo [%date% %time%]   pnputil /add-driver "%%F" /install >> C:\virtio-install.log
        pnputil /add-driver "%%F" /install >> C:\virtio-install.log 2>&1
        echo [%date% %time%]   Result: !ERRORLEVEL! >> C:\virtio-install.log
    )
)

REM Force DHCP on all adapters
echo [%date% %time%] Configuring DHCP... >> C:\virtio-install.log
powershell -Command "Get-NetAdapter | ForEach-Object { Set-NetIPInterface -InterfaceIndex $_.ifIndex -Dhcp Enabled -ErrorAction SilentlyContinue; Set-DnsClientServerAddress -InterfaceIndex $_.ifIndex -ResetServerAddresses -ErrorAction SilentlyContinue }" >> C:\virtio-install.log 2>&1
netsh interface ip set address name="Ethernet" dhcp >> C:\virtio-install.log 2>&1
netsh interface ip set address name="Ethernet Instance 0" dhcp >> C:\virtio-install.log 2>&1
netsh interface ip set address name="Ethernet 2" dhcp >> C:\virtio-install.log 2>&1
ipconfig /renew >> C:\virtio-install.log 2>&1

echo [%date% %time%] VirtIO driver installation complete >> C:\virtio-install.log
echo [%date% %time%] Rebooting in 15 seconds... >> C:\virtio-install.log
shutdown /r /t 15 /c "VirtIO driver installation complete - rebooting"
"""


def _stage_driver_dirs(qcow2_path, extracted_drivers):
    """Copy full driver directories to C:\\Drivers\\ in the guest image.

    These will be used by the pnputil firstboot script.
    """
    for drv_name, drv_info in extracted_drivers.items():
        drv_dir = drv_info["dir"]
        guest_dir = f"/Drivers/{drv_name}"

        # Create directory and upload all files
        _run(["guestfish", "-a", qcow2_path, "-i", "--",
              "mkdir-p", guest_dir])

        for f in drv_dir.iterdir():
            if f.is_file():
                _run(["guestfish", "-a", qcow2_path, "-i", "--",
                      "upload", str(f), f"{guest_dir}/{f.name}"])

        logger.info(f"  Staged {drv_name} driver package → C:\\Drivers\\{drv_name}\\")


def _inject_pnputil_firstboot(qcow2_path, work_dir):
    """Inject a firstboot script that runs pnputil to install drivers.

    This script runs BEFORE the existing firstboot scripts (prefix 0000).
    It installs all .inf files from C:\\Drivers\\*\\ via pnputil /add-driver /install,
    then forces DHCP and reboots.
    """
    script_file = work_dir / "0000-install-virtio-drivers.bat"
    script_file.write_text(PNPUTIL_FIRSTBOOT_SCRIPT, encoding="utf-8")

    _run(["guestfish", "-a", qcow2_path, "-i", "--",
          "upload", str(script_file),
          "/Program Files/Guestfs/Firstboot/scripts/0000-install-virtio-drivers.bat"])
    logger.info("  Injected pnputil firstboot script (0000-install-virtio-drivers.bat)")


# --- Main entry point ---

def ensure_all_virtio_drivers(qcow2_path, virtio_iso, work_dir=None):
    """Ensure all VirtIO drivers are installed and registered in a Windows guest.

    Layered approach:
      Layer 1: .sys in drivers/ + Services registry (allows boot)
      Layer 2: Full driver dirs in C:\\Drivers + pnputil firstboot (proper PnP binding)
      Layer 3: DHCP forcing (registry offline + firstboot)

    Safe to call after virt-v2v (idempotent for already-present drivers).
    """
    if work_dir is None:
        work_dir = Path(tempfile.mkdtemp(prefix="virtio-"))
    work_dir = Path(work_dir)
    work_dir.mkdir(parents=True, exist_ok=True)

    ensure_prerequisites()

    logger.info("=== Ensuring all VirtIO drivers for Windows guest ===")

    # Step 1: Check existing .sys files
    logger.info("Step 1: Checking existing drivers in guest...")
    existing = _check_existing_drivers(qcow2_path)
    all_needed = set(DRIVER_DEFS.keys())
    missing = all_needed - existing

    if existing:
        logger.info(f"  Present: {', '.join(sorted(existing))}")
    if missing:
        logger.info(f"  Missing: {', '.join(sorted(missing))}")

    # Step 2: Extract ALL drivers from ISO (even present ones — we need full dirs)
    logger.info("Step 2: Extracting drivers from virtio-win ISO...")
    extracted = _extract_drivers_from_iso(virtio_iso, work_dir)

    # Step 3: Upload missing .sys files to System32\drivers (Layer 1)
    if missing:
        logger.info("Step 3: Uploading missing .sys files...")
        for drv_name in missing:
            if drv_name not in extracted:
                continue
            _run(["guestfish", "-a", qcow2_path, "-i", "--",
                  "upload", str(extracted[drv_name]["sys"]),
                  f"/Windows/System32/drivers/{drv_name}.sys"])
            logger.info(f"  Uploaded {drv_name}.sys")
            existing.add(drv_name)
    else:
        logger.info("Step 3: All .sys files already present")

    # Step 4: Register Services in registry (Layer 1)
    logger.info("Step 4: Registering driver services in Windows registry...")
    reg_content = _build_driver_reg(existing)
    reg_file = work_dir / "virtio-drivers.reg"
    reg_file.write_text(reg_content, encoding="utf-8")
    _merge_reg_file(qcow2_path, reg_file, work_dir)
    logger.info(f"  Registered services: {', '.join(sorted(existing))}")

    # Step 5: Stage full driver directories in C:\Drivers (Layer 2)
    logger.info("Step 5: Staging driver packages in C:\\Drivers\\...")
    _stage_driver_dirs(qcow2_path, extracted)

    # Step 6: Inject pnputil firstboot script (Layer 2)
    logger.info("Step 6: Injecting pnputil firstboot script...")
    _inject_pnputil_firstboot(qcow2_path, work_dir)

    # Step 7: Force DHCP offline (Layer 3)
    logger.info("Step 7: Forcing DHCP on all network interfaces...")
    guids = _get_interface_guids(qcow2_path)
    if guids:
        logger.info(f"  Found {len(guids)} interface(s)")
        dhcp_content = _build_dhcp_reg(guids)
        dhcp_file = work_dir / "dhcp-fix.reg"
        dhcp_file.write_text(dhcp_content, encoding="utf-8")
        _merge_reg_file(qcow2_path, dhcp_file, work_dir)
        logger.info("  DHCP forced on all interfaces")
    else:
        logger.warning("  No interfaces found (DHCP will rely on firstboot script)")

    logger.info("=== VirtIO driver setup complete ===")
    logger.info("  Boot sequence: Windows loads viostor (boot) → rhsrvany runs "
                "pnputil script → netkvm installed → DHCP → network up → reboot")
    return True


# Backwards-compatible alias
inject_virtio_windows = ensure_all_virtio_drivers
