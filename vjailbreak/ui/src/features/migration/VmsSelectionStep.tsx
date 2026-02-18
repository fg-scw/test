import {
  Chip,
  FormControl,
  FormHelperText,
  Paper,
  styled,
  Tooltip,
  Box,
  Button,
  Autocomplete,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  MenuItem,
  Select,
  Typography,
  Snackbar,
  Alert,
  CircularProgress,
  GlobalStyles,
  InputAdornment
} from '@mui/material'
import {
  DataGrid,
  GridColDef,
  GridToolbarColumnsButton,
  GridRowSelectionModel,
  GridRow
} from '@mui/x-data-grid'
import { useQueryClient } from '@tanstack/react-query'
import { VmData } from 'src/features/migration/api/migration-templates/model'
import { OpenStackFlavor, OpenstackCreds } from 'src/api/openstack-creds/model'
import { patchVMwareMachine } from 'src/api/vmware-machines/vmwareMachines'
import { CustomLoadingOverlay, CustomSearchToolbar } from 'src/components/grid'
import { Step } from 'src/shared/components/forms'
import { useEffect, useState, useCallback, useRef } from 'react'
import * as React from 'react'
import { getMigrationPlans } from 'src/features/migration/api/migration-plans/migrationPlans'
import { useVMwareMachinesQuery } from 'src/hooks/api/useVMwareMachinesQuery'
import InfoIcon from '@mui/icons-material/Info'
import WarningIcon from '@mui/icons-material/Warning'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import ErrorIcon from '@mui/icons-material/Error'
import WindowsIcon from 'src/assets/windows_icon.svg'
import LinuxIcon from 'src/assets/linux_icon.svg'
import { useErrorHandler } from 'src/hooks/useErrorHandler'
import { validateOpenstackIPs } from 'src/api/openstack-creds/openstackCreds'
import { useAmplitude } from 'src/hooks/useAmplitude'
import { useRdmConfigValidation } from 'src/hooks/useRdmConfigValidation'
import { useRdmDisksQuery, RDM_DISKS_BASE_KEY } from 'src/hooks/api/useRdmDisksQuery'
import { patchRdmDisk } from 'src/api/rdm-disks/rdmDisks'
import { RdmDisk } from 'src/api/rdm-disks/model'
import axios from 'axios'
import { RdmDiskConfigurationPanel } from './components'
import { FieldLabel } from 'src/components'
import { ActionButton } from 'src/components'
import { TextField as SharedTextField } from 'src/shared/components/forms'

const VmsSelectionStepContainer = styled('div')(({ theme }) => ({
  display: 'grid',
  gridGap: theme.spacing(1),
  '& .disabled-row': {
    opacity: 0.6,
    cursor: 'not-allowed'
  },
  '& .hidden-column': {
    display: 'none'
  },
  '& .warning-row': {
    color: '#856404',
    fontWeight: 'bold'
  }
}))

const FieldsContainer = styled('div')({
  display: 'grid'
})

// Style for Clarity icons
const CdsIconWrapper = styled('div')({
  marginRight: 8,
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center'
})

