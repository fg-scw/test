import {
  Box,
  Typography,
  styled,
  Paper,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Alert,
  Select,
  MenuItem,
  GlobalStyles,
  FormLabel,
  Snackbar,
  useMediaQuery,
  Divider
} from '@mui/material'
import { ActionButton } from 'src/components'
import ClusterIcon from '@mui/icons-material/Hub'
import React, { useState, useMemo, useEffect, useCallback } from 'react'
import {
  DataGrid,
  GridColDef,
  GridRowSelectionModel,
  GridToolbarColumnsButton
} from '@mui/x-data-grid'
import { useNavigate } from 'react-router-dom'
import { useKeyboardSubmit } from 'src/hooks/ui/useKeyboardSubmit'
import { CustomSearchToolbar } from 'src/components/grid'
import SyntaxHighlighter from 'react-syntax-highlighter/dist/esm/prism'
import { oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { getVMwareHosts, patchVMwareHost } from 'src/api/vmware-hosts/vmwareHosts'
import { getVMwareMachines, patchVMwareMachine } from 'src/api/vmware-machines/vmwareMachines'
import { VMwareHost } from 'src/api/vmware-hosts/model'
import { VMwareMachine } from 'src/api/vmware-machines/model'
import { VJAILBREAK_DEFAULT_NAMESPACE } from 'src/api/constants'
import { getBMConfigList, getBMConfig } from 'src/api/bmconfig/bmconfig'
import { BMConfig } from 'src/api/bmconfig/model'
import MaasConfigDetailsModal from './components/MaasConfigDetailsModal'
import {
  getOpenstackCredentials,
  validateOpenstackIPs
} from 'src/api/openstack-creds/openstackCreds'
import { OpenstackCreds } from 'src/api/openstack-creds/model'
import NetworkAndStorageMappingStep, { ResourceMap } from './NetworkAndStorageMappingStep'
import {
  createRollingMigrationPlanJson,
  postRollingMigrationPlan,
  VMSequence,
  ClusterMapping
} from 'src/api/rolling-migration-plans'
import SourceDestinationClusterSelection from './SourceDestinationClusterSelection'
// Import required APIs for creating migration resources
import { createNetworkMappingJson } from 'src/api/network-mapping/helpers'
import { postNetworkMapping } from 'src/api/network-mapping/networkMappings'
import { createStorageMappingJson } from 'src/api/storage-mappings/helpers'
import { postStorageMapping } from 'src/api/storage-mappings/storageMappings'
import { createArrayCredsMappingJson } from 'src/api/arraycreds-mapping/helpers'
import { postArrayCredsMapping } from 'src/api/arraycreds-mapping/arrayCredsMapping'
import { createMigrationTemplateJson } from 'src/features/migration/api/migration-templates/helpers'
import {
  patchMigrationTemplate,
  postMigrationTemplate
} from 'src/features/migration/api/migration-templates/migrationTemplates'
import useParams from 'src/hooks/useParams'
import MigrationOptions from './MigrationOptionsAlt'
import { CUTOVER_TYPES } from './constants'
import WindowsIcon from 'src/assets/windows_icon.svg'
import LinuxIcon from 'src/assets/linux_icon.svg'
import WarningIcon from '@mui/icons-material/Warning'
import { useClusterData } from './useClusterData'
import { useErrorHandler } from 'src/hooks/useErrorHandler'

import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import ErrorIcon from '@mui/icons-material/Error'
import { TextField, CircularProgress } from '@mui/material'

// Import CDS icons
import '@cds/core/icon/register.js'
import { ClarityIcons, buildingIcon, clusterIcon, hostIcon, vmIcon } from '@cds/core/icon'
import { useAmplitude } from 'src/hooks/useAmplitude'
import { AMPLITUDE_EVENTS } from 'src/types/amplitude'

import { DrawerShell, DrawerHeader, DrawerFooter, SectionNav, SurfaceCard } from 'src/components'
import type { SectionNavItem } from 'src/components'
import { useTheme } from '@mui/material/styles'
import { FormProvider, useForm, useWatch } from 'react-hook-form'

// Define types for MigrationOptions
interface FormValues extends Record<string, unknown> {
  dataCopyMethod?: string
  dataCopyStartTime?: string
  cutoverOption?: string
  cutoverStartTime?: string
  cutoverEndTime?: string
  postMigrationScript?: string
  osFamily?: string
  storageCopyMethod?: 'normal' | 'StorageAcceleratedCopy'
}

type RollingMigrationRHFValues = {
  dataCopyStartTime: string
  cutoverStartTime: string
  cutoverEndTime: string
  postMigrationActionSuffix: string
  postMigrationActionFolderName: string
}

export interface SelectedMigrationOptionsType extends Record<string, unknown> {
  dataCopyMethod: boolean
  dataCopyStartTime: boolean
  cutoverOption: boolean
  cutoverStartTime: boolean
  cutoverEndTime: boolean
  postMigrationScript: boolean
  osFamily: boolean
  postMigrationAction?: {
    suffix?: boolean
    folderName?: boolean
    renameVm?: boolean
    moveToFolder?: boolean
  }
}

// Default state for checkboxes
const defaultMigrationOptions: SelectedMigrationOptionsType = {
  dataCopyMethod: false,
  dataCopyStartTime: false,
  cutoverOption: false,
  cutoverStartTime: false,
  cutoverEndTime: false,
  postMigrationScript: false,
  osFamily: false,
  postMigrationAction: {
    suffix: false,
    folderName: false,
    renameVm: false,
    moveToFolder: false
  }
}

type FieldErrors = { [formId: string]: string }

// Register clarity icons
ClarityIcons.addIcons(buildingIcon, clusterIcon, hostIcon, vmIcon)

// Style for Clarity icons
const CdsIconWrapper = styled('div')({
  marginRight: 8,
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center'
})

const drawerWidth = 1400

interface ESXHost {
  id: string
  name: string
  ip: string
  bmcIp: string
  maasState: string
  vms: number
  state: string
  pcdHostConfigName?: string
}

interface VM {
  id: string
  name: string
  ip: string
  esxHost: string
  networks?: string[]
  datastores?: string[]
  cpu?: number
  memory?: number
  powerState: string
  osFamily?: string
  flavor?: string
  targetFlavorId?: string
  ipValidationStatus?: 'pending' | 'valid' | 'invalid' | 'validating'
  ipValidationMessage?: string
  networkInterfaces?: VmNetworkInterface[]
}

export interface VmNetworkInterface {
  mac: string
  network: string
  ipAddress: string
}

// ESX columns will be defined inside the component

const CustomToolbarWithActions = (props) => {
  const { rowSelectionModel, onAssignFlavor, onAssignIP, hasPoweredOffVMs, ...toolbarProps } = props

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
            <ActionButton variant="text" color="primary" onClick={onAssignFlavor} size="small">
              Assign Flavor ({rowSelectionModel.length})
            </ActionButton>
            {hasPoweredOffVMs && (
              <ActionButton variant="text" color="primary" onClick={onAssignIP} size="small">
                Assign IP ({rowSelectionModel.length})
              </ActionButton>
            )}
          </>
        )}
        <CustomSearchToolbar {...toolbarProps} />
      </Box>
    </Box>
  )
}

const CustomESXToolbarWithActions = (props) => {
  const { onAssignHostConfig, ...toolbarProps } = props

  return (
    <Box
      sx={{
        display: 'flex',
        justifyContent: 'flex-end',
        width: '100%',
        padding: '4px 8px',
        gap: 1
      }}
    >
      <ActionButton variant="text" color="primary" onClick={onAssignHostConfig} size="small">
        Assign Host Config
      </ActionButton>
      <CustomSearchToolbar {...toolbarProps} />
    </Box>
  )
}

const MaasConfigDialog = styled(Dialog)({
  '& .MuiDialog-paper': {
    maxWidth: '900px',
    width: '100%'
  }
})

const ConfigSection = styled(Box)(({ theme }) => ({
  marginBottom: theme.spacing(3)
}))

const ConfigField = styled(Box)(({ theme }) => ({
  display: 'flex',
  gap: theme.spacing(1),
  marginBottom: theme.spacing(1.5)
}))

const FieldLabel = styled(Typography)(({ theme }) => ({
  fontWeight: 500,
  minWidth: '140px',
  color: theme.palette.text.secondary
}))

const FieldValue = styled(Typography)(({ theme }) => ({
  fontWeight: 400,
  color: theme.palette.text.primary
}))

const CodeEditorContainer = styled(Box)(({ theme }) => ({
  border: `1px solid ${theme.palette.grey[300]}`,
  borderRadius: theme.shape.borderRadius,
  overflow: 'auto',
  position: 'relative',
  resize: 'vertical',
  minHeight: '250px',
  maxHeight: '400px',
  backgroundColor: theme.palette.common.white,
  '& pre': {
    margin: 0,
    borderRadius: 0,
    height: '100%',
    overflow: 'auto',
    fontSize: '14px'
  },
  '&::-webkit-scrollbar': {
    width: '8px',
    height: '8px'
  },
  '&::-webkit-scrollbar-thumb': {
    backgroundColor: theme.palette.grey[300],
    borderRadius: '4px'
  },
  '&::-webkit-scrollbar-track': {
    backgroundColor: theme.palette.grey[100]
  }
}))

interface RollingMigrationFormDrawerProps {
  open: boolean
  onClose: () => void
}

