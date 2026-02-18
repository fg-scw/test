#!/bin/sh
# detect-bootdisk.sh
# Prints the correct BIOS boot disk (the one containing GRUB MBR)
# Works with LVM and /boot on a separate disk.
# Assumes we are running inside a guestfish-mounted environment

# Step 1: Identify all disks in guest
# List block devices (assuming /dev/vda, /dev/sda, etc. are available)
disks=$(ls /dev/[sv]d[a-z] 2>/dev/null)

bootdisk=""

# Step 2: Try to find which disk's MBR contains GRUB signature
#   GRUB stage1 MBRs contain either "GRUB" or "boot" strings near offset 0x3E
for disk in $disks; do
  # read first 512 bytes
  sig=$(dd if="$disk" bs=512 count=1 2>/dev/null | strings | grep -E 'GRUB|boot' | head -n1)
  if [ -n "$sig" ]; then
    bootdisk="$disk"
    break
  fi
done

# Step 3: Check for /boot partition in fstab
if [ -z "$bootdisk" ]; then
  boot_uuid=$(grep -w '/boot' /etc/fstab | awk '{print $1}' | sed 's/^UUID=//')
  if [ -n "$boot_uuid" ]; then
    bootdisk=$(blkid | grep "$boot_uuid" | awk -F: '{sub(/[0-9][0-9]*$/, "", $1); print $1; exit}')
  fi
fi

# Step 4: Check for LVM root device
if [ -z "$bootdisk" ]; then
  # Check if root is on LVM
  root_dev=$(mount | grep 'on / ' | awk '{print $1}')
  if echo "$root_dev" | grep -q '/dev/mapper/\|/dev/.*-vg/'; then
    # Root is on LVM, find physical volumes
    if command -v pvs >/dev/null 2>&1; then
      # Get VG name from root device
      vg_name=$(echo "$root_dev" | sed 's|/dev/mapper/\([^-]*\)-.*|\1|; s|/dev/\([^/]*\)-vg/.*|\1|')
      # Find PV for this VG
      pv_dev=$(pvs --noheadings -o pv_name,vg_name 2>/dev/null | awk -v vg="$vg_name" '$2 == vg {print $1; exit}')
      if [ -n "$pv_dev" ]; then
        # Strip partition number to get disk
        bootdisk=$(echo "$pv_dev" | sed 's/[0-9][0-9]*$//')
      fi
    fi
  else
    # Root is on regular partition, strip partition number
    bootdisk=$(echo "$root_dev" | sed 's/[0-9][0-9]*$//')
  fi
fi

# Step 5: Check for partitions with bootable flag set
if [ -z "$bootdisk" ]; then
  for disk in $disks; do
    # Get partition count for this disk
    partitions=$(ls "${disk}"[0-9]* 2>/dev/null)
    for part in $partitions; do
      # Extract partition number
      part_num=$(echo "$part" | sed "s|${disk}||")
      # Check if partition is bootable using parted
      if command -v parted >/dev/null 2>&1; then
        bootable=$(parted -s "$disk" print 2>/dev/null | awk -v pnum="$part_num" '$1 == pnum && /boot/ {print "true"}')
        if [ "$bootable" = "true" ]; then
          bootdisk="$disk"
          break 2
        fi
      fi
    done
  done
fi

# Step 6: Final fallback - use first disk
if [ -z "$bootdisk" ]; then
  bootdisk=$(echo "$disks" | awk '{print $1}')
fi

echo "$bootdisk"