const CustomToolbarWithActions = (props) => {
  const {
    rowSelectionModel,
    onAssignFlavor,
    onAssignIP,
    hasRdmVMs,
    onAssignRdmConfiguration,
    selectedCount,
    ...toolbarProps
  } = props

  return (
    <Box
      sx={{ display: 'flex', justifyContent: 'space-between', width: '100%', padding: '4px 8px' }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <GridToolbarColumnsButton />
      </Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        {rowSelectionModel && rowSelectionModel.length > 0 && (
          <>
            <Button variant="text" color="primary" onClick={onAssignFlavor} size="small">
              Assign Flavor ({rowSelectionModel.length})
            </Button>
            {hasRdmVMs && (
              <Button
                variant="text"
                color="secondary"
                onClick={onAssignRdmConfiguration}
                size="small"
              >
                Configure RDM ({rowSelectionModel.length})
              </Button>
            )}
            {selectedCount > 0 && (
              <Button variant="text" color="primary" onClick={onAssignIP} size="small">
                Assign IP ({selectedCount})
              </Button>
            )}
          </>
        )}
        <CustomSearchToolbar {...toolbarProps} />
      </Box>
    </Box>
  )
}

interface VmDataWithFlavor extends VmData {
  isMigrated?: boolean
  flavorName?: string // Add a field to store the flavor name
  flavorNotFound?: boolean // Add a flag to indicate if a flavor wasn't found
  ipValidationStatus?: 'pending' | 'valid' | 'invalid' | 'validating'
  ipValidationMessage?: string
  powerState?: string // Add power state for IP editing logic
  assignedIPs?: string
}

// Column definition moved inside component to access state

const paginationModel = { page: 0, pageSize: 5 }

const MIGRATED_TOOLTIP_MESSAGE = 'This VM is migrating or already has been migrated.'
const FLAVOR_NOT_FOUND_MESSAGE =
  'Appropriate flavor not found. Please assign a flavor before selecting this VM for migration or create a flavor.'

interface VmsSelectionStepProps {
  onChange: (id: string) => (value: unknown) => void
  error: string
  open?: boolean
  vmwareCredsValidated: boolean
  openstackCredsValidated: boolean
  sessionId?: string
  openstackFlavors?: OpenStackFlavor[]
  vmwareCredName?: string
  openstackCredName?: string
  openstackCredentials?: OpenstackCreds
  vmwareCluster?: string
  useGPU?: boolean
  showHeader?: boolean
}

function VmsSelectionStep({
  onChange,
  error,
  open = false,
  vmwareCredsValidated,
  openstackCredsValidated,
  sessionId = Date.now().toString(),
  openstackFlavors = [],
  vmwareCredName,
  openstackCredName,
  openstackCredentials,
  vmwareCluster,
  useGPU = false,
  showHeader = true
}: VmsSelectionStepProps) {
  const { reportError } = useErrorHandler({ component: 'VmsSelectionStep' })
  const { track } = useAmplitude({ component: 'VmsSelectionStep' })
  const queryClient = useQueryClient()
  const [migratedVms, setMigratedVms] = useState<Set<string>>(new Set())
  const [loadingMigratedVms, setLoadingMigratedVms] = useState(false)
  const [flavorDialogOpen, setFlavorDialogOpen] = useState(false)
  const [selectedFlavor, setSelectedFlavor] = useState<string>('')
  const [rdmConfigDialogOpen, setRdmConfigDialogOpen] = useState(false)
  const [selectedVMs, setSelectedVMs] = useState<Set<string>>(new Set())
  const [vmsWithFlavor, setVmsWithFlavor] = useState<VmDataWithFlavor[]>([])
  const [snackbarOpen, setSnackbarOpen] = useState(false)
  const [snackbarMessage, setSnackbarMessage] = useState('')
  const [snackbarSeverity, setSnackbarSeverity] = useState<'success' | 'error'>('success')

  // Toast notification for IP assignments
  const [toastOpen, setToastOpen] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastSeverityIp, setToastSeverityIp] = useState<'success' | 'error' | 'warning' | 'info'>(
    'success'
  )
  const [updating, setUpdating] = useState(false)

  // Toast notification helper
  const showToast = useCallback(
    (message: string, severity: 'success' | 'error' | 'warning' | 'info' = 'success') => {
      setToastMessage(message)
      setToastSeverityIp(severity)
      setToastOpen(true)
    },
    []
  )

  const handleCloseToast = useCallback((_event?: React.SyntheticEvent | Event, reason?: string) => {
    if (reason === 'clickaway') {
      return
    }
    setToastOpen(false)
  }, [])

  // OS assignment state
  const [vmOSAssignments, setVmOSAssignments] = useState<Record<string, string>>({})

  const { data: rdmDisks = [], isLoading: rdmDisksLoading } = useRdmDisksQuery()

  // RDM validation logic
  const rdmValidation = useRdmConfigValidation({
    selectedVMs: Array.from(selectedVMs)
      .map((vmName) => vmsWithFlavor.find((vm) => vm.name === vmName))
      .filter(Boolean) as VmData[],
    rdmDisks: rdmDisks
  })

  // RDM configuration state
  const [rdmConfigurations, setRdmConfigurations] = useState<
    Array<{
      uuid: string
      diskName: string
      cinderBackendPool: string
      volumeType: string
      source: Record<string, string>
    }>
  >([])
  const lastSelectedVmsPayloadRef = useRef<string>('__initial__')
  const lastRdmConfigPayloadRef = useRef<string>('__initial__')

  const setFormVms = React.useMemo(() => onChange('vms'), [onChange])
  const setFormRdmConfigurations = React.useMemo(() => onChange('rdmConfigurations'), [onChange])

  const syncSelectedVmSelection = useCallback(
    (selectedVmData: VmDataWithFlavor[]) => {
      const payload = JSON.stringify(selectedVmData)
      if (payload === lastSelectedVmsPayloadRef.current) {
        return
      }
      lastSelectedVmsPayloadRef.current = payload
      setFormVms(selectedVmData)
    },
    [setFormVms]
  )

  const syncRdmConfigurations = useCallback(
    (
      configs: Array<{
        uuid: string
        diskName: string
        cinderBackendPool: string
        volumeType: string
        source: Record<string, string>
      }>
    ) => {
      const payload = JSON.stringify(configs)
      if (payload === lastRdmConfigPayloadRef.current) {
        return
      }
      lastRdmConfigPayloadRef.current = payload
      setFormRdmConfigurations(configs)
    },
    [setFormRdmConfigurations]
  )

  const areSetsEqual = useCallback((a: Set<string>, b: Set<string>) => {
    if (a.size !== b.size) {
      return false
    }
    for (const value of a) {
      if (!b.has(value)) {
        return false
      }
    }
    return true
  }, [])

  // IP editing and validation state - similar to RollingMigrationForm

  // Bulk IP editing state (kept for potential future use but not accessible via UI)
  const [bulkEditDialogOpen, setBulkEditDialogOpen] = useState(false)
  const [bulkEditIPs, setBulkEditIPs] = useState<Record<string, Record<number, string>>>({})
  const [bulkValidationStatus, setBulkValidationStatus] = useState<
    Record<string, Record<number, 'empty' | 'valid' | 'invalid' | 'validating'>>
  >({})
  const [bulkValidationMessages, setBulkValidationMessages] = useState<
    Record<string, Record<number, string>>
  >({})
  const [assigningIPs, setAssigningIPs] = useState(false)
  const hasBulkIpValidationErrors = React.useMemo(() => {
    return Object.values(bulkValidationStatus).some((interfaces) =>
      Object.values(interfaces || {}).some((status) => status === 'invalid')
    )
  }, [bulkValidationStatus])
  const hasBulkIpsToApply = React.useMemo(() => {
    return Object.values(bulkEditIPs).some((interfaces) =>
      Object.values(interfaces || {}).some((ip) => Boolean(ip?.trim()))
    )
  }, [bulkEditIPs])

  const clusterName = React.useMemo(() => {
    if (!vmwareCluster) return undefined
    const parts = vmwareCluster.split(':')
    // The value is "credName:datacenter:clusterName"
    return parts.length === 3 ? parts[2] : undefined
  }, [vmwareCluster])

  const datacenterName = React.useMemo(() => {
    if (!vmwareCluster) return undefined
    const parts = vmwareCluster.split(':')
    // Extract datacenter from cluster ID
    return parts.length === 3 ? parts[1] : undefined
  }, [vmwareCluster])

  // Define columns inside component to access state and functions
  const columns: GridColDef[] = [
    {
      field: 'name',
      headerName: 'VM Name',
      flex: 2.5,
      renderCell: (params) => (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Tooltip title={params.row.vmState === 'running' ? 'Running' : 'Stopped'}>
              <CdsIconWrapper>
                {/* eslint-disable-next-line @typescript-eslint/ban-ts-comment */}
                {/* @ts-ignore */}
                <cds-icon
                  shape="vm"
                  size="md"
                  badge={params.row.vmState === 'running' ? 'success' : 'danger'}
                >
                  {/* @ts-ignore */}
                </cds-icon>
              </CdsIconWrapper>
            </Tooltip>
            <Box>{params.value}</Box>
          </Box>
          {params.row.isMigrated && (
            <Chip variant="outlined" label="Migrated" color="info" size="small" />
          )}
          {params.row.flavorNotFound && (
            <Box display="flex" alignItems="center" gap={0.5}>
              <WarningIcon color="warning" fontSize="small" />
            </Box>
          )}
          {params.row.hasSharedRdm && (
            <Tooltip title="This VM has shared RDM disks">
              <Chip
                variant="outlined"
                label="RDM"
                color="secondary"
                size="small"
                sx={{ fontSize: '0.7rem', height: '20px' }}
              />
            </Tooltip>
          )}
        </Box>
      )
    },
    {
      field: 'ipAddress',
      headerName: 'IP Address(es)',
      flex: 1,
      hideable: true,
      renderCell: (params) => {
        const vm = params.row as VmDataWithFlavor
        const vmId = vm.name
        const isSelected = selectedVMs.has(vmId)
        const networkInterfaces = Array.isArray(vm.networkInterfaces) ? vm.networkInterfaces : []
        const hasMultipleInterfaces = networkInterfaces.length > 1
        const ipDisplay = hasMultipleInterfaces
          ? networkInterfaces.map((nic) => nic.ipAddress || '—').join(', ')
          : networkInterfaces[0]?.ipAddress || vm.ipAddress || '—'
        const tooltipMessage = hasMultipleInterfaces
          ? "Use 'Assign IP' button in toolbar to edit IP addresses for multiple network interfaces"
          : "Use 'Assign IP' button in toolbar to assign IP address"

        const content = (
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              width: '100%',
              height: '100%'
            }}
          >
            <Typography
              variant="body2"
              sx={{
                fontSize: '0.875rem',
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis'
              }}
            >
              {ipDisplay}
            </Typography>
          </Box>
        )

        return isSelected ? (
          <Tooltip title={tooltipMessage} arrow placement="top">
            {content}
          </Tooltip>
        ) : (
          content
        )
      }
    },
    {
      field: 'osFamily',
      headerName: 'Operating System',
      flex: 1,
      hideable: true,
      renderCell: (params) => {
        const vmId = params.row.id
        const isSelected = selectedVMs.has(vmId)
        const powerState = params.row?.powerState
        const detectedOsFamily = params.row?.osFamily
        const assignedOsFamily = vmOSAssignments[vmId]
        const currentOsFamily = assignedOsFamily === undefined ? detectedOsFamily : assignedOsFamily
        // Show dropdown when:
        // - VM is selected
        if (isSelected) {
          return (
            <Box sx={{ display: 'flex', alignItems: 'center', height: '100%' }}>
              <Select
                size="small"
                value={(() => {
                  if (!currentOsFamily || currentOsFamily === 'Unknown') return ''
                  const osLower = currentOsFamily.toLowerCase()
                  if (osLower.includes('windows')) return 'windowsGuest'
                  if (osLower.includes('linux')) return 'linuxGuest'
                  return ''
                })()}
                onChange={(e) => handleOSAssignment(vmId, e.target.value)}
                displayEmpty
                sx={{
                  minWidth: 120,
                  '& .MuiSelect-select': {
                    padding: '4px 8px',
                    fontSize: '0.875rem'
                  }
                }}
                // small optimization: keep menu mounted to avoid remount cost
                MenuProps={{ keepMounted: true }}
              >
                <MenuItem value="">
                  <Box
                    sx={{ display: 'flex', alignItems: 'center', gap: 1, color: 'text.secondary' }}
                  >
                    <WarningIcon sx={{ fontSize: 16 }} />
                    <em>Select OS</em>
                  </Box>
                </MenuItem>
                <MenuItem value="windowsGuest">
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <img src={WindowsIcon} alt="Windows" style={{ width: 16, height: 16 }} />
                    Windows
                  </Box>
                </MenuItem>
                <MenuItem value="linuxGuest">
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <img src={LinuxIcon} alt="Linux" style={{ width: 16, height: 16 }} />
                    Linux
                  </Box>
                </MenuItem>
              </Select>
            </Box>
          )
        }

        let displayValue = currentOsFamily || 'Unknown'
        let icon: React.ReactNode = null

        if (currentOsFamily && currentOsFamily.toLowerCase().includes('windows')) {
          displayValue = 'Windows'
          icon = <img src={WindowsIcon} alt="Windows" style={{ width: 20, height: 20 }} />
        } else if (currentOsFamily && currentOsFamily.toLowerCase().includes('linux')) {
          displayValue = 'Linux'
          icon = <img src={LinuxIcon} alt="Linux" style={{ width: 20, height: 20 }} />
        } else if (currentOsFamily && currentOsFamily !== 'Unknown') {
          displayValue = 'Unknown'
        }

        return (
          <Tooltip
            title={
              powerState === 'powered-off'
                ? !currentOsFamily || currentOsFamily === 'Unknown'
                  ? 'OS assignment required for powered-off VMs'
                  : 'Click to change OS selection'
                : displayValue
            }
          >
            <Box
              sx={{
                display: 'flex',
                alignItems: 'center',
                height: '100%',
                gap: 1
              }}
            >
              {icon}
              {(!currentOsFamily || currentOsFamily === 'Unknown') && (
                <WarningIcon sx={{ color: 'warning.main', fontSize: 16 }} />
              )}
              <Typography
                variant="body2"
                sx={{
                  color:
                    !currentOsFamily || currentOsFamily === 'Unknown'
                      ? 'text.secondary'
                      : 'text.primary'
                }}
              >
                {displayValue}
              </Typography>
            </Box>
          </Tooltip>
        )
      }
    },
    {
      field: 'networks',
      headerName: 'Network Interface(s)',
      flex: 1.2,
      valueGetter: (value: string[]) => value?.join(', ') || '- '
    },
    {
      field: 'cpuCount',
      headerName: 'CPU',
      flex: 0.7,
      valueGetter: (value) => value || '- '
    },
    {
      field: 'memory',
      headerName: 'Memory (MB)',
      flex: 0.9,
      valueGetter: (value) => value || '- '
    },
    {
      field: 'esxHost',
      headerName: 'ESX Host',
      flex: 1,
      valueGetter: (value) => value || '—'
    },
    {
      field: 'flavor',
      headerName: 'Flavor',
      flex: 1,
      valueGetter: (value) => value || 'auto-assign',
      renderHeader: () => (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
          <div style={{ fontWeight: 500 }}>Flavor</div>
          <Tooltip title="Target PCD flavor to be assigned to this VM after migration.">
            <InfoIcon fontSize="small" sx={{ color: 'info.info', opacity: 0.7, cursor: 'help' }} />
          </Tooltip>
        </Box>
      )
    },
    {
      field: 'rdmDisks',
      headerName: 'RDM Disks',
      flex: 1.2,
      hideable: true,
      valueGetter: (value: string[]) => value?.join(', ') || '—',
      renderHeader: () => (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
          <div style={{ fontWeight: 500 }}>RDM Disks</div>
          <Tooltip title="Raw Device Mapping disks associated with this VM.">
            <InfoIcon fontSize="small" sx={{ color: 'info.info', opacity: 0.7, cursor: 'help' }} />
          </Tooltip>
        </Box>
      )
    },
    // Hidden column for sorting by vmState
    {
      field: 'vmState',
      headerName: 'Status',
      flex: 1,
      sortable: true,
      sortComparator: (v1, v2) => {
        if (v1 === 'running' && v2 === 'stopped') return -1
        if (v1 === 'stopped' && v2 === 'running') return 1
        return 0
      }
    }
  ]

  // IP validation and utility functions
  const isValidIPAddress = (ip: string): boolean => {
    const ipRegex =
      /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/
    return ipRegex.test(ip)
  }

  // Removed getOpenstackAccessInfo - no longer needed with new API

  useEffect(() => {
    if (!open) {
      setSelectedVMs(new Set())
    }
  }, [open])

  const {
    data: vmList = [],
    isLoading: loadingVms,
    refetch: refreshVMList
  } = useVMwareMachinesQuery({
    vmwareCredsValidated,
    openstackCredsValidated,
    enabled: open,
    sessionId,
    vmwareCredName,
    clusterName,
    datacenterName
  })

  useEffect(() => {
    if (open && vmwareCluster !== undefined) {
      refreshVMList()
    }
  }, [vmwareCluster])

  useEffect(() => {
    const fetchMigratedVms = async () => {
      if (!open) return

      setLoadingMigratedVms(true)
      try {
        const plans = await getMigrationPlans()
        const migratedVmSet = new Set<string>()

        plans.forEach((plan) => {
          plan.spec.virtualMachines.forEach((vmList) => {
            vmList.forEach((vm) => migratedVmSet.add(vm))
          })
        })

        setMigratedVms(migratedVmSet)
      } catch (error) {
        console.error('Error fetching migrated VMs:', error)
      } finally {
        setLoadingMigratedVms(false)
      }
    }

    fetchMigratedVms()
  }, [open, vmList])

  useEffect(() => {
    // Create a map of existing VM data (including assigned IPs) for quick lookup
    const existingVmsMap = new Map(vmsWithFlavor.map((vm) => [vm.name, vm]))

    const initialVmsWithFlavor = vmList.map((vm) => {
      let flavor = ''
      if (vm.targetFlavorId) {
        const foundFlavor = openstackFlavors.find((f) => f.id === vm.targetFlavorId)
        if (foundFlavor) {
          flavor = foundFlavor.name
        } else {
          flavor = vm.targetFlavorId
        }
      }

      // Check for NOT_FOUND label for OpenStack credentials
      const flavorNotFound = openstackCredName
        ? vm.labels?.[openstackCredName] === 'NOT_FOUND'
        : false

      // Map power state from vmState
      const powerState = vm.vmState === 'running' ? 'powered-on' : 'powered-off'

      // FIX: Use existing IP data from vmsWithFlavor if the VM exists,
      // otherwise, use the fresh data from vmList.
      const existingVm = existingVmsMap.get(vm.name)

      let allIPs = vm.networkInterfaces
        ? vm.networkInterfaces
            .map((nic) => nic.ipAddress)
            .filter((ip) => ip && ip.trim() !== '')
            .join(', ')
        : vm.ipAddress || ''

      // If the existing VM has a meaningful ipAddress (and not the placeholder '—'), prefer that
      // This preserves local assignments (assignedIPs, updated networkInterfaces) done elsewhere
      if (existingVm && existingVm.ipAddress && existingVm.ipAddress !== '—') {
        allIPs = existingVm.ipAddress ?? allIPs
      }

      // If existingVm has modified networkInterfaces, prefer them
      let preferredNetworkInterfaces = vm.networkInterfaces
      if (existingVm && existingVm.networkInterfaces && existingVm.networkInterfaces.length > 0) {
        preferredNetworkInterfaces = existingVm.networkInterfaces
      }

      // If existingVm stored assignedIPs, keep them in the new object
      const assignedIPs = existingVm?.assignedIPs ?? undefined

      // Use assigned OS family if available is handled by separate effect (do not include vmOSAssignments here)
      const finalOsFamily = vm.osFamily

      return {
        ...vm,
        ipAddress: allIPs || '—', // Update the main IP field to contain comma-separated IPs
        isMigrated: migratedVms.has(vm.name) || Boolean(vm.isMigrated),
        flavor,
        flavorNotFound,
        powerState,
        osFamily: finalOsFamily, // Use detected OS family (assignments applied separately)
        ipValidationStatus: 'pending' as const,
        ipValidationMessage: '',
        networkInterfaces: preferredNetworkInterfaces,
        assignedIPs
      }
    })
    setVmsWithFlavor(initialVmsWithFlavor)
  }, [
    vmList,
    migratedVms,
    openstackFlavors,
    openstackCredName,
    // removed vmOSAssignments here intentionally to avoid full remap when only OS assignment changes
    vmsWithFlavor.length
  ])

  // New small effect — apply local OS assignments to existing vmsWithFlavor without remapping from vmList.
  useEffect(() => {
    if (Object.keys(vmOSAssignments).length === 0) return

    setVmsWithFlavor((prev) =>
      prev.map((vm) => {
        if (vmOSAssignments && Object.prototype.hasOwnProperty.call(vmOSAssignments, vm.name)) {
          return {
            ...vm,
            osFamily: vmOSAssignments[vm.name]
          }
        }
        return vm
      })
    )
  }, [vmOSAssignments])

  // Separate effect for cleaning up selections when VM list changes
  useEffect(() => {
    if (vmsWithFlavor.length === 0) return

    // Clean up selection - remove VMs that no longer exist
    const availableVmNames = new Set(vmsWithFlavor.map((vm) => vm.name))
    const cleanedSelection = new Set(
      Array.from(selectedVMs).filter((vmName) => availableVmNames.has(vmName))
    )

    if (!areSetsEqual(cleanedSelection, selectedVMs)) {
      setSelectedVMs(cleanedSelection)

      const selectedVmData = vmsWithFlavor.filter((vm) => cleanedSelection.has(vm.name))
      syncSelectedVmSelection(selectedVmData)

      if (rdmConfigurations.length > 0) {
        syncRdmConfigurations(rdmConfigurations)
      }
    }
  }, [
    vmsWithFlavor,
    selectedVMs,
    rdmConfigurations,
    areSetsEqual,
    syncSelectedVmSelection,
    syncRdmConfigurations
  ])

  useEffect(() => {
    const selectedVmData = vmsWithFlavor.filter((vm) => selectedVMs.has(vm.name))
    syncSelectedVmSelection(selectedVmData)

    if (selectedVmData.length > 0 && rdmConfigurations.length > 0) {
      syncRdmConfigurations(rdmConfigurations)
    }
  }, [
    vmsWithFlavor,
    selectedVMs,
    rdmConfigurations,
    syncSelectedVmSelection,
    syncRdmConfigurations
  ])

  const handleVmSelection = (selectedRowIds: GridRowSelectionModel) => {
    const newSelection = new Set<string>(selectedRowIds as string[])

    if (areSetsEqual(newSelection, selectedVMs)) {
      return
    }

    setSelectedVMs(newSelection)

    const selectedVmData = vmsWithFlavor.filter((vm) => newSelection.has(vm.name))
    syncSelectedVmSelection(selectedVmData)

    if (rdmConfigurations.length > 0) {
      syncRdmConfigurations(rdmConfigurations)
    }
  }

  // OS assignment handler
  const handleOSAssignment = async (vmId: string, osFamily: string) => {
    try {
      // Update local state first for immediate UI feedback
      setVmOSAssignments((prev) => ({ ...prev, [vmId]: osFamily }))

      const vm = vmsWithFlavor.find((v) => v.name === vmId)
      if (vm?.vmWareMachineName) {
        await patchVMwareMachine(vm.vmWareMachineName, {
          spec: {
            vms: {
              osFamily: osFamily
            }
          }
        })
      }

      // Track the analytics event
      track('os_family_assigned', {
        vm_name: vmId,
        os_family: osFamily,
        action: 'os-family-assignment'
      })

      showToast(`OS family successfully assigned for VM "${vmId}"`)
    } catch (error) {
      reportError(error as Error, {
        context: 'os-family-assignment',
        metadata: {
          vmId: vmId,
          osFamily: osFamily,
          action: 'os-family-assignment'
        }
      })
      // Revert local state on error
      setVmOSAssignments((prev) => {
        const newState = { ...prev }
        delete newState[vmId]
        return newState
      })
      showToast(
        `Failed to assign OS family for VM "${vmId}": ${
          error instanceof Error ? error.message : 'Unknown error'
        }`,
        'error'
      )
    }
  }

  // Bulk IP editing functions (removed - not accessible via UI anymore)

  const handleCloseBulkEditDialog = () => {
    setBulkEditDialogOpen(false)
    setBulkEditIPs({})
    setBulkValidationStatus({})
    setBulkValidationMessages({})
  }

  const handleBulkIpChange = (vmName: string, interfaceIndex: number, value: string) => {
    setBulkEditIPs((prev) => ({
      ...prev,
      [vmName]: { ...prev[vmName], [interfaceIndex]: value }
    }))

    if (!value.trim()) {
      setBulkValidationStatus((prev) => ({
        ...prev,
        [vmName]: { ...prev[vmName], [interfaceIndex]: 'empty' }
      }))
      setBulkValidationMessages((prev) => ({
        ...prev,
        [vmName]: { ...prev[vmName], [interfaceIndex]: '' }
      }))
    } else if (!isValidIPAddress(value.trim())) {
      setBulkValidationStatus((prev) => ({
        ...prev,
        [vmName]: { ...prev[vmName], [interfaceIndex]: 'invalid' }
      }))
      setBulkValidationMessages((prev) => ({
        ...prev,
        [vmName]: { ...prev[vmName], [interfaceIndex]: 'Invalid IP format' }
      }))
    } else {
      setBulkValidationStatus((prev) => ({
        ...prev,
        [vmName]: { ...prev[vmName], [interfaceIndex]: 'empty' }
      }))
      setBulkValidationMessages((prev) => ({
        ...prev,
        [vmName]: { ...prev[vmName], [interfaceIndex]: '' }
      }))
    }
  }

  const handleClearAllIPs = () => {
    const clearedIPs: Record<string, Record<number, string>> = {}
    const clearedStatus: Record<
      string,
      Record<number, 'empty' | 'valid' | 'invalid' | 'validating'>
    > = {}

    Object.keys(bulkEditIPs).forEach((vmName) => {
      clearedIPs[vmName] = {}
      clearedStatus[vmName] = {}

      Object.keys(bulkEditIPs[vmName]).forEach((interfaceIndexStr) => {
        const interfaceIndex = parseInt(interfaceIndexStr)
        clearedIPs[vmName][interfaceIndex] = ''
        clearedStatus[vmName][interfaceIndex] = 'empty'
      })
    })

    setBulkEditIPs(clearedIPs)
    setBulkValidationStatus(clearedStatus)
    setBulkValidationMessages({})
  }

  const renderValidationAdornment = (status?: 'empty' | 'valid' | 'invalid' | 'validating') => {
    if (!status || status === 'empty') return null

    if (status === 'validating') {
      return (
        <InputAdornment position="end" sx={{ alignItems: 'center' }}>
          <CircularProgress size={16} />
        </InputAdornment>
      )
    }

    if (status === 'valid') {
      return (
        <InputAdornment position="end" sx={{ alignItems: 'center' }}>
          <CheckCircleIcon color="success" fontSize="small" />
        </InputAdornment>
      )
    }

    if (status === 'invalid') {
      return (
        <InputAdornment position="end" sx={{ alignItems: 'center' }}>
          <ErrorIcon color="error" fontSize="small" />
        </InputAdornment>
      )
    }

    return null
  }

  const handleApplyBulkIPs = async () => {
    // Collect all IPs to apply with their VM and interface info
    const ipsToApply: Array<{ vmName: string; interfaceIndex: number; ip: string }> = []

    Object.entries(bulkEditIPs).forEach(([vmName, interfaces]) => {
      Object.entries(interfaces).forEach(([interfaceIndexStr, ip]) => {
        if (ip.trim() !== '') {
          ipsToApply.push({
            vmName,
            interfaceIndex: parseInt(interfaceIndexStr),
            ip: ip.trim()
          })
        }
      })
    })

    if (ipsToApply.length === 0) return
    if (hasBulkIpValidationErrors) {
      showToast('Resolve invalid IP addresses before applying changes.', 'error')
      return
    }

    const markBulkValidationFailure = (message: string) => {
      setBulkValidationStatus((prev) => {
        const newStatus = { ...prev }
        ipsToApply.forEach(({ vmName, interfaceIndex }) => {
          if (!newStatus[vmName]) newStatus[vmName] = {}
          newStatus[vmName][interfaceIndex] = 'invalid'
        })
        return newStatus
      })
      setBulkValidationMessages((prev) => {
        const newMessages = { ...prev }
        ipsToApply.forEach(({ vmName, interfaceIndex }) => {
          if (!newMessages[vmName]) newMessages[vmName] = {}
          newMessages[vmName][interfaceIndex] = message
        })
        return newMessages
      })
    }

    setAssigningIPs(true)

    try {
      // Batch validation before applying any changes
      if (openstackCredentials) {
        const ipList = ipsToApply.map((item) => item.ip)

        // Set validating status for all IPs
        setBulkValidationStatus((prev) => {
          const newStatus = { ...prev }
          ipsToApply.forEach(({ vmName, interfaceIndex }) => {
            if (!newStatus[vmName]) newStatus[vmName] = {}
            newStatus[vmName][interfaceIndex] = 'validating'
          })
          return newStatus
        })

        let validationResult
        try {
          validationResult = await validateOpenstackIPs({
            ip: ipList,
            accessInfo: {
              secret_name: `${openstackCredentials.metadata.name}-openstack-secret`,
              secret_namespace: openstackCredentials.metadata.namespace
            }
          })
        } catch (error) {
          if (axios.isAxiosError(error) && error.response?.status === 500) {
            const responseData = error.response?.data as { message?: string } | string | undefined
            const apiMessage =
              typeof responseData === 'string' ? responseData : responseData?.message
            const validationErrorMessage =
              apiMessage ||
              'PCD IP validation service is unavailable (500). Please verify credentials or try again later.'

            markBulkValidationFailure(validationErrorMessage)
            showToast(validationErrorMessage, 'error')
            reportError(error as Error, {
              context: 'bulk-ip-validation-request',
              metadata: {
                bulkEditIPs: bulkEditIPs,
                action: 'bulk-ip-validation-assignment',
                status: error.response?.status
              }
            })
            setAssigningIPs(false)
            return
          }

          throw error
        }

        // Process validation results
        const validIPs: Array<{ vmName: string; interfaceIndex: number; ip: string }> = []
        let hasInvalidIPs = false

        ipsToApply.forEach((item, index) => {
          const isValid = validationResult.isValid[index]
          const reason = validationResult.reason[index]

          if (isValid) {
            validIPs.push(item)
            setBulkValidationStatus((prev) => ({
              ...prev,
              [item.vmName]: { ...prev[item.vmName], [item.interfaceIndex]: 'valid' }
            }))
            setBulkValidationMessages((prev) => ({
              ...prev,
              [item.vmName]: { ...prev[item.vmName], [item.interfaceIndex]: 'Valid' }
            }))
          } else {
            hasInvalidIPs = true
            setBulkValidationStatus((prev) => ({
              ...prev,
              [item.vmName]: { ...prev[item.vmName], [item.interfaceIndex]: 'invalid' }
            }))
            setBulkValidationMessages((prev) => ({
              ...prev,
              [item.vmName]: { ...prev[item.vmName], [item.interfaceIndex]: reason }
            }))
          }
        })

        // Only proceed if ALL IPs are valid
        if (hasInvalidIPs) {
          setAssigningIPs(false)
          return
        }

        // Group IPs by VM name
        const assignedIPsPerVM: Record<string, string[]> = {}

        validIPs.forEach(({ vmName, interfaceIndex, ip }) => {
          if (!assignedIPsPerVM[vmName]) {
            assignedIPsPerVM[vmName] = []
          }
          // Ensure the array has enough slots
          while (assignedIPsPerVM[vmName].length <= interfaceIndex) {
            assignedIPsPerVM[vmName].push('')
          }
          if (assignedIPsPerVM[vmName].length > interfaceIndex) {
            assignedIPsPerVM[vmName][interfaceIndex] = ip
          }
        })

        // Update vmsWithFlavor to include assigned IPs for display purposes only
        const updatedVms = vmsWithFlavor.map((vm) => {
          const assignedIPs = assignedIPsPerVM[vm.name]
          if (!assignedIPs) return vm

          // Update networkInterfaces with assigned IPs
          let updatedNetworkInterfaces = vm.networkInterfaces
          if (updatedNetworkInterfaces && updatedNetworkInterfaces.length > 0) {
            updatedNetworkInterfaces = updatedNetworkInterfaces.map((nic, index) => {
              const assignedIP = assignedIPs[index]
              if (assignedIP && assignedIP.trim() !== '') {
                return { ...nic, ipAddress: assignedIP }
              }
              return nic
            })
          }

          const validIPs = assignedIPs.filter((ip) => ip && ip.trim() !== '')
          const ipDisplay = validIPs.join(', ')

          return {
            ...vm,
            assignedIPs: assignedIPs.join(','),
            ipAddress: ipDisplay || vm.ipAddress,
            networkInterfaces: updatedNetworkInterfaces
          }
        })

        setVmsWithFlavor(updatedVms)

        // Mark all as successfully applied
        validIPs.forEach(({ vmName, interfaceIndex }) => {
          setBulkValidationStatus((prev) => ({
            ...prev,
            [vmName]: { ...prev[vmName], [interfaceIndex]: 'valid' }
          }))
          setBulkValidationMessages((prev) => ({
            ...prev,
            [vmName]: { ...prev[vmName], [interfaceIndex]: 'IP assigned locally' }
          }))
        })

        // Notify success
        showToast(`Successfully assigned IPs to ${validIPs.length} interface(s)`, 'success')

        handleCloseBulkEditDialog()
      }
    } catch (error) {
      reportError(error as Error, {
        context: 'bulk-ip-validation-assignment',
        metadata: {
          bulkEditIPs: bulkEditIPs,
          action: 'bulk-ip-validation-assignment'
        }
      })
    } finally {
      setAssigningIPs(false)
    }
  }

  const handleOpenFlavorDialog = () => {
    if (selectedVMs.size === 0) return
    setFlavorDialogOpen(true)
  }

  const handleOpenRdmConfigurationDialog = () => {
    if (selectedVMs.size === 0) return
    setRdmConfigDialogOpen(true)
  }

  const handleOpenBulkIPAssignment = () => {
    if (selectedVMs.size === 0) return

    // Initialize bulk edit IPs for selected VMs
    const initialBulkEditIPs: Record<string, Record<number, string>> = {}
    const initialValidationStatus: Record<
      string,
      Record<number, 'empty' | 'valid' | 'invalid' | 'validating'>
    > = {}

    Array.from(selectedVMs).forEach((vmName) => {
      const vm = vmsWithFlavor.find((v) => v.name === vmName)
      if (!vm) {
        return
      }

      initialBulkEditIPs[vmName] = {}
      initialValidationStatus[vmName] = {}

      if (vm.networkInterfaces && vm.networkInterfaces.length > 0) {
        // Multiple network interfaces
        vm.networkInterfaces.forEach((nic, index) => {
          initialBulkEditIPs[vmName][index] = nic.ipAddress || ''
          initialValidationStatus[vmName][index] = nic.ipAddress ? 'valid' : 'empty'
        })
      } else {
        // Single interface (treat as interface 0)
        initialBulkEditIPs[vmName][0] = vm.ipAddress && vm.ipAddress !== '—' ? vm.ipAddress : ''
        initialValidationStatus[vmName][0] =
          vm.ipAddress && vm.ipAddress !== '—' ? 'valid' : 'empty'
      }
    })

    setBulkEditIPs(initialBulkEditIPs)
    setBulkValidationStatus(initialValidationStatus)
    setBulkValidationMessages({})
    setBulkEditDialogOpen(true)
  }

  const handleCloseRdmConfigurationDialog = () => {
    setRdmConfigDialogOpen(false)
  }

  const handleCloseFlavorDialog = () => {
    setFlavorDialogOpen(false)
    setSelectedFlavor('')
  }

  const handleApplyFlavor = async () => {
    if (!selectedFlavor) {
      handleCloseFlavorDialog()
      return
    }

    setUpdating(true)

    try {
      const isAutoAssign = selectedFlavor === 'auto-assign'
      const selectedFlavorObj = !isAutoAssign
        ? openstackFlavors.find((f) => f.id === selectedFlavor)
        : null
      const flavorName = isAutoAssign
        ? 'auto-assign'
        : selectedFlavorObj
          ? selectedFlavorObj.name
          : selectedFlavor

      const updatedVms = vmsWithFlavor.map((vm) => {
        if (selectedVMs.has(vm.name)) {
          return {
            ...vm,
            targetFlavorId: isAutoAssign ? '' : selectedFlavor,
            flavorName,
            // If a flavor is assigned, the VM no longer has a flavor not found issue
            flavorNotFound: isAutoAssign ? vm.flavorNotFound : false
          }
        }
        return vm
      })

      const selectedVmNames = Array.from(selectedVMs)

      const updatePromises = selectedVmNames.map((vmName) => {
        const vmwareMachineName = vmList.find((vm) => vm.name === vmName)?.vmWareMachineName
        const payload = {
          spec: {
            targetFlavorId: isAutoAssign ? '' : selectedFlavor
          }
        }
        if (!vmwareMachineName) {
          return
        }
        return patchVMwareMachine(vmwareMachineName, payload)
      })

      await Promise.all(updatePromises)

      setVmsWithFlavor(updatedVms)
      onChange('vms')(updatedVms.filter((vm) => selectedVMs.has(vm.name)))

      const actionText = isAutoAssign ? 'cleared flavor assignment for' : 'assigned flavor to'
      setSnackbarMessage(
        `Successfully ${actionText} ${selectedVmNames.length} VM${
          selectedVmNames.length > 1 ? 's' : ''
        }`
      )
      setSnackbarSeverity('success')
      setSnackbarOpen(true)

      refreshVMList()

      handleCloseFlavorDialog()
    } catch (error) {
      reportError(error as Error, {
        context: 'vm-flavors-update',
        metadata: {
          selectedVMs: Array.from(selectedVMs),
          selectedFlavor: selectedFlavor,
          isAutoAssign: selectedFlavor === 'auto-assign',
          action: 'vm-flavors-bulk-update'
        }
      })
      setSnackbarMessage('Failed to assign flavor to VMs')
      setSnackbarSeverity('error')
      setSnackbarOpen(true)
    } finally {
      setUpdating(false)
    }
  }

  // RDM disk configuration functions
  const handleApplyRdmConfigurations = async () => {
    if (!rdmConfigurations || rdmConfigurations.length === 0) {
      showToast('No RDM configurations to apply', 'warning')
      return
    }

    setUpdating(true)

    try {
      track('rdm_configuration_applied', {
        rdmDisksCount: rdmConfigurations.length,
        selectedVMsCount: selectedVMs.size
      })

      const updatePromises = rdmConfigurations.map(async (config) => {
        // Find the RDM disk by uuid
        const rdmDisk = rdmDisks.find((disk) => disk.spec.uuid === config.uuid)
        if (!rdmDisk) {
          console.warn(`RDM disk not found for diskName: ${config.diskName}`)
          return
        }

        const payload = {
          spec: {
            openstackVolumeRef: {
              cinderBackendPool: config.cinderBackendPool,
              volumeType: config.volumeType,
              openstackCreds: openstackCredName
            }
          }
        } as Partial<RdmDisk>

        return patchRdmDisk(rdmDisk.metadata.name, payload)
      })

      await Promise.all(updatePromises)

      showToast(
        `Successfully configured ${rdmConfigurations.length} RDM disk${
          rdmConfigurations.length > 1 ? 's' : ''
        }`,
        'success'
      )

      // Close the dialog after successful configuration
      handleCloseRdmConfigurationDialog()

      // Invalidate RDM disks query to refetch updated configuration and re-run validation
      queryClient.invalidateQueries({ queryKey: [RDM_DISKS_BASE_KEY] })
    } catch (error) {
      reportError(error as Error, {
        context: 'rdm-disk-configuration',
        metadata: {
          rdmConfigurationsCount: rdmConfigurations.length,
          action: 'apply-rdm-configurations'
        }
      })
      showToast('Failed to configure RDM disks', 'error')
    } finally {
      setUpdating(false)
    }
  }

  const handleCloseSnackbar = () => {
    setSnackbarOpen(false)
  }

  const isRowSelectable = (params) => {
    // Allow selection for both running and stopped VMs for cold migration
    // Only disable if VM is already migrated
    return !params.row.isMigrated
  }

  const getNoRowsLabel = () => {
    return 'No VMs discovered'
  }

  const rowSelectionModelArray = React.useMemo(
    () =>
      Array.from(selectedVMs).filter((vmName) => vmsWithFlavor.some((vm) => vm.name === vmName)),
    [selectedVMs, vmsWithFlavor]
  )

  return (
    <VmsSelectionStepContainer>
      {showHeader ? <Step stepNumber="2" label="Select Virtual Machines to Migrate" /> : null}
      <FieldsContainer>
        {rdmValidation.hasRdmVMs && (
          <Alert severity="info" sx={{ mb: 2 }}>
            <Box>
              <Typography variant="body2" sx={{ fontWeight: 'medium', mb: 1 }}>
                RDM (Raw Device Mapping) Migration Detected
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Try and migrate VMs with shared RDM disks together as a group. You can configure
                Cinder backend settings for these disks using the "Configure RDM" button.
              </Typography>
            </Box>
          </Alert>
        )}
        <Box sx={{ mb: 1 }}>
          <FieldLabel label="Virtual Machines" required align="flex-start" />
        </Box>
        <FormControl error={!!error} required>
          <Paper sx={{ width: '100%', height: 389 }}>
            <DataGrid
              rows={vmsWithFlavor}
              columns={columns}
              initialState={{
                pagination: { paginationModel },
                sorting: {
                  sortModel: [{ field: 'vmState', sort: 'asc' }]
                },
                columns: {
                  columnVisibilityModel: {
                    vmState: false, // Hide the vmState column that we use only for sorting
                    rdmDisks: false, // Hide the RDM disks column by default
                    networks: false, // Hide the networks column by default
                    esxHost: false // Hide the esxHost column by default
                  }
                }
              }}
              pageSizeOptions={[5, 10, 25]}
              localeText={{ noRowsLabel: getNoRowsLabel() }}
              rowHeight={45}
              onRowSelectionModelChange={handleVmSelection}
              rowSelectionModel={rowSelectionModelArray}
              getRowId={(row) => row.name}
              isRowSelectable={isRowSelectable}
              disableRowSelectionOnClick
              slots={{
                toolbar: (props) => {
                  return (
                    <CustomToolbarWithActions
                      {...props}
                      onRefresh={() => refreshVMList()}
                      disableRefresh={
                        loadingVms ||
                        loadingMigratedVms ||
                        !vmwareCredsValidated ||
                        !openstackCredsValidated
                      }
                      placeholder="Search by Name, Network Interface, CPU, or Memory"
                      rowSelectionModel={rowSelectionModelArray}
                      onAssignFlavor={handleOpenFlavorDialog}
                      onAssignRdmConfiguration={handleOpenRdmConfigurationDialog}
                      hasRdmVMs={rdmValidation.hasRdmVMs}
                      onAssignIP={handleOpenBulkIPAssignment}
                      selectedCount={rowSelectionModelArray.length}
                    />
                  )
                },
                loadingOverlay: () => <CustomLoadingOverlay loadingMessage="Loading VMs ..." />,
                row: (props) => {
                  const isMigrated = props.row.isMigrated
                  const hasFlavorNotFound = props.row.flavorNotFound

                  let tooltipMessage = ''
                  if (isMigrated) {
                    tooltipMessage = MIGRATED_TOOLTIP_MESSAGE
                  } else if (hasFlavorNotFound) {
                    tooltipMessage = FLAVOR_NOT_FOUND_MESSAGE
                  }

                  return (
                    <Tooltip title={tooltipMessage} followCursor>
                      <span style={{ display: 'contents' }}>
                        <GridRow {...props} />
                      </span>
                    </Tooltip>
                  )
                }
              }}
              loading={loadingVms || loadingMigratedVms}
              checkboxSelection
              disableColumnMenu
              getRowClassName={(params) => {
                if (params.row.isMigrated) {
                  return 'disabled-row'
                } else {
                  return ''
                }
              }}
              keepNonExistentRowsSelected
            />
          </Paper>
        </FormControl>
        {error && <FormHelperText error>{error}</FormHelperText>}
        {/* Separate RDM Error Messages */}
        {rdmValidation.hasSelectionError && (
          <Alert severity="error" sx={{ mt: 2 }}>
            {rdmValidation.selectionErrorMessage}
          </Alert>
        )}

        {rdmValidation.hasPowerStateError && (
          <Alert severity="error" sx={{ mt: 2 }}>
            {rdmValidation.powerStateErrorMessage}
          </Alert>
        )}

        {/* GPU Warning Message */}
        {(() => {
          const selectedVmsData = vmsWithFlavor.filter((vm) => selectedVMs.has(vm.name))
          const hasGPUVMs = selectedVmsData.some((vm) => (vm as any).useGPU)
          const hasAssignedFlavors = selectedVmsData.some((vm) => vm.targetFlavorId)

          if (hasGPUVMs && !useGPU && !hasAssignedFlavors) {
            return (
              <Alert severity="warning" sx={{ mt: 2 }}>
                You have selected VMs with GPU enabled. Please assign appropriate flavour or select
                "Use GPU enabled flavours" checkbox in Migration Options.
              </Alert>
            )
          }
          return null
        })()}
      </FieldsContainer>

      {/* Flavor Assignment Dialog */}
      <Dialog open={flavorDialogOpen} onClose={handleCloseFlavorDialog} fullWidth maxWidth="sm">
        <DialogTitle>
          Assign Flavor to {selectedVMs.size} {selectedVMs.size === 1 ? 'VM' : 'VMs'}
        </DialogTitle>
        <DialogContent>
          <Box sx={{ my: 2 }}>
            <FieldLabel label="Select Flavor" align="flex-start" />
            <Autocomplete
              sx={{ mt: 1 }}
              size="small"
              options={[
                { id: 'auto-assign', name: 'Auto-assign', vcpus: 0, ram: 0, disk: 0 },
                ...openstackFlavors
              ]}
              value={
                selectedFlavor
                  ? ([
                      { id: 'auto-assign', name: 'Auto-assign', vcpus: 0, ram: 0, disk: 0 },
                      ...openstackFlavors
                    ].find((f) => f.id === selectedFlavor) ?? null)
                  : null
              }
              onChange={(_e, value) => {
                setSelectedFlavor(value?.id ?? '')
              }}
              getOptionLabel={(option) => {
                if (option.id === 'auto-assign') return option.name
                return `${option.name} (${option.vcpus} vCPU, ${option.ram}MB RAM, ${option.disk}GB Disk)`
              }}
              isOptionEqualToValue={(option, value) => option.id === value.id}
              renderInput={(params) => (
                <SharedTextField {...params} placeholder="Search flavors" fullWidth />
              )}
            />
          </Box>
        </DialogContent>
        <DialogActions
          sx={{ justifyContent: 'flex-end', alignItems: 'center', gap: 1, px: 3, py: 2 }}
        >
          <ActionButton tone="secondary" onClick={handleCloseFlavorDialog} disabled={updating}>
            Cancel
          </ActionButton>
          <ActionButton
            tone="primary"
            onClick={handleApplyFlavor}
            disabled={!selectedFlavor || updating}
            loading={updating}
          >
            Apply to selected VMs
          </ActionButton>
        </DialogActions>
      </Dialog>

      {/* RDM Configuration Dialog */}
      <Dialog
        open={rdmConfigDialogOpen}
        onClose={handleCloseRdmConfigurationDialog}
        fullWidth
        maxWidth="md"
      >
        <DialogTitle>
          Configure RDM Disks for {selectedVMs.size} {selectedVMs.size === 1 ? 'VM' : 'VMs'}
        </DialogTitle>
        <DialogContent>
          {/* Debug info */}
          {process.env.NODE_ENV === 'development' && (
            <Box sx={{ mb: 2, p: 1, bgcolor: 'grey.100', fontSize: '0.8rem' }}>
              Debug: hasRdmVMs={String(rdmValidation.hasRdmVMs)}, rdmDisks.length={rdmDisks.length},
              loading={String(rdmDisksLoading)}
            </Box>
          )}
          {rdmDisksLoading ? (
            <Box sx={{ p: 2, textAlign: 'center' }}>
              <CircularProgress size={24} sx={{ mr: 1 }} />
              <Typography color="text.secondary">Loading RDM disk information...</Typography>
            </Box>
          ) : rdmValidation.hasRdmVMs && rdmDisks.length > 0 ? (
            <RdmDiskConfigurationPanel
              rdmDisks={rdmDisks.filter((disk) => {
                // Only show RDM disks that have at least one selected VM as owner
                const selectedVMsArray = Array.from(selectedVMs)
                return disk.spec.ownerVMs.some((ownerVM) => selectedVMsArray.includes(ownerVM))
              })}
              openstackCreds={openstackCredentials}
              selectedVMs={Array.from(selectedVMs)}
              onConfigurationChange={setRdmConfigurations}
            />
          ) : (
            <Box sx={{ p: 2, textAlign: 'center' }}>
              <Typography color="text.secondary">
                {rdmValidation.hasRdmVMs
                  ? 'No RDM disk configurations found.'
                  : 'No RDM disks detected for selected VMs.'}
              </Typography>
            </Box>
          )}
        </DialogContent>
        <DialogActions
          sx={{ justifyContent: 'flex-end', alignItems: 'center', gap: 1, px: 3, py: 2 }}
        >
          <ActionButton
            tone="secondary"
            onClick={handleCloseRdmConfigurationDialog}
            disabled={updating}
          >
            Close
          </ActionButton>
          {rdmValidation.hasRdmVMs && rdmDisks.length > 0 && (
            <ActionButton
              tone="primary"
              onClick={handleApplyRdmConfigurations}
              disabled={
                updating ||
                !rdmConfigurations ||
                rdmConfigurations.length === 0 ||
                rdmConfigurations.some((config) => !config.cinderBackendPool || !config.volumeType)
              }
              loading={updating}
            >
              Apply RDM Configuration
            </ActionButton>
          )}
        </DialogActions>
      </Dialog>

      <Snackbar
        open={snackbarOpen}
        autoHideDuration={6000}
        onClose={handleCloseSnackbar}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
      >
        <Alert onClose={handleCloseSnackbar} severity={snackbarSeverity}>
          {snackbarMessage}
        </Alert>
      </Snackbar>

      {/* Bulk IP Editor Dialog */}
      <Dialog open={bulkEditDialogOpen} onClose={handleCloseBulkEditDialog} maxWidth="md">
        <DialogTitle>
          Edit IP Addresses for {selectedVMs.size} {selectedVMs.size === 1 ? 'VM' : 'VMs'}
        </DialogTitle>
        <DialogContent dividers>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
            {/* Quick Actions */}
            <Box sx={{ display: 'flex', justifyContent: { xs: 'flex-start', sm: 'flex-end' } }}>
              <Button size="small" variant="outlined" onClick={handleClearAllIPs}>
                Clear All
              </Button>
            </Box>

            {/* IP Editor Fields */}
            <Box
              sx={{
                maxHeight: 420,
                overflowY: 'auto',
                pr: 1,
                display: 'flex',
                flexDirection: 'column',
                gap: 2
              }}
            >
              {Object.entries(bulkEditIPs).map(([vmName, interfaces]) => {
                const vm = vmsWithFlavor.find((v) => v.name === vmName)
                if (!vm) return null

                return (
                  <Box
                    key={vmName}
                    sx={{
                      p: 2,
                      border: '1px solid',
                      borderColor: 'divider',
                      borderRadius: 1,
                      display: 'flex',
                      flexDirection: 'column',
                      gap: 2
                    }}
                  >
                    <Typography variant="body2" sx={{ fontWeight: 600 }}>
                      {vm.name}
                    </Typography>

                    {Object.entries(interfaces).map(([interfaceIndexStr, ip]) => {
                      const interfaceIndex = parseInt(interfaceIndexStr)
                      const networkInterface = vm.networkInterfaces?.[interfaceIndex]
                      const status = bulkValidationStatus[vmName]?.[interfaceIndex]
                      const message = bulkValidationMessages[vmName]?.[interfaceIndex]
                      return (
                        <Box
                          key={interfaceIndex}
                          sx={{
                            display: 'grid',
                            gridTemplateColumns: { xs: '1fr', sm: '220px 1fr' },
                            columnGap: { xs: 1.5, sm: 2 },
                            rowGap: 1,
                            alignItems: 'flex-start'
                          }}
                        >
                          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                            <Typography variant="body2" fontWeight={500}>
                              {networkInterface?.mac ||
                                networkInterface?.network ||
                                `Interface ${interfaceIndex + 1}`}
                            </Typography>
                            <Typography variant="caption" color="text.secondary">
                              Current: {networkInterface?.ipAddress || vm.ipAddress || '—'}
                            </Typography>
                          </Box>
                          <SharedTextField
                            value={ip}
                            onChange={(e) =>
                              handleBulkIpChange(vmName, interfaceIndex, e.target.value)
                            }
                            placeholder="Enter IP address"
                            size="small"
                            fullWidth
                            error={status === 'invalid'}
                            helperText={message || ' '}
                            FormHelperTextProps={{ sx: { ml: 0 } }}
                            InputProps={{ endAdornment: renderValidationAdornment(status) }}
                          />
                        </Box>
                      )
                    })}
                  </Box>
                )
              })}
            </Box>
          </Box>
        </DialogContent>
        <DialogActions
          sx={{ justifyContent: 'flex-end', alignItems: 'center', gap: 1, px: 3, py: 2 }}
        >
          <ActionButton
            tone="secondary"
            onClick={handleCloseBulkEditDialog}
            disabled={assigningIPs}
          >
            Cancel
          </ActionButton>
          <ActionButton
            tone="primary"
            onClick={handleApplyBulkIPs}
            disabled={!hasBulkIpsToApply || assigningIPs || hasBulkIpValidationErrors}
            loading={assigningIPs}
          >
            Apply Changes
          </ActionButton>
        </DialogActions>
      </Dialog>

      {/* Add GlobalStyles similar to RollingMigrationForm */}
      <GlobalStyles
        styles={{
          '.MuiDataGrid-columnsManagement, .MuiDataGrid-columnsManagementPopover': {
            '& .MuiFormControlLabel-label': {
              fontSize: '0.875rem !important'
            },
            '& .MuiCheckbox-root': {
              padding: '4px !important'
            },
            '& .MuiListItem-root': {
              fontSize: '0.875rem !important',
              minHeight: '32px !important',
              padding: '2px 8px !important'
            },
            '& .MuiTypography-root': {
              fontSize: '0.875rem !important'
            },
            '& .MuiInputBase-input': {
              fontSize: '0.875rem !important'
            },
            '& .MuiTextField-root .MuiInputBase-input': {
              fontSize: '0.875rem !important'
            }
          }
        }}
      />

      {/* Toast Notification for IP assignments */}
      <Snackbar
        open={toastOpen}
        autoHideDuration={4000}
        onClose={handleCloseToast}
        anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
      >
        <Alert
          onClose={handleCloseToast}
          severity={toastSeverityIp}
          sx={{ width: '100%' }}
          variant="standard"
        >
          {toastMessage}
        </Alert>
      </Snackbar>
    </VmsSelectionStepContainer>
  )
}