export default function RollingMigrationFormDrawer({
  open,
  onClose
}: RollingMigrationFormDrawerProps) {
  const navigate = useNavigate()
  const { reportError } = useErrorHandler({ component: 'RollingMigrationForm' })
  const { track } = useAmplitude({ component: 'RollingMigrationForm' })
  const [sourceCluster, setSourceCluster] = useState('')
  const [destinationPCD, setDestinationPCD] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const [selectedVMwareCredName, setSelectedVMwareCredName] = useState('')

  const [selectedPcdCredName, setSelectedPcdCredName] = useState('')

  const [selectedVMs, setSelectedVMs] = useState<GridRowSelectionModel>([])
  const [pcdHostConfigDialogOpen, setPcdHostConfigDialogOpen] = useState(false)
  const [selectedPcdHostConfig, setSelectedPcdHostConfig] = useState('')
  const [updatingPcdMapping, setUpdatingPcdMapping] = useState(false)

  const [loadingHosts, setLoadingHosts] = useState(false)
  const [loadingVMs, setLoadingVMs] = useState(false)

  const [orderedESXHosts, setOrderedESXHosts] = useState<ESXHost[]>([])
  const [vmsWithAssignments, setVmsWithAssignments] = useState<VM[]>([])

  const [maasConfigDialogOpen, setMaasConfigDialogOpen] = useState(false)
  const [maasConfigs, setMaasConfigs] = useState<BMConfig[]>([])
  const [selectedMaasConfig, setSelectedMaasConfig] = useState<BMConfig | null>(null)
  const [loadingMaasConfig, setLoadingMaasConfig] = useState(false)
  const [maasDetailsModalOpen, setMaasDetailsModalOpen] = useState(false)

  const [networkMappings, setNetworkMappings] = useState<ResourceMap[]>([])
  const [storageMappings, setStorageMappings] = useState<ResourceMap[]>([])
  const [arrayCredsMappings, setArrayCredsMappings] = useState<ResourceMap[]>([])
  const [networkMappingError, setNetworkMappingError] = useState<string>('')
  const [storageMappingError, setStorageMappingError] = useState<string>('')

  const [openstackCredData, setOpenstackCredData] = useState<OpenstackCreds | null>(null)
  const [loadingOpenstackDetails, setLoadingOpenstackDetails] = useState(false)

  // IP editing and validation state - updated for multiple interfaces
  // IP editing and validation state removed - using bulk assignment instead

  // OS assignment state
  const [vmOSAssignments, setVmOSAssignments] = useState<Record<string, string>>({})
  const [osValidationError, setOsValidationError] = useState<string>('')

  // Migration Options state
  const { params, getParamsUpdater } = useParams<FormValues>({})
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
  const getFieldErrorsUpdater = useCallback(
    (key: string | number) => (value: string) => {
      setFieldErrors((prev) => ({ ...prev, [key]: value }))
    },
    []
  )

  const rhfForm = useForm<RollingMigrationRHFValues, any, RollingMigrationRHFValues>({
    defaultValues: {
      dataCopyStartTime: params.dataCopyStartTime ?? '',
      cutoverStartTime: params.cutoverStartTime ?? '',
      cutoverEndTime: params.cutoverEndTime ?? '',
      postMigrationActionSuffix: (params as any)?.postMigrationAction?.suffix ?? '',
      postMigrationActionFolderName: (params as any)?.postMigrationAction?.folderName ?? ''
    }
  })

  const rhfDataCopyStartTime = useWatch({ control: rhfForm.control, name: 'dataCopyStartTime' })
  const rhfCutoverStartTime = useWatch({ control: rhfForm.control, name: 'cutoverStartTime' })
  const rhfCutoverEndTime = useWatch({ control: rhfForm.control, name: 'cutoverEndTime' })
  const rhfPostMigrationActionSuffix = useWatch({
    control: rhfForm.control,
    name: 'postMigrationActionSuffix'
  })
  const rhfPostMigrationActionFolderName = useWatch({
    control: rhfForm.control,
    name: 'postMigrationActionFolderName'
  })
  const { params: selectedMigrationOptions, getParamsUpdater: updateSelectedMigrationOptions } =
    useParams<SelectedMigrationOptionsType>(defaultMigrationOptions)

  const { sourceData, pcdData, loadingVMware: loading, loadingPCD } = useClusterData()
  const [assigningIPs, setAssigningIPs] = useState(false)

  // IP validation error state
  const [vmIpValidationError, setVmIpValidationError] = useState<string>('')

  // ESX host config validation error state
  const [esxHostConfigValidationError, setEsxHostConfigValidationError] = useState<string>('')

  // Bulk IP editing state - updated for multiple interfaces
  const [bulkEditDialogOpen, setBulkEditDialogOpen] = useState(false)
  const [bulkEditIPs, setBulkEditIPs] = useState<Record<string, Record<number, string>>>({})
  const [bulkValidationStatus, setBulkValidationStatus] = useState<
    Record<string, Record<number, 'empty' | 'valid' | 'invalid' | 'validating'>>
  >({})
  const [bulkValidationMessages, setBulkValidationMessages] = useState<
    Record<string, Record<number, string>>
  >({}) // Updated for multiple interfaces

  const hasBulkIpValidationErrors = useMemo(() => {
    return Object.values(bulkValidationStatus).some((interfaces) =>
      Object.values(interfaces || {}).some((status) => status === 'invalid')
    )
  }, [bulkValidationStatus])

  const hasBulkIpsToApply = useMemo(() => {
    return Object.values(bulkEditIPs).some((interfaces) =>
      Object.values(interfaces || {}).some((ip) => Boolean(ip?.trim()))
    )
  }, [bulkEditIPs])

  // Flavor assignment state
  const [flavorDialogOpen, setFlavorDialogOpen] = useState(false)
  const [selectedFlavor, setSelectedFlavor] = useState('')
  const [updating, setUpdating] = useState(false)

  // Toast notification state
  const [toastOpen, setToastOpen] = useState(false)
  const [toastMessage] = useState('')
  const [toastSeverity] = useState<'success' | 'error' | 'warning' | 'info'>('success')

  const paginationModel = { page: 0, pageSize: 5 }

  const handleCloseToast = useCallback((_event?: React.SyntheticEvent | Event, reason?: string) => {
    if (reason === 'clickaway') {
      return
    }
    setToastOpen(false)
  }, [])

  // Clear selection when component is closed
  useEffect(() => {
    if (!open) {
      setSelectedVMs([])
    }
  }, [open])

  useEffect(() => {
    if (open) {
      fetchMaasConfigs()
    }
  }, [open])

  useEffect(() => {
    const nextDataCopyStartTime = params.dataCopyStartTime ?? ''
    const nextCutoverStartTime = params.cutoverStartTime ?? ''
    const nextCutoverEndTime = params.cutoverEndTime ?? ''
    const nextPostMigrationActionSuffix = (params as any)?.postMigrationAction?.suffix ?? ''
    const nextPostMigrationActionFolderName = (params as any)?.postMigrationAction?.folderName ?? ''

    const currentDataCopyStartTime = rhfForm.getValues('dataCopyStartTime') ?? ''
    const currentCutoverStartTime = rhfForm.getValues('cutoverStartTime') ?? ''
    const currentCutoverEndTime = rhfForm.getValues('cutoverEndTime') ?? ''
    const currentPostMigrationActionSuffix = rhfForm.getValues('postMigrationActionSuffix') ?? ''
    const currentPostMigrationActionFolderName =
      rhfForm.getValues('postMigrationActionFolderName') ?? ''

    if (currentDataCopyStartTime !== nextDataCopyStartTime) {
      rhfForm.setValue('dataCopyStartTime', nextDataCopyStartTime)
    }
    if (currentCutoverStartTime !== nextCutoverStartTime) {
      rhfForm.setValue('cutoverStartTime', nextCutoverStartTime)
    }
    if (currentCutoverEndTime !== nextCutoverEndTime) {
      rhfForm.setValue('cutoverEndTime', nextCutoverEndTime)
    }
    if (currentPostMigrationActionSuffix !== nextPostMigrationActionSuffix) {
      rhfForm.setValue('postMigrationActionSuffix', nextPostMigrationActionSuffix)
    }
    if (currentPostMigrationActionFolderName !== nextPostMigrationActionFolderName) {
      rhfForm.setValue('postMigrationActionFolderName', nextPostMigrationActionFolderName)
    }
  }, [
    params.dataCopyStartTime,
    params.cutoverStartTime,
    params.cutoverEndTime,
    (params as any)?.postMigrationAction?.suffix,
    (params as any)?.postMigrationAction?.folderName,
    rhfForm
  ])

  useEffect(() => {
    const next = String(rhfDataCopyStartTime ?? '')
    const normalized = next.trim() ? next.trim() : ''
    const current = String(params.dataCopyStartTime ?? '')
    if (normalized !== current) {
      getParamsUpdater('dataCopyStartTime')(normalized)
    }
  }, [getParamsUpdater, params.dataCopyStartTime, rhfDataCopyStartTime])

  useEffect(() => {
    const next = String(rhfCutoverStartTime ?? '')
    const normalized = next.trim() ? next.trim() : ''
    const current = String(params.cutoverStartTime ?? '')
    if (normalized !== current) {
      getParamsUpdater('cutoverStartTime')(normalized)
    }
  }, [getParamsUpdater, params.cutoverStartTime, rhfCutoverStartTime])

  useEffect(() => {
    const next = String(rhfCutoverEndTime ?? '')
    const normalized = next.trim() ? next.trim() : ''
    const current = String(params.cutoverEndTime ?? '')
    if (normalized !== current) {
      getParamsUpdater('cutoverEndTime')(normalized)
    }
  }, [getParamsUpdater, params.cutoverEndTime, rhfCutoverEndTime])

  useEffect(() => {
    const nextSuffix = String(rhfPostMigrationActionSuffix ?? '')
    const normalized = nextSuffix.trim() ? nextSuffix.trim() : ''
    const current = (params as any)?.postMigrationAction?.suffix ?? ''

    if (normalized !== current && selectedMigrationOptions?.postMigrationAction?.renameVm) {
      getParamsUpdater('postMigrationAction')({
        ...(params as any)?.postMigrationAction,
        suffix: normalized
      })
    }
  }, [
    getParamsUpdater,
    (params as any)?.postMigrationAction,
    rhfPostMigrationActionSuffix,
    selectedMigrationOptions?.postMigrationAction?.renameVm
  ])

  useEffect(() => {
    const nextFolderName = String(rhfPostMigrationActionFolderName ?? '')
    const normalized = nextFolderName.trim() ? nextFolderName.trim() : ''
    const current = (params as any)?.postMigrationAction?.folderName ?? ''

    if (normalized !== current && selectedMigrationOptions?.postMigrationAction?.moveToFolder) {
      getParamsUpdater('postMigrationAction')({
        ...(params as any)?.postMigrationAction,
        folderName: normalized
      })
    }
  }, [
    getParamsUpdater,
    (params as any)?.postMigrationAction,
    rhfPostMigrationActionFolderName,
    selectedMigrationOptions?.postMigrationAction?.moveToFolder
  ])

  const fetchMaasConfigs = async () => {
    try {
      setLoadingMaasConfig(true)
      const configs = await getBMConfigList(VJAILBREAK_DEFAULT_NAMESPACE)
      if (configs && configs.length > 0) {
        setMaasConfigs(configs)
        try {
          const config = await getBMConfig(configs[0].metadata.name, VJAILBREAK_DEFAULT_NAMESPACE)
          setSelectedMaasConfig(config)
        } catch (error) {
          console.error(`Failed to fetch Bare Metal config:`, error)
        }
      }
    } catch (error) {
      console.error('Failed to fetch Bare Metal configs:', error)
    } finally {
      setLoadingMaasConfig(false)
    }
  }

  useEffect(() => {
    if (sourceCluster) {
      fetchClusterHosts()
      fetchClusterVMs()
    }
  }, [sourceCluster])

  const fetchClusterHosts = async () => {
    if (!sourceCluster) return

    setLoadingHosts(true)
    try {
      const parts = sourceCluster.split(':')
      const credName = parts[0]

      const sourceItem = sourceData.find((item) => item.credName === credName)
      const clusterObj = sourceItem?.clusters.find((cluster) => cluster.id === sourceCluster)
      const clusterName = clusterObj?.name

      if (!clusterName) {
        setOrderedESXHosts([])
        setLoadingHosts(false)
        return
      }

      const hostsResponse = await getVMwareHosts(
        VJAILBREAK_DEFAULT_NAMESPACE,
        // credName,
        '',
        clusterName
      )

      const mappedHosts: ESXHost[] = hostsResponse.items.map((host: VMwareHost) => ({
        id: host.metadata.name,
        name: host.spec.name,
        ip: '',
        bmcIp: '',
        maasState: 'Unknown',
        vms: 0,
        state: 'Active'
      }))

      setOrderedESXHosts(mappedHosts)
    } catch (error) {
      console.error('Failed to fetch cluster hosts:', error)
    } finally {
      setLoadingHosts(false)
    }
  }

  const fetchClusterVMs = async () => {
    if (!sourceCluster) return

    setLoadingVMs(true)
    try {
      const parts = sourceCluster.split(':')
      const credName = parts[0]

      const sourceItem = sourceData.find((item) => item.credName === credName)
      const clusterObj = sourceItem?.clusters.find((cluster) => cluster.id === sourceCluster)
      const clusterName = clusterObj?.name

      if (!clusterName) {
        setVmsWithAssignments([])
        setLoadingVMs(false)
        return
      }

      const vmsResponse = await getVMwareMachines(VJAILBREAK_DEFAULT_NAMESPACE, credName)

      const filteredVMs = vmsResponse.items.filter((vm: VMwareMachine) => {
        const clusterLabel = vm.metadata?.labels?.[`vjailbreak.k8s.pf9.io/vmware-cluster`]
        return clusterLabel === clusterName
      })

      const mappedVMs: VM[] = filteredVMs.map((vm: VMwareMachine) => {
        const esxiHost = vm.metadata?.labels?.[`vjailbreak.k8s.pf9.io/esxi-name`] || ''

        // Get flavor information from the VM spec
        const targetFlavorId = vm.spec.targetFlavorId || ''
        // We'll resolve flavor names later when openstackFlavors is available
        const flavorName = targetFlavorId || 'auto-assign'

        if (vm.spec.vms.name == 'nvidia-bcm-router') {
          console.log(vm.spec.vms.networkInterfaces)
        }

        // Get all IP addresses from network interfaces in comma-separated format
        const allIPs =
          vm.spec.vms.networkInterfaces && vm.spec.vms.networkInterfaces.length > 0
            ? vm.spec.vms.networkInterfaces
                .map((nic) => nic.ipAddress)
                .filter((ip) => ip && ip.trim() !== '') // Filter out empty/null IPs
                .join(', ')
            : vm.spec.vms.ipAddress || vm.spec.vms.assignedIp || '—'

        return {
          id: vm.metadata.name,
          name: vm.spec.vms.name || vm.metadata.name,
          ip: allIPs || '—',
          esxHost: esxiHost,
          networks: vm.spec.vms.networks,
          datastores: vm.spec.vms.datastores,
          cpu: vm.spec.vms.cpu,
          memory: vm.spec.vms.memory,
          osFamily: vm.spec.vms.osFamily,
          flavor: flavorName,
          targetFlavorId: targetFlavorId,
          powerState: vm.status.powerState === 'running' ? 'powered-on' : 'powered-off',
          ipValidationStatus: 'pending',
          ipValidationMessage: '',
          networkInterfaces: vm.spec.vms.networkInterfaces
        }
      })

      setVmsWithAssignments(mappedVMs)

      // Clean up persistent selection - remove VMs that no longer exist
      const availableVmIds = new Set(mappedVMs.map((vm) => vm.id))
      const cleanedSelection = selectedVMs.filter((vmId) => availableVmIds.has(String(vmId)))

      if (cleanedSelection.length !== selectedVMs.length) {
        setSelectedVMs(cleanedSelection)
      }
    } catch (error) {
      console.error('Failed to fetch cluster VMs:', error)
      setVmsWithAssignments([])
    } finally {
      setLoadingVMs(false)
    }
  }

  useEffect(() => {
    if (orderedESXHosts.length > 0 && vmsWithAssignments.length > 0) {
      const esxHostOrder = new Map()
      orderedESXHosts.forEach((host, index) => {
        esxHostOrder.set(host.id, index)
      })

      const sortedVMs = [...vmsWithAssignments].sort((a, b) => {
        const aHostIndex = esxHostOrder.get(a.esxHost) ?? 999
        const bHostIndex = esxHostOrder.get(b.esxHost) ?? 999
        return aHostIndex - bHostIndex
      })

      setVmsWithAssignments(sortedVMs)
    }
  }, [orderedESXHosts])

  const handleCloseMaasConfig = () => {
    setMaasConfigDialogOpen(false)
  }

  const handleSourceClusterChange = (value) => {
    markTouched('sourceDestination')
    setSourceCluster(value)

    if (value) {
      const parts = value.split(':')
      const credName = parts[0]
      setSelectedVMwareCredName(credName)
    } else {
      setSelectedVMwareCredName('')
    }
  }

  const handleDestinationPCDChange = (value) => {
    markTouched('sourceDestination')
    setDestinationPCD(value)

    if (value) {
      const selectedPCD = pcdData.find((p) => p.id === value)
      if (selectedPCD) {
        setSelectedPcdCredName(selectedPCD.openstackCredName)
        fetchOpenstackCredentialDetails(selectedPCD.openstackCredName)
      }
    } else {
      setSelectedPcdCredName('')
      setOpenstackCredData(null)
    }
  }

  const fetchOpenstackCredentialDetails = async (credName) => {
    if (!credName) return

    setLoadingOpenstackDetails(true)
    try {
      const response = await getOpenstackCredentials(credName)
      setOpenstackCredData(response)
    } catch (error) {
      console.error('Failed to fetch OpenStack credential details:', error)
    } finally {
      setLoadingOpenstackDetails(false)
    }
  }

  // IP validation and editing functions
  const isValidIPAddress = (ip: string): boolean => {
    const ipRegex =
      /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/
    return ipRegex.test(ip)
  }

  // Modal functions for multi-NIC IP editing removed - using bulk assignment instead

  // OS assignment handler
  const handleOSAssignment = async (vmId: string, osFamily: string) => {
    try {
      setVmOSAssignments((prev) => ({ ...prev, [vmId]: osFamily }))

      await patchVMwareMachine(
        vmId,
        {
          spec: {
            vms: {
              osFamily: osFamily
            }
          }
        },
        VJAILBREAK_DEFAULT_NAMESPACE
      )

      const updatedVMs = vmsWithAssignments.map((v) =>
        v.id === vmId ? { ...v, osFamily: osFamily } : v
      )
      setVmsWithAssignments(updatedVMs)
    } catch (error) {
      console.error('Failed to assign OS family:', error)
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
    }
  }

  const availableVmwareNetworks = useMemo(() => {
    if (!vmsWithAssignments.length || !selectedVMs.length) return []

    const selectedVMsData = vmsWithAssignments.filter((vm) => selectedVMs.includes(vm.id))

    const extractedNetworks = selectedVMsData
      .filter((vm) => vm.networks)
      .flatMap((vm) => vm.networks || [])

    if (extractedNetworks.length > 0) {
      return extractedNetworks.sort() // Remove Array.from(new Set()) to keep duplicates
    }
    return []
  }, [vmsWithAssignments, selectedVMs])

  const availableVmwareDatastores = useMemo(() => {
    if (!vmsWithAssignments.length || !selectedVMs.length) return []

    const selectedVMsData = vmsWithAssignments.filter((vm) => selectedVMs.includes(vm.id))

    const extractedDatastores = selectedVMsData
      .filter((vm) => vm.datastores)
      .flatMap((vm) => vm.datastores || [])

    if (extractedDatastores.length > 0) {
      return Array.from(new Set(extractedDatastores)).sort()
    }
    return []
  }, [vmsWithAssignments, selectedVMs])

  // Define ESX columns inside component to access state and functions
  const esxColumns: GridColDef[] = [
    {
      field: 'name',
      headerName: 'ESX Name',
      flex: 2,
      renderCell: (params) => (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <CdsIconWrapper>
            {/* eslint-disable-next-line @typescript-eslint/ban-ts-comment */}
            {/* @ts-ignore */}
            <cds-icon shape="host" size="md" badge="info"></cds-icon>
          </CdsIconWrapper>
          {params.value}
        </Box>
      )
    },
    {
      field: 'pcdHostConfigName',
      headerName: 'Host Config',
      flex: 1,
      renderCell: (params) => {
        const hostId = params.row.id
        const currentConfig = params.value || ''

        return (
          <Box sx={{ display: 'flex', alignItems: 'center', height: '100%' }}>
            <Select
              size="small"
              value={currentConfig}
              onChange={(e) => handleIndividualHostConfigChange(hostId, e.target.value)}
              displayEmpty
              sx={{
                width: 250,
                '& .MuiSelect-select': {
                  padding: '4px 8px',
                  fontSize: '0.875rem'
                }
              }}
              renderValue={(selected) => {
                if (!selected) {
                  return (
                    <Box
                      sx={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 1,
                        color: 'text.secondary'
                      }}
                    >
                      <WarningIcon sx={{ fontSize: 16 }} />
                      <em>Select Host Config</em>
                    </Box>
                  )
                }
                return <Typography variant="body2">{selected}</Typography>
              }}
            >
              <MenuItem value="">
                <Box
                  sx={{ display: 'flex', alignItems: 'center', gap: 1, color: 'text.secondary' }}
                >
                  <WarningIcon sx={{ fontSize: 16 }} />
                  <em>Select Host Config</em>
                </Box>
              </MenuItem>
              {(openstackCredData?.spec?.pcdHostConfig || []).map((config) => (
                <MenuItem key={config.id} value={config.name}>
                  <Box sx={{ display: 'flex', flexDirection: 'column' }}>
                    <Typography variant="body2">{config.name}</Typography>
                    <Typography variant="caption" color="text.secondary">
                      {config.mgmtInterface}
                    </Typography>
                  </Box>
                </MenuItem>
              ))}
            </Select>
          </Box>
        )
      }
    }
  ]

  // Calculate ESX host to PCD config mapping status
  const esxHostMappingStatus = useMemo(() => {
    const mappedHostsCount = orderedESXHosts.filter((host) => host.pcdHostConfigName).length
    return {
      mapped: mappedHostsCount,
      total: orderedESXHosts.length,
      fullyMapped: mappedHostsCount === orderedESXHosts.length
    }
  }, [orderedESXHosts])

  const openstackNetworks = useMemo(() => {
    if (!openstackCredData) return []

    const networks = openstackCredData?.status?.openstack?.networks || []
    return networks.sort((a, b) => a.toLowerCase().localeCompare(b.toLowerCase()))
  }, [openstackCredData])

  const openstackVolumeTypes = useMemo(() => {
    if (!openstackCredData) return []

    const volumeTypes = openstackCredData?.status?.openstack?.volumeTypes || []
    return volumeTypes.sort((a, b) => a.toLowerCase().localeCompare(b.toLowerCase()))
  }, [openstackCredData])

  const openstackFlavors = useMemo(() => {
    if (!openstackCredData) return []

    return openstackCredData?.spec?.flavors || []
  }, [openstackCredData])

  // Update VM flavor names when OpenStack flavors become available
  useEffect(() => {
    if (openstackFlavors.length > 0 && vmsWithAssignments.length > 0) {
      const updatedVMs = vmsWithAssignments.map((vm) => {
        if (vm.targetFlavorId) {
          const flavorObj = openstackFlavors.find((f) => f.id === vm.targetFlavorId)
          if (flavorObj && vm.flavor !== flavorObj.name) {
            return { ...vm, flavor: flavorObj.name }
          }
        }
        return vm
      })

      // Only update if there are actual changes
      const hasChanges = updatedVMs.some(
        (vm, index) => vm.flavor !== vmsWithAssignments[index]?.flavor
      )

      if (hasChanges) {
        setVmsWithAssignments(updatedVMs)
      }
    }
  }, [openstackFlavors, vmsWithAssignments])

  const handleMappingsChange = (key: string) => (value: unknown) => {
    markTouched('mapResources')

    if (!Array.isArray(value) && key !== 'storageCopyMethod') {
      return
    }

    switch (key) {
      case 'networkMappings': {
        const typed = value as ResourceMap[]
        setNetworkMappings(typed)
        getParamsUpdater('networkMappings')(typed)
        setNetworkMappingError('')
        break
      }
      case 'storageMappings': {
        const typed = value as ResourceMap[]
        setStorageMappings(typed)
        getParamsUpdater('storageMappings')(typed)
        setStorageMappingError('')
        break
      }
      case 'arrayCredsMappings': {
        const typed = value as ResourceMap[]
        setArrayCredsMappings(typed)
        getParamsUpdater('arrayCredsMappings')(typed)
        setStorageMappingError('')
        break
      }
      case 'storageCopyMethod':
        if (typeof value === 'string') {
          getParamsUpdater('storageCopyMethod')(value)
        }
        break
      default:
        break
    }
  }

  // Validate IP addresses for selected VMs
  const vmIpValidation = useMemo(() => {
    if (selectedVMs.length === 0) {
      setVmIpValidationError('Please select VMs to assign IP addresses.')
      return { hasError: true, vmsWithoutIPs: [] }
    }

    const selectedVMsData = vmsWithAssignments.filter((vm) => selectedVMs.includes(vm.id))
    const vmsWithoutIPs = selectedVMsData.filter((vm) => vm.ip === '—' || !vm.ip)

    if (vmsWithoutIPs.length > 0) {
      const errorMessage = `Cannot proceed with Migration: ${vmsWithoutIPs.length} selected VM${vmsWithoutIPs.length === 1 ? '' : 's'} do not have IP addresses assigned. Please assign IP addresses to all selected VMs before continuing.`
      setVmIpValidationError(errorMessage)
      return { hasError: true, vmsWithoutIPs }
    } else {
      setVmIpValidationError('')
      return { hasError: false, vmsWithoutIPs: [] }
    }
  }, [selectedVMs, vmsWithAssignments])

  // Validate ESX host configs for all hosts
  const esxHostConfigValidation = useMemo(() => {
    if (orderedESXHosts.length === 0) {
      setEsxHostConfigValidationError('Please select VMs to migrate.')
      return { hasError: true, hostsWithoutConfigs: [] }
    }

    const hostsWithoutConfigs = orderedESXHosts.filter((host) => !host.pcdHostConfigName)

    if (hostsWithoutConfigs.length > 0) {
      const errorMessage = `Cannot proceed with Migration: ${hostsWithoutConfigs.length} ESXi host${hostsWithoutConfigs.length === 1 ? '' : 's'} do not have Host Config assigned. Please assign Host Config to all ESXi hosts before continuing.`
      setEsxHostConfigValidationError(errorMessage)
      return { hasError: true, hostsWithoutConfigs }
    } else {
      setEsxHostConfigValidationError('')
      return { hasError: false, hostsWithoutConfigs: [] }
    }
  }, [orderedESXHosts])

  // Validate OS assignment for selected powered-off VMs
  const osValidation = useMemo(() => {
    if (selectedVMs.length === 0) {
      setOsValidationError('Please select VMs to assign OS.')
      return { hasError: true, vmsWithoutOS: [] }
    }

    const selectedVMsData = vmsWithAssignments.filter((vm) => selectedVMs.includes(vm.id))
    const poweredOffVMsWithoutOS = selectedVMsData.filter((vm) => {
      const assignedOS = vmOSAssignments[vm.id]
      const currentOS = assignedOS || vm.osFamily
      return vm.powerState === 'powered-off' && (!currentOS || currentOS === 'Unknown')
    })

    if (poweredOffVMsWithoutOS.length > 0) {
      const errorMessage = `Cannot proceed with Migration: ${poweredOffVMsWithoutOS.length} powered-off VM${poweredOffVMsWithoutOS.length === 1 ? '' : 's'} do not have Operating System assigned. Please assign OS to all powered-off VMs before continuing.`
      setOsValidationError(errorMessage)
      return { hasError: true, vmsWithoutOS: poweredOffVMsWithoutOS }
    } else {
      setOsValidationError('')
      return { hasError: false, vmsWithoutOS: [] }
    }
  }, [selectedVMs, vmsWithAssignments, vmOSAssignments])

  const handleSubmit = async () => {
    setSubmitting(true)

    const storageCopyMethod = (params.storageCopyMethod || 'normal') as
      | 'normal'
      | 'StorageAcceleratedCopy'

    if (selectedVMs.length > 0) {
      if (
        availableVmwareNetworks.some(
          (network) => !networkMappings.some((mapping) => mapping.source === network)
        )
      ) {
        setNetworkMappingError('All networks from selected VMs must be mapped')
        setSubmitting(false)
        return
      }

      if (storageCopyMethod === 'StorageAcceleratedCopy') {
        if (
          availableVmwareDatastores.some(
            (datastore) => !arrayCredsMappings.some((mapping) => mapping.source === datastore)
          )
        ) {
          setStorageMappingError('All datastores from selected VMs must be mapped')
          setSubmitting(false)
          return
        }
      } else {
        if (
          availableVmwareDatastores.some(
            (datastore) => !storageMappings.some((mapping) => mapping.source === datastore)
          )
        ) {
          setStorageMappingError('All datastores from selected VMs must be mapped')
          setSubmitting(false)
          return
        }
      }
    } else if (sourceCluster && destinationPCD) {
      alert('Please select at least one VM to migrate')
      setSubmitting(false)
      return
    }

    try {
      const parts = sourceCluster.split(':')
      const credName = parts[0]

      const sourceItem = sourceData.find((item) => item.credName === credName)
      const clusterObj = sourceItem?.clusters.find((cluster) => cluster.id === sourceCluster)
      const clusterName = clusterObj?.name || ''

      const selectedVMsData = vmsWithAssignments
        .filter((vm) => selectedVMs.includes(vm.id))
        .map((vm) => ({
          vmName: vm.name
        })) as VMSequence[]

      // Create cluster mapping between VMware cluster and PCD cluster
      const selectedPCD = pcdData.find((p) => p.id === destinationPCD)
      const pcdClusterName = selectedPCD?.name || ''
      const targetPCDClusterName = selectedPCD?.name

      const clusterMapping: ClusterMapping[] = [
        {
          vmwareClusterName: clusterName,
          pcdClusterName: pcdClusterName
        }
      ]

      // Update VMware hosts with their host config IDs
      const hostsToUpdate = orderedESXHosts.filter((host) => host.pcdHostConfigName)

      for (const host of hostsToUpdate) {
        try {
          // Find the config ID from the name
          const availablePcdHostConfigs = openstackCredData?.spec?.pcdHostConfig || []
          const selectedPcdConfig = availablePcdHostConfigs.find(
            (config) => config.name === host.pcdHostConfigName
          )
          const hostConfigId = selectedPcdConfig ? selectedPcdConfig.id : host.pcdHostConfigName

          if (hostConfigId) {
            console.log(`Updating host ${host.name} with hostConfigId: ${hostConfigId}`)
            await patchVMwareHost(host.id, hostConfigId, VJAILBREAK_DEFAULT_NAMESPACE)
          }
        } catch (error) {
          console.error(`Failed to update host config for ${host.name}:`, error)
          reportError(error as Error, {
            context: 'host-config-update',
            metadata: {
              hostId: host.id,
              hostName: host.name,
              hostConfigId: host.pcdHostConfigName,
              action: 'host-config-update'
            }
          })
          // Continue with other hosts even if one fails
        }
      }

      // 1. Create network mapping
      const networkMappingJson = createNetworkMappingJson({
        networkMappings: networkMappings.map((mapping) => ({
          source: mapping.source,
          target: mapping.target
        }))
      })
      const networkMappingResponse = await postNetworkMapping(networkMappingJson)

      // 2. Create storage mapping
      let storageMappingResponse: any = null
      let arrayCredsMappingResponse: any = null

      if (storageCopyMethod === 'StorageAcceleratedCopy') {
        const arrayCredsMappingJson = createArrayCredsMappingJson({
          mappings: arrayCredsMappings.map((mapping) => ({
            source: mapping.source,
            target: mapping.target
          }))
        })
        arrayCredsMappingResponse = await postArrayCredsMapping(arrayCredsMappingJson)
      } else {
        const storageMappingJson = createStorageMappingJson({
          storageMappings: storageMappings.map((mapping) => ({
            source: mapping.source,
            target: mapping.target
          }))
        })
        storageMappingResponse = await postStorageMapping(storageMappingJson)
      }

      // 3. Create migration template
      const migrationTemplateJson = createMigrationTemplateJson({
        vmwareRef: selectedVMwareCredName,
        openstackRef: selectedPcdCredName,
        networkMapping: networkMappingResponse.metadata.name,
        ...(storageMappingResponse?.metadata?.name && {
          storageMapping: storageMappingResponse.metadata.name
        }),
        targetPCDClusterName: targetPCDClusterName
      })
      const migrationTemplateResponse = await postMigrationTemplate(migrationTemplateJson)

      // Update template to include storageCopyMethod and mapping selection
      if (migrationTemplateResponse?.metadata?.name) {
        await patchMigrationTemplate(migrationTemplateResponse.metadata.name, {
          spec: {
            networkMapping: networkMappingResponse.metadata.name,
            storageCopyMethod,
            ...(storageCopyMethod === 'StorageAcceleratedCopy' &&
              arrayCredsMappingResponse?.metadata?.name && {
                arrayCredsMapping: arrayCredsMappingResponse.metadata.name
              }),
            ...(storageCopyMethod !== 'StorageAcceleratedCopy' &&
              storageMappingResponse?.metadata?.name && {
                storageMapping: storageMappingResponse.metadata.name
              })
          }
        })
      }

      // 4. Create rolling migration plan with the template
      const migrationPlanJson = createRollingMigrationPlanJson({
        clusterName,
        vms: selectedVMsData,
        clusterMapping,
        bmConfigRef: {
          name: selectedMaasConfig?.metadata.name || ''
        },
        migrationStrategy: {
          adminInitiatedCutOver:
            selectedMigrationOptions.cutoverOption &&
            params.cutoverOption === CUTOVER_TYPES.ADMIN_INITIATED,
          healthCheckPort: '443',
          performHealthChecks: false,
          type: selectedMigrationOptions.dataCopyMethod
            ? (params.dataCopyMethod as string)
            : 'cold',
          ...(selectedMigrationOptions.dataCopyStartTime &&
            params.dataCopyStartTime && {
              dataCopyStart: params.dataCopyStartTime
            }),
          ...(selectedMigrationOptions.cutoverOption &&
            params.cutoverOption === CUTOVER_TYPES.TIME_WINDOW &&
            params.cutoverStartTime && {
              vmCutoverStart: params.cutoverStartTime
            }),
          ...(selectedMigrationOptions.cutoverOption &&
            params.cutoverOption === CUTOVER_TYPES.TIME_WINDOW &&
            params.cutoverEndTime && {
              vmCutoverEnd: params.cutoverEndTime
            })
        },
        migrationTemplate: migrationTemplateResponse.metadata.name,
        namespace: VJAILBREAK_DEFAULT_NAMESPACE
      })

      await postRollingMigrationPlan(migrationPlanJson, VJAILBREAK_DEFAULT_NAMESPACE)

      console.log('Submitted rolling migration plan', migrationPlanJson)

      // Track successful cluster conversion creation
      track(AMPLITUDE_EVENTS.ROLLING_MIGRATION_CREATED, {
        clusterMigrationName: clusterName,
        sourceCluster: clusterObj?.name,
        destinationCluster: selectedPCD?.name,
        vmwareCredential: selectedVMwareCredName,
        pcdCredential: selectedPcdCredName,
        maasConfig: selectedMaasConfig?.metadata.name,
        virtualMachineCount: selectedVMsData?.length || 0,
        esxHostCount: orderedESXHosts?.length || 0,
        networkMappingCount: networkMappings?.length || 0,
        storageMappingCount:
          storageCopyMethod === 'StorageAcceleratedCopy'
            ? arrayCredsMappings?.length || 0
            : storageMappings?.length || 0,
        migrationType: params.dataCopyMethod || 'cold',
        hasAdminInitiatedCutover:
          selectedMigrationOptions.cutoverOption &&
          params.cutoverOption === CUTOVER_TYPES.ADMIN_INITIATED,
        hasTimedCutover:
          selectedMigrationOptions.cutoverOption &&
          params.cutoverOption === CUTOVER_TYPES.TIME_WINDOW,
        migrationTemplate: migrationTemplateResponse.metadata.name,
        namespace: VJAILBREAK_DEFAULT_NAMESPACE
      })

      onClose()
      navigate('/dashboard/cluster-conversions')
    } catch (error) {
      console.error('Failed to submit rolling migration plan:', error)

      // Track cluster conversion failure
      const parts = sourceCluster.split(':')
      const credName = parts[0]
      const sourceItem = sourceData.find((item) => item.credName === credName)
      const clusterObj = sourceItem?.clusters.find((cluster) => cluster.id === sourceCluster)
      const selectedPCD = pcdData.find((p) => p.id === destinationPCD)
      const selectedVMsData = vmsWithAssignments.filter((vm) => selectedVMs.includes(vm.id))

      track(AMPLITUDE_EVENTS.ROLLING_MIGRATION_SUBMISSION_FAILED, {
        clusterMigrationName: clusterObj?.name,
        sourceCluster: clusterObj?.name,
        destinationCluster: selectedPCD?.name,
        vmwareCredential: selectedVMwareCredName,
        pcdCredential: selectedPcdCredName,
        virtualMachineCount: selectedVMsData?.length || 0,
        esxHostCount: orderedESXHosts?.length || 0,
        errorMessage: error instanceof Error ? error.message : String(error),
        stage: 'creation'
      })

      reportError(error as Error, {
        context: 'rolling-migration-plan-submission',
        metadata: {
          sourceCluster: sourceCluster,
          destinationPCD: destinationPCD,
          selectedVMwareCredName: selectedVMwareCredName,
          selectedPcdCredName: selectedPcdCredName,
          action: 'rolling-migration-plan-submission'
        }
      })
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      alert(`Failed to submit rolling migration plan: ${errorMessage}`)
    } finally {
      setSubmitting(false)
    }
  }

  const handleClose = () => {
    if (!submitting) {
      onClose()
    }
  }

  const isSubmitDisabled = useMemo(() => {
    const basicRequirementsMissing =
      !sourceCluster || !destinationPCD || !selectedMaasConfig || !selectedVMs.length || submitting

    const storageMappingComplete =
      params.storageCopyMethod === 'StorageAcceleratedCopy'
        ? availableVmwareDatastores.every((d) => arrayCredsMappings.some((m) => m.source === d))
        : availableVmwareDatastores.every((d) => storageMappings.some((m) => m.source === d))

    const mappingsValid = !(
      availableVmwareNetworks.some(
        (network) => !networkMappings.some((mapping) => mapping.source === network)
      ) || !storageMappingComplete
    )

    // Migration options validation
    const migrationOptionValidated = Object.keys(selectedMigrationOptions).every((key) => {
      if (selectedMigrationOptions[key]) {
        if (key === 'cutoverOption' && params.cutoverOption === CUTOVER_TYPES.TIME_WINDOW) {
          return (
            params.cutoverStartTime &&
            params.cutoverEndTime &&
            !fieldErrors['cutoverStartTime'] &&
            !fieldErrors['cutoverEndTime']
          )
        }
        return params?.[key] && !fieldErrors[key]
      }
      return true
    })

    // PCD host config validation - not needed anymore since validation is handled by esxHostConfigValid

    // ESX host config validation - ensure all ESX hosts have host configs assigned
    const esxHostConfigValid = !esxHostConfigValidation.hasError

    // IP validation - ensure all selected VMs have IP addresses assigned
    const ipValidationPassed = !vmIpValidation.hasError

    // OS validation - ensure all selected powered-off VMs have OS assigned
    const osValidationPassed = !osValidation.hasError

    return (
      basicRequirementsMissing ||
      !mappingsValid ||
      !migrationOptionValidated ||
      !esxHostConfigValid ||
      !ipValidationPassed ||
      !osValidationPassed
    )
  }, [
    sourceCluster,
    destinationPCD,
    selectedMaasConfig,
    submitting,
    selectedVMs,
    availableVmwareNetworks,
    networkMappings,
    availableVmwareDatastores,
    storageMappings,
    arrayCredsMappings,
    selectedMigrationOptions,
    params,
    fieldErrors,
    orderedESXHosts,
    vmIpValidation.hasError,
    esxHostConfigValidation.hasError,
    osValidation.hasError
  ])

  useKeyboardSubmit({
    open,
    isSubmitDisabled: isSubmitDisabled,
    onSubmit: handleSubmit,
    onClose: handleClose
  })

  const theme = useTheme()
  const isSmallNav = useMediaQuery(theme.breakpoints.down('md'))
  const [activeSectionId, setActiveSectionId] = useState<string>('source-destination')

  const [touchedSections, setTouchedSections] = useState({
    sourceDestination: false,
    baremetal: false,
    hosts: false,
    vms: false,
    mapResources: false,
    options: false
  })

  const markTouched = useCallback(
    (key: keyof typeof touchedSections) => {
      setTouchedSections((prev) => (prev[key] ? prev : { ...prev, [key]: true }))
    },
    [setTouchedSections]
  )

  useEffect(() => {
    if (!open) return
    setTouchedSections({
      sourceDestination: false,
      baremetal: false,
      hosts: false,
      vms: false,
      mapResources: false,
      options: false
    })
  }, [open])

  const contentRootRef = React.useRef<HTMLDivElement | null>(null)
  const section1Ref = React.useRef<HTMLDivElement | null>(null)
  const section2Ref = React.useRef<HTMLDivElement | null>(null)
  const section3Ref = React.useRef<HTMLDivElement | null>(null)
  const section4Ref = React.useRef<HTMLDivElement | null>(null)
  const section5Ref = React.useRef<HTMLDivElement | null>(null)
  const section6Ref = React.useRef<HTMLDivElement | null>(null)
  const section7Ref = React.useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (!open) return

    let cancelled = false
    let observer: IntersectionObserver | undefined
    let rafId: number | undefined

    const init = () => {
      if (cancelled) {
        if (rafId) cancelAnimationFrame(rafId)
        return
      }

      const root = contentRootRef.current?.parentElement ?? undefined
      const nodes = [
        section1Ref.current,
        section2Ref.current,
        section3Ref.current,
        section4Ref.current,
        section5Ref.current,
        section6Ref.current
      ].filter(Boolean) as HTMLDivElement[]

      if (!root || nodes.length === 0) {
        rafId = requestAnimationFrame(init)
        return
      }

      const idByNode = new Map<Element, string>([
        [section1Ref.current as HTMLDivElement, 'source-destination'],
        [section2Ref.current as HTMLDivElement, 'baremetal'],
        [section3Ref.current as HTMLDivElement, 'hosts'],
        [section4Ref.current as HTMLDivElement, 'vms'],
        [section5Ref.current as HTMLDivElement, 'map-resources'],
        [section6Ref.current as HTMLDivElement, 'options']
      ])

      observer = new IntersectionObserver(
        (entries) => {
          const visible = entries
            .filter((e) => e.isIntersecting)
            .sort((a, b) => (b.intersectionRatio ?? 0) - (a.intersectionRatio ?? 0))[0]

          if (!visible) return
          const id = idByNode.get(visible.target)
          if (id) setActiveSectionId(id)
        },
        {
          root,
          threshold: [0.2, 0.35, 0.5, 0.65]
        }
      )

      nodes.forEach((n) => observer?.observe(n))
    }

    rafId = requestAnimationFrame(init)

    return () => {
      cancelled = true
      if (rafId) cancelAnimationFrame(rafId)
      if (observer) observer.disconnect()
    }
  }, [open])

  const step1HasErrors = false

  const step2HasErrors = false

  const step3HasErrors = Boolean(touchedSections.hosts && Boolean(esxHostConfigValidationError))

  const step4HasErrors = Boolean(
    touchedSections.vms && Boolean(vmIpValidationError || osValidationError)
  )

  const step5HasErrors = Boolean(
    touchedSections.mapResources && Boolean(networkMappingError || storageMappingError)
  )

  const step6HasErrors = Boolean(
    touchedSections.options &&
      ((selectedMigrationOptions.dataCopyStartTime && fieldErrors['dataCopyStartTime']) ||
        (selectedMigrationOptions.cutoverOption &&
          (fieldErrors['cutoverOption'] ||
            (params.cutoverOption === CUTOVER_TYPES.TIME_WINDOW &&
              (fieldErrors['cutoverStartTime'] || fieldErrors['cutoverEndTime'])))) ||
        (selectedMigrationOptions.postMigrationScript && fieldErrors['postMigrationScript']))
  )

  const hasAnyMigrationOptionSelected = useMemo(() => {
    const postMigrationAction = selectedMigrationOptions.postMigrationAction
    const postMigrationActionSelected = Boolean(
      postMigrationAction &&
        typeof postMigrationAction === 'object' &&
        Object.values(postMigrationAction as Record<string, unknown>).some(Boolean)
    )

    return (
      Boolean(selectedMigrationOptions.dataCopyMethod) ||
      Boolean(selectedMigrationOptions.dataCopyStartTime) ||
      Boolean(selectedMigrationOptions.cutoverOption) ||
      Boolean(selectedMigrationOptions.postMigrationScript) ||
      Boolean(selectedMigrationOptions.osFamily) ||
      postMigrationActionSelected
    )
  }, [selectedMigrationOptions])

  const areSelectedMigrationOptionsConfigured = useMemo(() => {
    if (!hasAnyMigrationOptionSelected) return false

    const dataCopyStartTimeOk =
      !selectedMigrationOptions.dataCopyStartTime ||
      (Boolean(params.dataCopyStartTime) && !fieldErrors['dataCopyStartTime'])

    const cutoverOk = !selectedMigrationOptions.cutoverOption
      ? true
      : Boolean(
          params.cutoverOption &&
            !fieldErrors['cutoverOption'] &&
            (params.cutoverOption !== CUTOVER_TYPES.TIME_WINDOW ||
              (params.cutoverStartTime &&
                params.cutoverEndTime &&
                !fieldErrors['cutoverStartTime'] &&
                !fieldErrors['cutoverEndTime']))
        )

    const postMigrationScriptOk =
      !selectedMigrationOptions.postMigrationScript ||
      (Boolean(params.postMigrationScript) && !fieldErrors['postMigrationScript'])

    const osFamilyOk = !selectedMigrationOptions.osFamily || Boolean(params.osFamily)

    const postMigrationAction = selectedMigrationOptions.postMigrationAction
    const postMigrationActionSelected = Boolean(
      postMigrationAction &&
        typeof postMigrationAction === 'object' &&
        Object.values(postMigrationAction as Record<string, unknown>).some(Boolean)
    )

    const postMigrationActionOk = !postMigrationActionSelected
      ? true
      : Boolean(
          postMigrationAction &&
            typeof postMigrationAction === 'object' &&
            (Boolean(postMigrationAction.renameVm) ||
              Boolean(postMigrationAction.moveToFolder) ||
              !postMigrationAction.suffix ||
              Boolean((params as any)?.postMigrationActionSuffix) ||
              !postMigrationAction.folderName ||
              Boolean((params as any)?.postMigrationActionFolderName))
        )

    return (
      dataCopyStartTimeOk &&
      cutoverOk &&
      postMigrationScriptOk &&
      osFamilyOk &&
      postMigrationActionOk
    )
  }, [
    hasAnyMigrationOptionSelected,
    selectedMigrationOptions,
    params.dataCopyStartTime,
    params.cutoverOption,
    params.cutoverStartTime,
    params.cutoverEndTime,
    params.postMigrationScript,
    params.osFamily,
    params,
    fieldErrors
  ])

  const sectionNavItems = useMemo<SectionNavItem[]>(
    () => [
      {
        id: 'source-destination',
        title: 'Source And Destination',
        description: 'Pick clusters',
        status:
          touchedSections.sourceDestination && sourceCluster && destinationPCD
            ? 'complete'
            : step1HasErrors
              ? 'attention'
              : 'incomplete'
      },
      {
        id: 'baremetal',
        title: 'Bare Metal Config',
        description: 'Verify configuration',
        status:
          touchedSections.baremetal && selectedMaasConfig
            ? 'complete'
            : step2HasErrors
              ? 'attention'
              : 'incomplete'
      },
      {
        id: 'hosts',
        title: 'ESXi Hosts',
        description: 'Assign host configs',
        status:
          touchedSections.hosts && orderedESXHosts.length > 0
            ? 'complete'
            : step3HasErrors
              ? 'attention'
              : 'incomplete'
      },
      {
        id: 'vms',
        title: 'Select VMs',
        description: 'Choose VMs and required fields',
        status:
          touchedSections.vms && selectedVMs.length > 0
            ? 'complete'
            : step4HasErrors
              ? 'attention'
              : 'incomplete'
      },
      {
        id: 'map-resources',
        title: 'Map Networks And Storage',
        description: 'Map VMware resources to PCD',
        status:
          touchedSections.mapResources &&
          availableVmwareNetworks.every((n) => networkMappings.some((m) => m.source === n)) &&
          (params.storageCopyMethod === 'StorageAcceleratedCopy'
            ? availableVmwareDatastores.every((d) => arrayCredsMappings.some((m) => m.source === d))
            : availableVmwareDatastores.every((d) => storageMappings.some((m) => m.source === d)))
            ? 'complete'
            : step5HasErrors
              ? 'attention'
              : 'incomplete'
      },
      {
        id: 'options',
        title: 'Migration Options',
        description: 'Scheduling and advanced behavior',
        status: step6HasErrors
          ? 'attention'
          : touchedSections.options && areSelectedMigrationOptionsConfigured && !step6HasErrors
            ? 'complete'
            : 'incomplete'
      }
    ],
    [
      sourceCluster,
      destinationPCD,
      selectedMaasConfig,
      orderedESXHosts.length,
      esxHostConfigValidation.hasError,
      selectedVMs.length,
      vmIpValidation.hasError,
      osValidation.hasError,
      networkMappingError,
      storageMappingError,
      availableVmwareNetworks,
      availableVmwareDatastores,
      networkMappings,
      storageMappings,
      params.cutoverOption,
      selectedMigrationOptions.cutoverOption,
      selectedMigrationOptions.dataCopyStartTime,
      selectedMigrationOptions.postMigrationScript,
      fieldErrors,
      step1HasErrors,
      step2HasErrors,
      step3HasErrors,
      step4HasErrors,
      step5HasErrors,
      step6HasErrors,
      hasAnyMigrationOptionSelected,
      areSelectedMigrationOptionsConfigured,
      touchedSections
    ]
  )

  const scrollToSection = useCallback((id: string) => {
    const map: Record<string, React.RefObject<HTMLDivElement | null>> = {
      'source-destination': section1Ref,
      baremetal: section2Ref,
      hosts: section3Ref,
      vms: section4Ref,
      'map-resources': section5Ref,
      options: section6Ref
    }

    const el = map[id]?.current
    if (!el) return
    el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    setActiveSectionId(id)
  }, [])

  const onOptionsChange = useCallback(
    (key: string | number) => (value: any) => {
      return (getParamsUpdater as any)(key)(value)
    },
    [getParamsUpdater]
  )

  const onOptionsSelectionChange = useCallback(
    (key: string | number) => (value: any) => {
      return (updateSelectedMigrationOptions as any)(key)(value)
    },
    [updateSelectedMigrationOptions]
  )

  const uniqueVmwareNetworks = useMemo(() => {
    return Array.from(new Set(availableVmwareNetworks))
  }, [availableVmwareNetworks])

  const unmappedNetworksCount = useMemo(() => {
    return uniqueVmwareNetworks.filter((n) => !networkMappings.some((m) => m.source === n)).length
  }, [uniqueVmwareNetworks, networkMappings])

  const unmappedStorageCount = useMemo(() => {
    return availableVmwareDatastores.filter((d) => !storageMappings.some((m) => m.source === d))
      .length
  }, [availableVmwareDatastores, storageMappings])

  const handleViewMaasConfig = () => {
    markTouched('baremetal')
    setMaasDetailsModalOpen(true)
  }

  const handleCloseMaasDetailsModal = () => {
    setMaasDetailsModalOpen(false)
  }

  const handleOpenPcdHostConfigDialog = () => {
    setPcdHostConfigDialogOpen(true)
  }

  const handleClosePcdHostConfigDialog = () => {
    setPcdHostConfigDialogOpen(false)
    setSelectedPcdHostConfig('')
  }

  const handlePcdHostConfigChange = (event) => {
    setSelectedPcdHostConfig(event.target.value)
  }

  const handleApplyPcdHostConfig = async () => {
    if (!selectedPcdHostConfig) {
      handleClosePcdHostConfigDialog()
      return
    }

    markTouched('hosts')

    setUpdatingPcdMapping(true)

    try {
      const availablePcdHostConfigs = openstackCredData?.spec?.pcdHostConfig || []
      const selectedPcdConfig = availablePcdHostConfigs.find(
        (config) => config.id === selectedPcdHostConfig
      )
      const pcdConfigName = selectedPcdConfig ? selectedPcdConfig.name : selectedPcdHostConfig

      // Update ALL ESX hosts with the selected host config
      const updatedESXHosts = orderedESXHosts.map((host) => ({
        ...host,
        pcdHostConfigName: pcdConfigName
      }))

      setOrderedESXHosts(updatedESXHosts)

      handleClosePcdHostConfigDialog()
    } catch (error) {
      console.error('Error updating PCD host config mapping:', error)
      reportError(error as Error, {
        context: 'pcd-host-config-mapping',
        metadata: {
          selectedPcdHostConfig: selectedPcdHostConfig,
          action: 'update-pcd-host-config-mapping'
        }
      })
    } finally {
      setUpdatingPcdMapping(false)
    }
  }

  // Define VM columns inside component to access state
  const vmColumns: GridColDef[] = [
    {
      field: 'name',
      headerName: 'VM Name',
      flex: 1.3,
      minWidth: 150,
      hideable: false,
      renderCell: (params) => (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Tooltip title={params.row.powerState === 'powered-on' ? 'Powered On' : 'Powered Off'}>
            <CdsIconWrapper>
              {/* eslint-disable-next-line @typescript-eslint/ban-ts-comment */}
              {/* @ts-ignore */}
              <cds-icon
                shape="vm"
                size="md"
                badge={params.row.powerState === 'powered-on' ? 'success' : 'danger'}
              ></cds-icon>
            </CdsIconWrapper>
          </Tooltip>
          <Box sx={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>{params.value}</Box>
        </Box>
      )
    },
    {
      field: 'ip',
      headerName: 'IP Address(es)',
      flex: 1,
      hideable: true,
      renderCell: (params) => {
        const vm = params.row as VM
        const vmId = vm.id
        const isSelected = selectedVMs.includes(vmId)
        const powerState = vm.powerState

        // For powered-off VMs - consistent display with tooltip to use toolbar
        if (powerState === 'powered-off') {
          let ipDisplay = ''
          let tooltipMessage = ''

          if (vm.networkInterfaces && vm.networkInterfaces.length > 1) {
            // Multiple network interfaces
            ipDisplay = vm.networkInterfaces.map((nic) => nic.ipAddress || '—').join(', ')
            tooltipMessage =
              "Use 'Assign IP' button in toolbar to edit IP addresses for multiple network interfaces"
          } else {
            // Single interface
            ipDisplay = vm.ip || '—'
            tooltipMessage = "Use 'Assign IP' button in toolbar to assign IP address"
          }

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

        // For single interface or when not using multi-interface modal
        const currentIp = vm.ip || '—'

        // For powered-on VMs, show IP but indicate it's not editable
        if (powerState === 'powered-on') {
          return (
            <Tooltip title="IP assignment is only available for powered-off VMs" arrow>
              <Box
                sx={{
                  display: 'flex',
                  alignItems: 'center',
                  width: '100%',
                  height: '100%'
                }}
              >
                <Typography variant="body2">{currentIp}</Typography>
              </Box>
            </Tooltip>
          )
        }

        return (
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              width: '100%',
              height: '100%'
            }}
          >
            <Typography variant="body2">{currentIp}</Typography>
          </Box>
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
        const isSelected = selectedVMs.includes(vmId)
        const powerState = params.row?.powerState
        const detectedOsFamily = params.row?.osFamily
        const assignedOsFamily = vmOSAssignments[vmId]
        const currentOsFamily = assignedOsFamily || detectedOsFamily

        // Show dropdown for ALL powered-off VMs (allows changing selection)
        if (isSelected && powerState === 'powered-off') {
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

        // Show OS with icon for assigned/detected OS
        let displayValue = currentOsFamily || 'Unknown'
        let icon: React.ReactNode = null

        if (currentOsFamily && currentOsFamily.toLowerCase().includes('windows')) {
          displayValue = 'Windows'
          icon = <img src={WindowsIcon} alt="Windows" style={{ width: 20, height: 20 }} />
        } else if (currentOsFamily && currentOsFamily.toLowerCase().includes('linux')) {
          displayValue = 'Linux'
          icon = <img src={LinuxIcon} alt="Linux" style={{ width: 20, height: 20 }} />
        } else if (currentOsFamily && currentOsFamily !== 'Unknown') {
          displayValue = 'Other'
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
              {powerState === 'powered-off' &&
                (!currentOsFamily || currentOsFamily === 'Unknown') && (
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
      flex: 1,
      hideable: true,
      valueGetter: (value) => value || '—'
    },
    {
      field: 'cpu',
      headerName: 'CPU',
      flex: 0.3,
      hideable: true,
      valueGetter: (value) => value || '- '
    },
    {
      field: 'memory',
      headerName: 'Memory (MB)',
      flex: 0.8,
      hideable: true,
      valueGetter: (value) => value || '—'
    },
    {
      field: 'esxHost',
      headerName: 'ESX Host',
      flex: 1,
      hideable: true,
      valueGetter: (value) => value || '—'
    },
    {
      field: 'flavor',
      headerName: 'Flavor',
      flex: 1,
      hideable: true,
      renderCell: (params) => {
        const vmId = params.row.id
        const isSelected = selectedVMs.includes(vmId)
        const currentFlavor = params.value || 'auto-assign'

        if (isSelected) {
          return (
            <Box sx={{ display: 'flex', alignItems: 'center', height: '100%', width: '100%' }}>
              <Select
                size="small"
                value={(() => {
                  if (currentFlavor === 'auto-assign') return 'auto-assign'
                  const flavorByName = openstackFlavors.find((f) => f.name === currentFlavor)
                  const flavorById = openstackFlavors.find((f) => f.id === currentFlavor)
                  return flavorByName?.id || flavorById?.id || currentFlavor
                })()}
                onChange={(e) => handleIndividualFlavorChange(vmId, e.target.value)}
                displayEmpty
                sx={{
                  minWidth: 120,
                  width: '100%',
                  '& .MuiSelect-select': {
                    padding: '4px 8px',
                    fontSize: '0.875rem'
                  }
                }}
              >
                <MenuItem value="auto-assign">
                  <Typography variant="body2">Auto Assign</Typography>
                </MenuItem>
                {openstackFlavors.map((flavor) => (
                  <MenuItem key={flavor.id} value={flavor.id}>
                    <Typography variant="body2">{flavor.name}</Typography>
                  </MenuItem>
                ))}
              </Select>
            </Box>
          )
        }

        return (
          <Box
            sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}
          >
            <Typography variant="body2">{currentFlavor}</Typography>
          </Box>
        )
      }
    },
    {
      field: 'powerState',
      headerName: 'Power State',
      hideable: true,
      flex: 0.8,
      valueGetter: (value) => value || '—'
    }
  ]

  const handleCloseBulkEditDialog = () => {
    setBulkEditDialogOpen(false)
    setBulkEditIPs({})
    setBulkValidationStatus({})
    setBulkValidationMessages({})
  }

  const handleBulkIpChange = (vmId: string, interfaceIndex: number, value: string) => {
    setBulkEditIPs((prev) => ({
      ...prev,
      [vmId]: { ...prev[vmId], [interfaceIndex]: value }
    }))

    if (!value.trim()) {
      setBulkValidationStatus((prev) => ({
        ...prev,
        [vmId]: { ...prev[vmId], [interfaceIndex]: 'empty' }
      }))
      setBulkValidationMessages((prev) => ({
        ...prev,
        [vmId]: { ...prev[vmId], [interfaceIndex]: '' }
      }))
    } else if (!isValidIPAddress(value.trim())) {
      setBulkValidationStatus((prev) => ({
        ...prev,
        [vmId]: { ...prev[vmId], [interfaceIndex]: 'invalid' }
      }))
      setBulkValidationMessages((prev) => ({
        ...prev,
        [vmId]: { ...prev[vmId], [interfaceIndex]: 'Invalid IP format' }
      }))
    } else {
      setBulkValidationStatus((prev) => ({
        ...prev,
        [vmId]: { ...prev[vmId], [interfaceIndex]: 'empty' }
      }))
      setBulkValidationMessages((prev) => ({
        ...prev,
        [vmId]: { ...prev[vmId], [interfaceIndex]: '' }
      }))
    }
  }

  const handleClearAllIPs = () => {
    const clearedIPs: Record<string, Record<number, string>> = {}
    const clearedStatus: Record<
      string,
      Record<number, 'empty' | 'valid' | 'invalid' | 'validating'>
    > = {}

    Object.keys(bulkEditIPs).forEach((vmId) => {
      clearedIPs[vmId] = {}
      clearedStatus[vmId] = {}

      Object.keys(bulkEditIPs[vmId]).forEach((interfaceIndexStr) => {
        const interfaceIndex = parseInt(interfaceIndexStr)
        clearedIPs[vmId][interfaceIndex] = ''
        clearedStatus[vmId][interfaceIndex] = 'empty'
      })
    })

    setBulkEditIPs(clearedIPs)
    setBulkValidationStatus(clearedStatus)
    setBulkValidationMessages({})
  }

  const handleApplyBulkIPs = async () => {
    // Collect all IPs to apply with their VM and interface info
    const ipsToApply: Array<{ vmId: string; interfaceIndex: number; ip: string }> = []

    Object.entries(bulkEditIPs).forEach(([vmId, interfaces]) => {
      Object.entries(interfaces).forEach(([interfaceIndexStr, ip]) => {
        if (ip.trim() !== '') {
          ipsToApply.push({
            vmId,
            interfaceIndex: parseInt(interfaceIndexStr),
            ip: ip.trim()
          })
        }
      })
    })

    if (ipsToApply.length === 0) return

    setAssigningIPs(true)

    try {
      // Batch validation before applying any changes
      if (openstackCredData) {
        const ipList = ipsToApply.map((item) => item.ip)

        // Set validating status for all IPs
        setBulkValidationStatus((prev) => {
          const newStatus = { ...prev }
          ipsToApply.forEach(({ vmId, interfaceIndex }) => {
            if (!newStatus[vmId]) newStatus[vmId] = {}
            newStatus[vmId][interfaceIndex] = 'validating'
          })
          return newStatus
        })

        const validationResult = await validateOpenstackIPs({
          ip: ipList,
          accessInfo: {
            secret_name: `${openstackCredData.metadata.name}-openstack-secret`,
            secret_namespace: openstackCredData.metadata.namespace
          }
        })

        // Process validation results
        const validIPs: Array<{ vmId: string; interfaceIndex: number; ip: string }> = []
        let hasInvalidIPs = false

        ipsToApply.forEach((item, index) => {
          const isValid = validationResult.isValid[index]
          const reason = validationResult.reason[index]

          if (isValid) {
            validIPs.push(item)
            setBulkValidationStatus((prev) => ({
              ...prev,
              [item.vmId]: { ...prev[item.vmId], [item.interfaceIndex]: 'valid' }
            }))
            setBulkValidationMessages((prev) => ({
              ...prev,
              [item.vmId]: { ...prev[item.vmId], [item.interfaceIndex]: 'Valid' }
            }))
          } else {
            hasInvalidIPs = true
            setBulkValidationStatus((prev) => ({
              ...prev,
              [item.vmId]: { ...prev[item.vmId], [item.interfaceIndex]: 'invalid' }
            }))
            setBulkValidationMessages((prev) => ({
              ...prev,
              [item.vmId]: { ...prev[item.vmId], [item.interfaceIndex]: reason }
            }))
          }
        })

        // Only proceed if ALL IPs are valid
        if (hasInvalidIPs) {
          setAssigningIPs(false)
          return
        }

        // Apply the valid IPs to VMs
        const updatePromises = validIPs.map(async ({ vmId, interfaceIndex, ip }) => {
          try {
            const vm = vmsWithAssignments.find((v) => v.id === vmId)
            if (!vm) throw new Error('VM not found')

            // Update network interfaces
            if (vm.networkInterfaces && vm.networkInterfaces[interfaceIndex]) {
              const updatedInterfaces = [...vm.networkInterfaces]
              updatedInterfaces[interfaceIndex] = {
                ...updatedInterfaces[interfaceIndex],
                ipAddress: ip
              }

              await patchVMwareMachine(
                vmId,
                {
                  spec: {
                    vms: {
                      networkInterfaces: updatedInterfaces
                    }
                  }
                },
                VJAILBREAK_DEFAULT_NAMESPACE
              )
            } else {
              // Fallback for single IP assignment
              await patchVMwareMachine(
                vmId,
                {
                  spec: {
                    vms: {
                      assignedIp: ip
                    }
                  }
                },
                VJAILBREAK_DEFAULT_NAMESPACE
              )
            }

            return { success: true, vmId, interfaceIndex, ip }
          } catch (error) {
            setBulkValidationStatus((prev) => ({
              ...prev,
              [vmId]: { ...prev[vmId], [interfaceIndex]: 'invalid' }
            }))
            setBulkValidationMessages((prev) => ({
              ...prev,
              [vmId]: {
                ...prev[vmId],
                [interfaceIndex]: error instanceof Error ? error.message : 'Failed to apply IP'
              }
            }))
            return { success: false, vmId, interfaceIndex, error }
          }
        })

        const results = await Promise.all(updatePromises)

        // Check if any updates failed
        const failedUpdates = results.filter((result) => !result.success)
        if (failedUpdates.length > 0) {
          setAssigningIPs(false)
          return // Don't close modal if any updates failed
        }

        // Update local VM state
        const updatedVMs = vmsWithAssignments.map((vm) => {
          const vmUpdates = validIPs.filter((item) => item.vmId === vm.id)
          if (vmUpdates.length === 0) return vm

          const updatedVM = { ...vm }

          if (vm.networkInterfaces) {
            const updatedInterfaces = [...vm.networkInterfaces]
            vmUpdates.forEach(({ interfaceIndex, ip }) => {
              if (updatedInterfaces[interfaceIndex]) {
                updatedInterfaces[interfaceIndex] = {
                  ...updatedInterfaces[interfaceIndex],
                  ipAddress: ip
                }
              }
            })
            updatedVM.networkInterfaces = updatedInterfaces

            // Recalculate comma-separated IP string
            const allIPs = updatedInterfaces
              .map((nic) => nic.ipAddress)
              .filter((ip) => ip && ip.trim() !== '')
              .join(', ')
            updatedVM.ip = allIPs || '—'
          } else {
            // Fallback for single IP
            const firstUpdate = vmUpdates[0]
            if (firstUpdate) {
              updatedVM.ip = firstUpdate.ip
            }
          }

          return updatedVM
        })
        setVmsWithAssignments(updatedVMs)

        // Update bulk validation status
        const newBulkValidationStatus = { ...bulkValidationStatus }
        const newBulkValidationMessages = { ...bulkValidationMessages }

        validIPs.forEach(({ vmId, interfaceIndex }) => {
          if (!newBulkValidationStatus[vmId]) newBulkValidationStatus[vmId] = {}
          if (!newBulkValidationMessages[vmId]) newBulkValidationMessages[vmId] = {}

          newBulkValidationStatus[vmId][interfaceIndex] = 'valid'
          newBulkValidationMessages[vmId][interfaceIndex] = 'IP validated and applied successfully'
        })

        setBulkValidationStatus(newBulkValidationStatus)
        setBulkValidationMessages(newBulkValidationMessages)

        handleCloseBulkEditDialog()
      }
    } catch (error) {
      console.error('Error in bulk IP validation/assignment:', error)
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

  // Flavor assignment handlers
  const handleOpenFlavorDialog = () => {
    if (selectedVMs.length === 0) return
    setFlavorDialogOpen(true)
  }

  const handleCloseFlavorDialog = () => {
    setFlavorDialogOpen(false)
    setSelectedFlavor('')
  }

  const handleOpenBulkIPAssignment = () => {
    if (selectedVMs.length === 0) return

    // Initialize bulk edit IPs for selected VMs
    const initialBulkEditIPs: Record<string, Record<number, string>> = {}
    const initialValidationStatus: Record<
      string,
      Record<number, 'empty' | 'valid' | 'invalid' | 'validating'>
    > = {}

    selectedVMs.forEach((vmId) => {
      const vm = vmsWithAssignments.find((v) => v.id === vmId)
      if (vm && vm.powerState === 'powered-off') {
        initialBulkEditIPs[vm.id] = {}
        initialValidationStatus[vm.id] = {}

        if (vm.networkInterfaces && vm.networkInterfaces.length > 0) {
          // Multiple network interfaces
          vm.networkInterfaces.forEach((nic, index) => {
            initialBulkEditIPs[vm.id][index] = nic.ipAddress || ''
            initialValidationStatus[vm.id][index] = nic.ipAddress ? 'valid' : 'empty'
          })
        } else {
          // Single interface (treat as interface 0)
          initialBulkEditIPs[vm.id][0] = vm.ip && vm.ip !== '—' ? vm.ip : ''
          initialValidationStatus[vm.id][0] = vm.ip && vm.ip !== '—' ? 'valid' : 'empty'
        }
      }
    })

    setBulkEditIPs(initialBulkEditIPs)
    setBulkValidationStatus(initialValidationStatus)
    setBulkValidationMessages({})
    setBulkEditDialogOpen(true)
  }

  const handleFlavorChange = (event) => {
    setSelectedFlavor(event.target.value)
  }

  const handleIndividualFlavorChange = async (vmId: string, flavorValue: string) => {
    try {
      const isAutoAssign = flavorValue === 'auto-assign'
      const selectedFlavorObj = !isAutoAssign
        ? openstackFlavors.find((f) => f.id === flavorValue)
        : null
      const flavorName = isAutoAssign
        ? 'auto-assign'
        : selectedFlavorObj
          ? selectedFlavorObj.name
          : flavorValue

      // Update VM via API
      const payload = {
        spec: {
          targetFlavorId: isAutoAssign ? '' : flavorValue
        }
      }

      await patchVMwareMachine(vmId, payload, VJAILBREAK_DEFAULT_NAMESPACE)

      // Update local state
      const updatedVMs = vmsWithAssignments.map((vm) => {
        if (vm.id === vmId) {
          return {
            ...vm,
            flavor: flavorName,
            targetFlavorId: isAutoAssign ? '' : flavorValue
          }
        }
        return vm
      })
      setVmsWithAssignments(updatedVMs)

      console.log(`Successfully assigned flavor "${flavorName}" to VM ${vmId}`)
    } catch (error) {
      console.error(`Failed to update flavor for VM ${vmId}:`, error)
      reportError(error as Error, {
        context: 'individual-vm-flavor-update',
        metadata: {
          vmId: vmId,
          flavorValue: flavorValue,
          isAutoAssign: flavorValue === 'auto-assign',
          action: 'individual-vm-flavor-update'
        }
      })
      alert(
        `Failed to assign flavor to VM: ${error instanceof Error ? error.message : 'Unknown error'}`
      )
    }
  }

  const handleIndividualHostConfigChange = async (hostId: string, configName: string) => {
    try {
      markTouched('hosts')
      // Update the ESX host with the selected host config
      const updatedESXHosts = orderedESXHosts.map((host) => {
        if (host.id === hostId) {
          return {
            ...host,
            pcdHostConfigName: configName
          }
        }
        return host
      })

      setOrderedESXHosts(updatedESXHosts)

      console.log(`Successfully assigned host config "${configName}" to ESX host ${hostId}`)
    } catch (error) {
      console.error(`Failed to update host config for ESX host ${hostId}:`, error)
      reportError(error as Error, {
        context: 'individual-host-config-update',
        metadata: {
          hostId: hostId,
          configName: configName,
          action: 'individual-host-config-update'
        }
      })
      alert(
        `Failed to assign host config to ESX host: ${error instanceof Error ? error.message : 'Unknown error'}`
      )
    }
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

      // Update VMs via API
      const updatePromises = selectedVMs.map(async (vmId) => {
        try {
          const payload = {
            spec: {
              targetFlavorId: isAutoAssign ? '' : selectedFlavor
            }
          }

          await patchVMwareMachine(vmId as string, payload, VJAILBREAK_DEFAULT_NAMESPACE)
          return { success: true, vmId }
        } catch (error) {
          console.error(`Failed to update flavor for VM ${vmId}:`, error)
          return { success: false, vmId, error }
        }
      })

      const results = await Promise.all(updatePromises)
      const failedUpdates = results.filter((result) => !result.success)

      if (failedUpdates.length > 0) {
        console.error(`Failed to update flavor for ${failedUpdates.length} VMs`)
        reportError(new Error(`Failed to update flavor for ${failedUpdates.length} VMs`), {
          context: 'vm-flavor-batch-update-failures',
          metadata: {
            failedUpdates: failedUpdates,
            totalVMs: selectedVMs.length,
            successCount: results.length - failedUpdates.length,
            failedCount: failedUpdates.length,
            action: 'vm-flavor-batch-update'
          }
        })
        alert(
          `Failed to assign flavor to ${failedUpdates.length} VM${failedUpdates.length > 1 ? 's' : ''}`
        )
      } else {
        // Update local state only if all API calls succeeded
        const updatedVMs = vmsWithAssignments.map((vm) => {
          if (selectedVMs.includes(vm.id)) {
            return {
              ...vm,
              flavor: flavorName,
              targetFlavorId: isAutoAssign ? '' : selectedFlavor
            }
          }
          return vm
        })
        setVmsWithAssignments(updatedVMs)

        const actionText = isAutoAssign ? 'cleared flavor assignment for' : 'assigned flavor to'
        console.log(
          `Successfully ${actionText} ${selectedVMs.length} VM${selectedVMs.length > 1 ? 's' : ''}`
        )

        // Refresh VM list to get updated flavor information from API
        await fetchClusterVMs()
      }

      handleCloseFlavorDialog()
    } catch (error) {
      console.error('Error updating flavors:', error)
      reportError(error as Error, {
        context: 'vm-flavor-assignment',
        metadata: {
          selectedVMs: selectedVMs,
          selectedFlavor: selectedFlavor,
          action: 'vm-flavor-assignment'
        }
      })
      alert('Failed to assign flavor to VMs')
    } finally {
      setUpdating(false)
    }
  }

  return (
    <>
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
      <FormProvider {...rhfForm}>
        <DrawerShell
          data-testid="rolling-migration-form-drawer"
          open={open}
          onClose={handleClose}
          width={drawerWidth}
          ModalProps={{
            keepMounted: false,
            style: { zIndex: 1300 }
          }}
          header={
            <DrawerHeader
              data-testid="rolling-migration-form-header"
              icon={<ClusterIcon />}
              title="Rolling Cluster Conversion"
            />
          }
          footer={
            <DrawerFooter data-testid="rolling-migration-form-footer">
              <ActionButton
                tone="secondary"
                onClick={handleClose}
                disabled={submitting}
                data-testid="rolling-migration-form-cancel"
              >
                Cancel
              </ActionButton>
              <ActionButton
                tone="primary"
                onClick={handleSubmit}
                disabled={isSubmitDisabled}
                loading={submitting}
                data-testid="rolling-migration-form-submit"
              >
                Start Conversion
              </ActionButton>
            </DrawerFooter>
          }
        >
          <Box
            ref={contentRootRef}
            data-testid="rolling-migration-form-content"
            sx={{
              display: 'grid',
              gridTemplateColumns: isSmallNav ? '1fr' : '56px 1fr',
              gap: 3
            }}
          >
            {!isSmallNav ? (
              <SectionNav
                data-testid="rolling-migration-form-section-nav"
                items={sectionNavItems}
                activeId={activeSectionId}
                onSelect={scrollToSection}
                dense
                showDescriptions={false}
              />
            ) : null}

            <Box sx={{ display: 'grid', gap: 3 }}>
              {isSmallNav ? (
                <SurfaceCard
                  title="Steps"
                  subtitle="Jump to any section"
                  data-testid="rolling-migration-form-steps-card"
                >
                  <Select
                    size="small"
                    value={activeSectionId}
                    onChange={(e) => scrollToSection(e.target.value as string)}
                    fullWidth
                    data-testid="rolling-migration-form-steps-select"
                  >
                    {sectionNavItems.map((item) => (
                      <MenuItem key={item.id} value={item.id}>
                        {item.title}
                      </MenuItem>
                    ))}
                  </Select>
                </SurfaceCard>
              ) : null}

              <Box ref={section1Ref} data-testid="rolling-migration-form-step-source-destination">
                <SurfaceCard
                  variant="section"
                  title="Source And Destination"
                  subtitle="Choose where you convert from and where you convert to"
                  data-testid="rolling-migration-form-step1-card"
                >
                  <SourceDestinationClusterSelection
                    onChange={() => () => {}}
                    errors={{}}
                    onVmwareClusterChange={handleSourceClusterChange}
                    onPcdClusterChange={handleDestinationPCDChange}
                    vmwareCluster={sourceCluster}
                    pcdCluster={destinationPCD}
                    loadingVMware={loading}
                    loadingPCD={loadingPCD}
                    showHeader={false}
                  />
                </SurfaceCard>
              </Box>

              <Divider />

              <Box ref={section2Ref} data-testid="rolling-migration-form-step-baremetal">
                <SurfaceCard
                  variant="section"
                  title="Bare Metal Config"
                  subtitle="Verify the selected configuration"
                  data-testid="rolling-migration-form-step2-card"
                >
                  {loadingMaasConfig ? (
                    <Typography variant="body2">Loading Bare Metal Config...</Typography>
                  ) : maasConfigs.length === 0 ? (
                    <Typography variant="body2">No Bare Metal Config available</Typography>
                  ) : (
                    <Typography
                      variant="subtitle2"
                      component="a"
                      data-testid="rolling-migration-form-baremetal-view-details"
                      sx={{
                        color: 'primary.main',
                        textDecoration: 'underline',
                        cursor: 'pointer',
                        fontWeight: '500'
                      }}
                      onClick={handleViewMaasConfig}
                    >
                      View Bare Metal Config Details
                    </Typography>
                  )}
                </SurfaceCard>
              </Box>

              <Divider />

              <Box ref={section3Ref} data-testid="rolling-migration-form-step-hosts">
                <SurfaceCard
                  variant="section"
                  title="ESXi Hosts"
                  subtitle="Assign PCD host configurations to all ESXi hosts"
                  data-testid="rolling-migration-form-step3-card"
                >
                  <Box
                    sx={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      mb: 1
                    }}
                  >
                    <Typography variant="body2" color="text.secondary">
                      Select ESXi hosts and assign PCD host configurations
                    </Typography>
                    {esxHostMappingStatus.fullyMapped && esxHostMappingStatus.total > 0 ? (
                      <Typography variant="body2" color="success.main">
                        All hosts mapped ✓
                      </Typography>
                    ) : (
                      <Typography variant="body2" color="warning.main">
                        {esxHostMappingStatus.mapped} of {esxHostMappingStatus.total} hosts unmapped
                      </Typography>
                    )}
                  </Box>
                  <Paper
                    sx={{ width: '100%', height: 389 }}
                    data-testid="rolling-migration-form-hosts-grid"
                  >
                    <DataGrid
                      rows={orderedESXHosts}
                      columns={esxColumns}
                      initialState={{
                        pagination: { paginationModel },
                        columns: {
                          columnVisibilityModel: {}
                        }
                      }}
                      pageSizeOptions={[5, 10, 25]}
                      rowHeight={45}
                      slots={{
                        toolbar: (props) => (
                          <CustomESXToolbarWithActions
                            {...props}
                            onAssignHostConfig={handleOpenPcdHostConfigDialog}
                          />
                        )
                      }}
                      disableColumnMenu
                      disableColumnFilter
                      loading={loadingHosts}
                    />
                  </Paper>
                  {esxHostConfigValidationError && (
                    <Alert severity="warning" sx={{ mt: 2 }}>
                      {esxHostConfigValidationError}
                    </Alert>
                  )}
                </SurfaceCard>
              </Box>

              <Divider />

              <Box ref={section4Ref} data-testid="rolling-migration-form-step-vms">
                <SurfaceCard
                  variant="section"
                  title="Select VMs"
                  subtitle="Choose the virtual machines to convert and assign required fields"
                  data-testid="rolling-migration-form-step4-card"
                >
                  <Typography
                    variant="caption"
                    color="text.secondary"
                    sx={{ mt: 1, display: 'block', mb: 1 }}
                  >
                    Tip: Powered-off VMs require IP Address and OS assignment for proper migration
                    configuration
                  </Typography>
                  <Paper
                    sx={{ width: '100%', height: 389 }}
                    data-testid="rolling-migration-form-vms-grid"
                  >
                    <DataGrid
                      rows={vmsWithAssignments}
                      columns={vmColumns}
                      initialState={{
                        pagination: { paginationModel },
                        columns: {
                          columnVisibilityModel: {}
                        }
                      }}
                      pageSizeOptions={[5, 10, 25]}
                      rowHeight={52}
                      checkboxSelection
                      onRowSelectionModelChange={(selectedRowIds) => {
                        markTouched('vms')
                        setSelectedVMs(selectedRowIds)
                      }}
                      rowSelectionModel={selectedVMs.filter((vmId) =>
                        vmsWithAssignments.some((vm) => vm.id === vmId)
                      )}
                      slots={{
                        toolbar: (props) => (
                          <CustomToolbarWithActions
                            {...props}
                            rowSelectionModel={selectedVMs.filter((vmId) =>
                              vmsWithAssignments.some((vm) => vm.id === vmId)
                            )}
                            onAssignFlavor={handleOpenFlavorDialog}
                            onAssignIP={handleOpenBulkIPAssignment}
                            hasPoweredOffVMs={selectedVMs.some((vmId) => {
                              const vm = vmsWithAssignments.find((v) => v.id === vmId)
                              return vm && vm.powerState === 'powered-off'
                            })}
                          />
                        ),
                        noRowsOverlay: () => (
                          <Box
                            sx={{
                              display: 'flex',
                              flexDirection: 'column',
                              alignItems: 'center',
                              justifyContent: 'center',
                              height: '100%',
                              p: 2
                            }}
                          >
                            <Typography
                              variant="body1"
                              color="text.secondary"
                              align="center"
                              sx={{ mb: 1 }}
                            >
                              {loadingVMs
                                ? 'Loading VMs...'
                                : 'No VMs found for the selected cluster.'}
                            </Typography>
                          </Box>
                        )
                      }}
                      disableColumnFilter
                      disableRowSelectionOnClick
                      loading={loadingVMs}
                    />
                  </Paper>
                  {vmIpValidationError && (
                    <Alert severity="warning" sx={{ mt: 2 }}>
                      {vmIpValidationError}
                    </Alert>
                  )}
                  {osValidationError && (
                    <Alert severity="warning" sx={{ mt: 2 }}>
                      {osValidationError}
                    </Alert>
                  )}
                </SurfaceCard>
              </Box>

              <Divider />

              <Box ref={section5Ref} data-testid="rolling-migration-form-step-map-resources">
                <SurfaceCard
                  variant="section"
                  title="Map Networks And Storage"
                  subtitle="Ensure VMware networks/datastores have PCD targets"
                  data-testid="rolling-migration-form-step5-card"
                >
                  {sourceCluster && destinationPCD ? (
                    <NetworkAndStorageMappingStep
                      vmwareNetworks={availableVmwareNetworks}
                      vmWareStorage={availableVmwareDatastores}
                      openstackNetworks={openstackNetworks}
                      openstackStorage={openstackVolumeTypes}
                      params={{
                        networkMappings: networkMappings,
                        storageMappings: storageMappings,
                        arrayCredsMappings: arrayCredsMappings,
                        storageCopyMethod: params.storageCopyMethod as any
                      }}
                      onChange={handleMappingsChange}
                      networkMappingError={networkMappingError}
                      storageMappingError={storageMappingError}
                      loading={loadingOpenstackDetails}
                      showHeader={false}
                    />
                  ) : (
                    <Typography variant="body2" color="text.secondary">
                      Please select both source cluster and destination PCD to configure mappings.
                    </Typography>
                  )}
                </SurfaceCard>
              </Box>

              <Divider />

              <Box
                ref={section6Ref}
                data-testid="rolling-migration-form-step-options"
                onChangeCapture={() => markTouched('options')}
                onInputCapture={() => markTouched('options')}
              >
                <SurfaceCard
                  variant="section"
                  title="Migration Options"
                  subtitle="Optional scheduling, cutover behavior, and advanced settings"
                  data-testid="rolling-migration-form-step6-card"
                >
                  <MigrationOptions
                    stepNumber="6"
                    params={params}
                    onChange={onOptionsChange}
                    selectedMigrationOptions={selectedMigrationOptions}
                    updateSelectedMigrationOptions={onOptionsSelectionChange}
                    errors={fieldErrors}
                    getErrorsUpdater={getFieldErrorsUpdater}
                    showHeader={false}
                  />
                </SurfaceCard>
              </Box>

              <Divider />

              <Box ref={section7Ref} data-testid="rolling-migration-form-step-preview">
                <SurfaceCard
                  variant="section"
                  title="Preview"
                  subtitle="Verify your selections before starting the conversion"
                  data-testid="rolling-migration-form-step7-card"
                >
                  <Box sx={{ display: 'grid', gap: 1.5 }}>
                    <Typography variant="subtitle2">Summary</Typography>
                    <Box sx={{ display: 'grid', gap: 1 }}>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 2 }}>
                        <Typography variant="body2" color="text.secondary">
                          Source
                        </Typography>
                        <Typography variant="body2">{sourceCluster || '—'}</Typography>
                      </Box>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 2 }}>
                        <Typography variant="body2" color="text.secondary">
                          Destination
                        </Typography>
                        <Typography variant="body2">{destinationPCD || '—'}</Typography>
                      </Box>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 2 }}>
                        <Typography variant="body2" color="text.secondary">
                          VMs selected
                        </Typography>
                        <Typography variant="body2">{selectedVMs.length}</Typography>
                      </Box>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 2 }}>
                        <Typography variant="body2" color="text.secondary">
                          Network mappings
                        </Typography>
                        <Typography variant="body2">
                          {uniqueVmwareNetworks.length === 0
                            ? '—'
                            : unmappedNetworksCount === 0
                              ? 'All mapped'
                              : `${unmappedNetworksCount} unmapped`}
                        </Typography>
                      </Box>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 2 }}>
                        <Typography variant="body2" color="text.secondary">
                          Storage mappings
                        </Typography>
                        <Typography variant="body2">
                          {availableVmwareDatastores.length === 0
                            ? '—'
                            : unmappedStorageCount === 0
                              ? 'All mapped'
                              : `${unmappedStorageCount} unmapped`}
                        </Typography>
                      </Box>
                    </Box>
                  </Box>
                </SurfaceCard>
              </Box>
            </Box>
          </Box>
        </DrawerShell>
      </FormProvider>

      <MaasConfigDialog
        open={maasConfigDialogOpen}
        onClose={handleCloseMaasConfig}
        aria-labelledby="baremetal-config-dialog-title"
        data-testid="rolling-migration-form-baremetal-dialog"
      >
        <DialogTitle id="baremetal-config-dialog-title">
          <Typography variant="h6">ESXi - Bare Metal Configuration</Typography>
        </DialogTitle>
        <DialogContent dividers>
          {loadingMaasConfig ? (
            <Typography>Loading configuration details...</Typography>
          ) : !selectedMaasConfig ? (
            <Typography>No configuration available</Typography>
          ) : (
            <>
              <ConfigSection>
                <Typography variant="subtitle1" gutterBottom sx={{ fontWeight: 500 }}>
                  Provider Configuration
                </Typography>
                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 3, ml: 1 }}>
                  <ConfigField>
                    <FieldLabel>Provider Type:</FieldLabel>
                    <FieldValue>{selectedMaasConfig!.spec.providerType}</FieldValue>
                  </ConfigField>
                  <ConfigField>
                    <FieldLabel>Bare Metal Provider URL:</FieldLabel>
                    <FieldValue>{selectedMaasConfig!.spec.apiUrl}</FieldValue>
                  </ConfigField>
                  <ConfigField>
                    <FieldLabel>Insecure:</FieldLabel>
                    <FieldValue>{selectedMaasConfig!.spec.insecure ? 'Yes' : 'No'}</FieldValue>
                  </ConfigField>
                  {selectedMaasConfig!.spec.os && (
                    <ConfigField>
                      <FieldLabel>OS:</FieldLabel>
                      <FieldValue>{selectedMaasConfig!.spec.os}</FieldValue>
                    </ConfigField>
                  )}
                  <ConfigField>
                    <FieldLabel>Status:</FieldLabel>
                    <FieldValue>
                      {selectedMaasConfig!.status?.validationStatus || 'Pending validation'}
                    </FieldValue>
                  </ConfigField>
                  {selectedMaasConfig!.status?.validationMessage && (
                    <ConfigField>
                      <FieldLabel>Validation Message:</FieldLabel>
                      <FieldValue>{selectedMaasConfig!.status!.validationMessage}</FieldValue>
                    </ConfigField>
                  )}
                </Box>
              </ConfigSection>

              {selectedMaasConfig!.spec.userDataSecretRef && (
                <ConfigSection>
                  <Typography variant="subtitle1" gutterBottom sx={{ fontWeight: 500 }}>
                    Cloud-Init Configuration
                  </Typography>
                  <Typography
                    variant="caption"
                    sx={{ mb: 1, display: 'block', color: 'text.secondary' }}
                  >
                    User data is stored in a secret:{' '}
                    {selectedMaasConfig!.spec.userDataSecretRef.name}
                  </Typography>
                  <CodeEditorContainer>
                    <SyntaxHighlighter
                      language="yaml"
                      style={oneLight}
                      showLineNumbers
                      wrapLongLines
                      customStyle={{
                        margin: 0,
                        maxHeight: '100%'
                      }}
                    >
                      {`# Cloud-init configuration is stored in Kubernetes Secret: 
# ${selectedMaasConfig!.spec.userDataSecretRef.name}
# in namespace: ${selectedMaasConfig!.spec.userDataSecretRef.namespace || VJAILBREAK_DEFAULT_NAMESPACE}

# The cloud-init configuration includes:
# - package updates and installations
# - configuration files
# - commands to run on startup
# - network configuration
# - and other system setup parameters

# This will be used when provisioning ESXi hosts in the bare metal environment.`}
                    </SyntaxHighlighter>
                  </CodeEditorContainer>
                </ConfigSection>
              )}

              <ConfigSection>
                <Typography variant="subtitle1" gutterBottom sx={{ fontWeight: 500 }}>
                  Resource Information
                </Typography>
                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 3, ml: 1 }}>
                  <ConfigField>
                    <FieldLabel>Name:</FieldLabel>
                    <FieldValue>{selectedMaasConfig.metadata.name}</FieldValue>
                  </ConfigField>
                  <ConfigField>
                    <FieldLabel>Namespace:</FieldLabel>
                    <FieldValue>{selectedMaasConfig.metadata.namespace}</FieldValue>
                  </ConfigField>
                  <ConfigField>
                    <FieldLabel>Created:</FieldLabel>
                    <FieldValue>
                      {new Date(selectedMaasConfig.metadata.creationTimestamp).toLocaleString()}
                    </FieldValue>
                  </ConfigField>
                </Box>
              </ConfigSection>
            </>
          )}
        </DialogContent>
        <DialogActions sx={{ gap: 1, p: 2 }}>
          <ActionButton
            tone="primary"
            onClick={handleCloseMaasConfig}
            data-testid="rolling-migration-form-baremetal-dialog-close"
          >
            Close
          </ActionButton>
        </DialogActions>
      </MaasConfigDialog>

      {maasConfigs && maasConfigs.length > 0 && (
        <MaasConfigDetailsModal
          open={maasDetailsModalOpen}
          onClose={handleCloseMaasDetailsModal}
          config={maasConfigs[0]}
        />
      )}

      {/* PCD Host Config Assignment Dialog */}
      <Dialog
        open={pcdHostConfigDialogOpen}
        onClose={handleClosePcdHostConfigDialog}
        fullWidth
        maxWidth="sm"
        data-testid="rolling-migration-form-host-config-dialog"
      >
        <DialogTitle>Assign Host Config To All ESXi Hosts</DialogTitle>
        <DialogContent>
          <Box sx={{ my: 2 }}>
            <Typography variant="body2" gutterBottom>
              Select Host Configuration
            </Typography>
            <Select
              fullWidth
              value={selectedPcdHostConfig}
              onChange={handlePcdHostConfigChange}
              size="small"
              sx={{ mt: 1 }}
              displayEmpty
              data-testid="rolling-migration-form-host-config-select"
            >
              <MenuItem value="">
                <em>Select a host configuration</em>
              </MenuItem>
              {(openstackCredData?.spec?.pcdHostConfig || []).map((config) => (
                <MenuItem key={config.id} value={config.id}>
                  <Box sx={{ display: 'flex', flexDirection: 'column' }}>
                    <Typography variant="body1">{config.name}</Typography>
                    <Typography variant="caption" color="text.secondary">
                      Management Interface: {config.mgmtInterface}
                    </Typography>
                  </Box>
                </MenuItem>
              ))}
            </Select>
          </Box>
        </DialogContent>
        <DialogActions sx={{ gap: 1, p: 2 }}>
          <ActionButton
            tone="secondary"
            onClick={handleClosePcdHostConfigDialog}
            data-testid="rolling-migration-form-host-config-cancel"
          >
            Cancel
          </ActionButton>
          <ActionButton
            tone="primary"
            onClick={handleApplyPcdHostConfig}
            disabled={!selectedPcdHostConfig || updatingPcdMapping}
            loading={updatingPcdMapping}
            data-testid="rolling-migration-form-host-config-apply"
          >
            Apply To All Hosts
          </ActionButton>
        </DialogActions>
      </Dialog>

      {/* Bulk IP Editor Dialog */}
      <Dialog open={bulkEditDialogOpen} onClose={handleCloseBulkEditDialog} maxWidth="lg">
        <DialogTitle>
          Edit IP Addresses for {selectedVMs.length} {selectedVMs.length === 1 ? 'VM' : 'VMs'}
        </DialogTitle>
        <DialogContent>
          <Box sx={{ my: 2 }}>
            {/* Quick Actions */}
            <Box sx={{ mb: 3, display: 'flex', justifyContent: 'flex-end' }}>
              <ActionButton
                size="small"
                tone="secondary"
                onClick={handleClearAllIPs}
                data-testid="rolling-migration-form-bulk-ip-clear"
              >
                Clear All
              </ActionButton>
            </Box>

            {/* IP Editor Fields */}
            <Box sx={{ maxHeight: 400, overflowY: 'auto' }}>
              {Object.entries(bulkEditIPs).map(([vmId, interfaces]) => {
                const vm = vmsWithAssignments.find((v) => v.id === vmId)
                if (!vm) return null

                return (
                  <Box
                    key={vmId}
                    sx={{ mb: 3, p: 2, border: '1px solid #e0e0e0', borderRadius: 1 }}
                  >
                    <Typography variant="body2" sx={{ fontWeight: 500, mb: 1 }}>
                      {vm.name}
                    </Typography>

                    {Object.entries(interfaces).map(([interfaceIndexStr, ip]) => {
                      const interfaceIndex = parseInt(interfaceIndexStr)
                      const networkInterface = vm.networkInterfaces?.[interfaceIndex]
                      const status = bulkValidationStatus[vmId]?.[interfaceIndex]
                      const message = bulkValidationMessages[vmId]?.[interfaceIndex]

                      return (
                        <Box
                          key={interfaceIndex}
                          sx={{ mb: 2, display: 'flex', alignItems: 'center', gap: 2 }}
                        >
                          <Box sx={{ width: 120, flexShrink: 0 }}>
                            <Typography variant="caption" color="text.secondary">
                              {networkInterface?.network || `Interface ${interfaceIndex + 1}`}:
                            </Typography>
                            <Typography variant="caption" display="block" color="text.secondary">
                              Current: {networkInterface?.ipAddress || vm.ip || '—'}
                            </Typography>
                          </Box>
                          <Box sx={{ flex: 1, display: 'flex', alignItems: 'center', gap: 1 }}>
                            <TextField
                              value={ip}
                              onChange={(e) =>
                                handleBulkIpChange(vmId, interfaceIndex, e.target.value)
                              }
                              placeholder="Enter IP address"
                              size="small"
                              sx={{ flex: 1 }}
                              error={status === 'invalid'}
                              helperText={message}
                            />
                            <Box sx={{ width: 24, display: 'flex' }}>
                              {status === 'validating' && <CircularProgress size={20} />}
                              {status === 'valid' && (
                                <CheckCircleIcon color="success" fontSize="small" />
                              )}
                              {status === 'invalid' && <ErrorIcon color="error" fontSize="small" />}
                            </Box>
                          </Box>
                        </Box>
                      )
                    })}
                  </Box>
                )
              })}
            </Box>
          </Box>
        </DialogContent>
        <DialogActions sx={{ gap: 1, p: 2 }}>
          <ActionButton
            tone="secondary"
            onClick={handleCloseBulkEditDialog}
            data-testid="rolling-migration-form-bulk-ip-cancel"
          >
            Cancel
          </ActionButton>
          <ActionButton
            tone="primary"
            onClick={handleApplyBulkIPs}
            disabled={!hasBulkIpsToApply || assigningIPs || hasBulkIpValidationErrors}
            loading={assigningIPs}
            data-testid="rolling-migration-form-bulk-ip-apply"
          >
            Apply Changes
          </ActionButton>
        </DialogActions>
      </Dialog>

      {/* Flavor Assignment Dialog */}
      <Dialog open={flavorDialogOpen} onClose={handleCloseFlavorDialog} fullWidth maxWidth="sm">
        <DialogTitle>
          Assign Flavor to {selectedVMs.length} {selectedVMs.length === 1 ? 'VM' : 'VMs'}
        </DialogTitle>
        <DialogContent>
          <Box sx={{ my: 2 }}>
            <FormLabel>Select Flavor</FormLabel>
            <Select
              fullWidth
              value={selectedFlavor}
              onChange={handleFlavorChange}
              size="small"
              sx={{ mt: 1 }}
              displayEmpty
            >
              <MenuItem value="">
                <em>Select a flavor</em>
              </MenuItem>
              <MenuItem value="auto-assign">
                <Box sx={{ display: 'flex', flexDirection: 'column' }}>
                  <Typography variant="body1">Auto Assign</Typography>
                  <Typography variant="caption" color="text.secondary">
                    Let OpenStack automatically assign the most suitable flavor
                  </Typography>
                </Box>
              </MenuItem>
              {openstackFlavors.map((flavor) => (
                <MenuItem key={flavor.id} value={flavor.id}>
                  <Box sx={{ display: 'flex', flexDirection: 'column' }}>
                    <Typography variant="body1">{flavor.name}</Typography>
                    <Typography variant="caption" color="text.secondary">
                      {flavor.vcpus} vCPU, {flavor.ram / 1024}GB RAM, {flavor.disk}GB Storage
                    </Typography>
                  </Box>
                </MenuItem>
              ))}
            </Select>
          </Box>
        </DialogContent>
        <DialogActions sx={{ gap: 1, p: 2 }}>
          <ActionButton tone="secondary" onClick={handleCloseFlavorDialog}>
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

      {/* Toast Notification */}
      <Snackbar
        open={toastOpen}
        autoHideDuration={4000}
        onClose={handleCloseToast}
        anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
      >
        <Alert
          onClose={handleCloseToast}
          severity={toastSeverity}
          sx={{ width: '100%' }}
          variant="standard"
        >
          {toastMessage}
        </Alert>
      </Snackbar>
    </>
  )
}