const areOpenstackFlavorsEqual = (prev?: OpenStackFlavor[], next?: OpenStackFlavor[]): boolean => {
  if (prev === next) {
    return true
  }

  if (!prev || !next || prev.length !== next.length) {
    return false
  }

  return prev.every((prevFlavor, index) => {
    const nextFlavor = next[index]
    return (
      prevFlavor.id === nextFlavor.id &&
      prevFlavor.name === nextFlavor.name &&
      prevFlavor.disk === nextFlavor.disk &&
      prevFlavor.ram === nextFlavor.ram &&
      prevFlavor.vcpus === nextFlavor.vcpus
    )
  })
}

const arePropsEqual = (
  prevProps: VmsSelectionStepProps,
  nextProps: VmsSelectionStepProps
): boolean => {
  if (prevProps.onChange !== nextProps.onChange) {
    return false
  }

  if (prevProps.open !== nextProps.open) return false
  if (prevProps.error !== nextProps.error) return false
  if (prevProps.vmwareCredsValidated !== nextProps.vmwareCredsValidated) return false
  if (prevProps.openstackCredsValidated !== nextProps.openstackCredsValidated) return false
  if (prevProps.sessionId !== nextProps.sessionId) return false
  if (!areOpenstackFlavorsEqual(prevProps.openstackFlavors, nextProps.openstackFlavors))
    return false
  if (prevProps.vmwareCredName !== nextProps.vmwareCredName) return false
  if (prevProps.openstackCredName !== nextProps.openstackCredName) return false
  if (prevProps.openstackCredentials !== nextProps.openstackCredentials) return false
  if (prevProps.vmwareCluster !== nextProps.vmwareCluster) return false

  return true
}

export default React.memo(VmsSelectionStep, arePropsEqual)
