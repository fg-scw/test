import {
  Alert,
  Box,
  CircularProgress,
  Divider,
  FormControlLabel,
  Paper,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography
} from '@mui/material'
import LanOutlinedIcon from '@mui/icons-material/LanOutlined'
import StorageOutlinedIcon from '@mui/icons-material/StorageOutlined'
import { SyntheticEvent, useCallback, useEffect, useMemo, useState } from 'react'
import {
  ActionButton,
  DrawerFooter,
  DrawerHeader,
  DrawerShell,
  FieldLabel,
  KeyValueGrid,
  NavTab,
  NavTabs,
  Section,
  StatusChip,
  SurfaceCard
} from 'src/components'
import { formatDateTime, formatDiskSize } from 'src/utils'
import type { Migration } from 'src/features/migration/api/migrations'
import { useMigrationDetailResourcesQuery } from 'src/hooks/api/useMigrationDetailResourcesQuery'

import { isDefaultishValue, normalizeMappingRows } from './helpers'
import { MIGRATION_ENVIRONMENT_FIELDS, MIGRATION_POLICY_FIELDS } from './migrationDetailConstants'

export interface MigrationDetailModalProps {
  open: boolean
  migration: Migration | null
  onClose: () => void
}

const formatCommaSeparated = (value: unknown) => {
  if (!value) return 'N/A'
  const parts = String(value)
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
  if (!parts.length) return 'N/A'
  return parts.join(', ')
}

const enabledOrNA = (value: unknown) => {
  return value === true ? 'Enabled' : 'N/A'
}


function MappingTable({
  rows,
  sourceLabel,
  targetLabel,
  emptyLabel,
  sourceIcon,
  targetIcon
}: {
  rows: Array<{ source: string; target: string }>
  sourceLabel: string
  targetLabel: string
  emptyLabel: string
  sourceIcon?: React.ReactNode
  targetIcon?: React.ReactNode
}) {
  if (!rows.length) {
    return <Typography variant="body2">{emptyLabel}</Typography>
  }

  return (
    <TableContainer component={Paper} variant="outlined">
      <Table size="small" sx={{ tableLayout: 'fixed' }}>
        <TableHead>
          <TableRow>
            <TableCell sx={{ width: '50%' }}>{sourceLabel}</TableCell>
            <TableCell sx={{ width: '50%' }}>{targetLabel}</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {rows.map((row, idx) => (
            <TableRow key={`${row.source}-${row.target}-${idx}`}>
              <TableCell sx={{ width: '50%' }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 0 }}>
                  {sourceIcon}
                  <Typography variant="body2" sx={{ wordBreak: 'break-word' }}>
                    {row.source}
                  </Typography>
                </Box>
              </TableCell>
              <TableCell sx={{ width: '50%' }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 0 }}>
                  {targetIcon}
                  <Typography variant="body2" sx={{ wordBreak: 'break-word' }}>
                    {row.target}
                  </Typography>
                </Box>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  )
}


export default function MigrationDetailModal({
  open,
  migration,
  onClose
}: MigrationDetailModalProps) {
  const [tab, setTab] = useState(0)
  const [onlyShowOverrides, setOnlyShowOverrides] = useState(true)

  useEffect(() => {
    if (open) {
      setTab(0)
      setOnlyShowOverrides(true)
    }
  }, [open])

  const vmName = useMemo(() => ((migration?.spec as any)?.vmName as string) || '', [migration])
  const phase = migration?.status?.phase
  const phaseLabel = (phase as string) || 'Unknown'

  const { data, isLoading, error } = useMigrationDetailResourcesQuery({ open, migration })

  const migrationSpec = (migration?.spec as any) || {}
  const migrationStatus = (migration?.status as any) || {}

  const createdAt = formatDateTime(migration?.metadata?.creationTimestamp)

  const planSpec = (data?.migrationPlan?.spec as any) || {}
  const planStrategy = (planSpec?.migrationStrategy as any) || {}
  const planAdvanced = (planSpec?.advancedOptions as any) || {}
  const planPostAction = (planSpec?.postMigrationAction as any) || {}

  const assignedIpRaw =
    (migrationSpec?.assignedIP as string) ||
    (migrationSpec?.assignedIp as string) ||
    (migrationSpec?.assignedIpAddress as string) ||
    ''
  const assignedIpFromPlan = ((planSpec?.assignedIPsPerVM as Record<string, string> | undefined) || {})[vmName] || ''
  const assignedIps = formatCommaSeparated(assignedIpRaw || assignedIpFromPlan)

  const initiateCutoverEnabled = migrationSpec?.initiateCutover === true

  const templateSpec = (data?.migrationTemplate?.spec as any) || {}
  const useFlavorless = templateSpec?.useFlavorless === true
  const useGPUFlavor = templateSpec?.useGPUFlavor === true

  const vmSpec = (data?.vmwareMachine?.spec as any)?.vms || {}
  const vmMeta = (data?.vmwareMachine?.metadata as any) || {}
  const sourceDatacenter =
    (vmMeta?.annotations?.['vjailbreak.k8s.pf9.io/datacenter'] as string) ||
    (templateSpec?.source?.datacenter as string) ||
    'N/A'
  const sourceCluster =
    (vmSpec?.clusterName as string) || (vmMeta?.labels?.['vjailbreak.k8s.pf9.io/vmware-cluster'] as string) || 'N/A'
  const esxiHost =
    (vmSpec?.esxiName as string) || (vmMeta?.labels?.['vjailbreak.k8s.pf9.io/esxi-name'] as string) || 'N/A'

  const guestOS = (vmSpec?.osFamily as string) || 'N/A'
  const cpu = typeof vmSpec?.cpu === 'number' ? String(vmSpec.cpu) : 'N/A'
  const memory = typeof vmSpec?.memory === 'number' ? `${vmSpec.memory} MB` : 'N/A'

  const diskCount = useMemo(() => {
    const disks = vmSpec?.disks
    if (Array.isArray(disks)) return String(disks.length)
    return 'N/A'
  }, [vmSpec?.disks])

  const networkAdapterCount = useMemo(() => {
    const ifaces = vmSpec?.networkInterfaces
    if (Array.isArray(ifaces)) return String(ifaces.length)
    const networks = vmSpec?.networks
    if (Array.isArray(networks)) return String(networks.length)
    return 'N/A'
  }, [vmSpec?.networkInterfaces, vmSpec?.networks])

  const destinationCluster = (templateSpec?.targetPCDClusterName as string) || 'N/A'
  const destinationTenant = (data?.openstackCreds?.spec as any)?.projectName || 'N/A'

  const rawNetworkMappings = useMemo(
    () => normalizeMappingRows(((((data?.networkMapping?.spec as any)?.networks as any[]) || []) as any) || []),
    [data?.networkMapping]
  )

  const rawStorageMappings = useMemo(
    () => normalizeMappingRows(((((data?.storageMapping?.spec as any)?.storages as any[]) || []) as any) || []),
    [data?.storageMapping]
  )

  const vmSourceNetworks = useMemo(() => {
    const directNetworks = (vmSpec?.networks as string[]) || []
    const ifaceNetworks = Array.isArray(vmSpec?.networkInterfaces)
      ? (vmSpec.networkInterfaces as any[])
          .map((n) => (n?.network as string) || '')
          .map((s) => s.trim())
          .filter(Boolean)
      : []
    return Array.from(new Set([...directNetworks, ...ifaceNetworks].map((s) => String(s).trim()).filter(Boolean)))
  }, [vmSpec?.networks, vmSpec?.networkInterfaces])

  const vmSourceDatastores = useMemo(() => {
    const ds = (vmSpec?.datastores as string[]) || []
    return Array.from(new Set(ds.map((s) => String(s).trim()).filter(Boolean)))
  }, [vmSpec?.datastores])

  const networkMappings = useMemo(() => {
    if (!vmSourceNetworks.length) return rawNetworkMappings
    return rawNetworkMappings.filter((row) => vmSourceNetworks.includes(row.source))
  }, [rawNetworkMappings, vmSourceNetworks])

  const storageMappings = useMemo(() => {
    if (!vmSourceDatastores.length) return rawStorageMappings
    return rawStorageMappings.filter((row) => vmSourceDatastores.includes(row.source))
  }, [rawStorageMappings, vmSourceDatastores])

  const migrationType = (planStrategy?.type as string) || 'N/A'
  const scheduleDataCopy = planStrategy?.dataCopyStart ? formatDateTime(planStrategy?.dataCopyStart) : 'N/A'

  const periodicSyncEnabled = planAdvanced?.periodicSyncEnabled === true
  const periodicSyncInterval = (planAdvanced?.periodicSyncInterval as string) || ''

  const cutoverPolicy = useMemo(() => {
    if (!initiateCutoverEnabled) return 'N/A'

    if (planStrategy?.adminInitiatedCutOver === true) {
      const periodicSyncValue = periodicSyncEnabled
        ? periodicSyncInterval
          ? `Enabled (${periodicSyncInterval})`
          : 'Enabled'
        : 'Disabled'
      return `Admin initiated (Periodic sync: ${periodicSyncValue})`
    }

    if (planStrategy?.vmCutoverStart || planStrategy?.vmCutoverEnd) {
      const start = planStrategy?.vmCutoverStart ? formatDateTime(planStrategy?.vmCutoverStart) : 'N/A'
      const end = planStrategy?.vmCutoverEnd ? formatDateTime(planStrategy?.vmCutoverEnd) : 'N/A'
      return `Time window (${start} - ${end})`
    }

    return 'Immediately after data copy'
  }, [
    initiateCutoverEnabled,
    periodicSyncEnabled,
    periodicSyncInterval,
    planStrategy?.adminInitiatedCutOver,
    planStrategy?.vmCutoverEnd,
    planStrategy?.vmCutoverStart
  ])

  const securityGroupOptions = ((data?.openstackCreds as any)?.status?.openstack?.securityGroups as any[]) || []
  const serverGroupOptions = ((data?.openstackCreds as any)?.status?.openstack?.serverGroups as any[]) || []

  const securityGroups = useMemo(() => {
    const configured = planSpec?.securityGroups
    if (!Array.isArray(configured) || !configured.length) return 'N/A'

    const names = configured
      .map((value) => {
        const match = securityGroupOptions.find((opt) => opt?.id === value || opt?.name === value)
        return (match?.name as string) || String(value)
      })
      .filter(Boolean)

    return names.length ? names.join(', ') : 'N/A'
  }, [planSpec?.securityGroups, securityGroupOptions])

  const serverGroup = useMemo(() => {
    const configured = (planSpec?.serverGroup as string) || ''
    if (!configured) return 'N/A'
    const match = serverGroupOptions.find((opt) => opt?.id === configured || opt?.name === configured)
    return (match?.name as string) || configured
  }, [planSpec?.serverGroup, serverGroupOptions])

  const renameVmEnabled = planPostAction?.renameVm === true
  const renameSuffix = renameVmEnabled ? ((planPostAction?.suffix as string) || 'N/A') : 'N/A'

  const moveToFolderEnabled = planPostAction?.moveToFolder === true
  const folderName = moveToFolderEnabled ? ((planPostAction?.folderName as string) || 'N/A') : 'N/A'

  const disconnectSourceNetwork = enabledOrNA(planStrategy?.disconnectSourceNetwork)
  const fallbackToDhcp = enabledOrNA(planSpec?.fallbackToDHCP)
  const networkPersistence = enabledOrNA(planAdvanced?.networkPersistence)

  const postMigrationScript = useMemo(() => {
    const raw = (planSpec?.firstBootScript as string) || ''
    const trimmed = raw.trim()
    if (!trimmed) return ''
    if (trimmed === 'echo "Add your startup script here!"') return ''
    if (trimmed === 'echo "Add your startup script here!"\n') return ''
    return raw
  }, [planSpec?.firstBootScript])

  const handleTabChange = useCallback((_: SyntheticEvent, newValue: number) => {
    setTab(newValue)
  }, [])

  const rdmDisksSummary = useMemo(() => {
    if (data?.rdmDisks?.length) return `${data.rdmDisks.length} disk(s)`
    if (Array.isArray(vmSpec?.rdmDisks) && vmSpec.rdmDisks.length) return `${vmSpec.rdmDisks.length} disk(s)`
    return 'N/A'
  }, [data?.rdmDisks, vmSpec?.rdmDisks])

  const generalInfoItems = useMemo(
    () => [
      { label: 'VM Name', value: vmName || 'N/A' },
      { label: 'Migration Type', value: migrationType },
      { label: 'Assigned IP(s)', value: assignedIps },
      { label: 'Created At', value: createdAt },
      { label: 'Guest OS', value: guestOS },
      { label: 'CPU', value: cpu },
      { label: 'Memory', value: memory },
      { label: 'Total Disks', value: diskCount },
      { label: 'Network Adapters', value: networkAdapterCount },
      { label: 'vJailbreak Agent', value: (migrationStatus?.agentName as string) || 'N/A' },
      { label: 'RDM Disks', value: rdmDisksSummary }
    ],
    [
      assignedIps,
      cpu,
      createdAt,
      diskCount,
      guestOS,
      memory,
      migrationStatus?.agentName,
      migrationType,
      networkAdapterCount,
      rdmDisksSummary,
      vmName
    ]
  )

  const handleOnlyShowOverridesChange = useCallback((checked: boolean) => {
    setOnlyShowOverrides(checked)
  }, [])

  const migrationPolicyValues = useMemo(
    () => ({
      securityGroups,
      serverGroup,
      scheduleDataCopy,
      cutoverPolicy,
      renameSuffix,
      folderName,
      disconnectSourceNetwork,
      fallbackToDhcp,
      networkPersistence,
      useGPUFlavor: enabledOrNA(useGPUFlavor),
      useFlavorless: enabledOrNA(useFlavorless)
    }),
    [
      cutoverPolicy,
      disconnectSourceNetwork,
      fallbackToDhcp,
      folderName,
      networkPersistence,
      renameSuffix,
      scheduleDataCopy,
      securityGroups,
      serverGroup,
      useFlavorless,
      useGPUFlavor
    ]
  )

  const migrationPolicyItems = useMemo(
    () =>
      MIGRATION_POLICY_FIELDS.map((field) => ({
        label: field.label,
        value: (migrationPolicyValues as any)[field.key] as string
      })),
    [migrationPolicyValues]
  )

  const migrationEnvironmentValues = useMemo(
    () => ({
      sourceDatacenter,
      sourceCluster,
      esxiHost,
      destinationTenant,
      destinationCluster
    }),
    [destinationCluster, destinationTenant, esxiHost, sourceCluster, sourceDatacenter]
  )

  const migrationEnvironmentItems = useMemo(
    () =>
      MIGRATION_ENVIRONMENT_FIELDS.map((field) => ({
        label: field.label,
        value: (migrationEnvironmentValues as any)[field.key] as string
      })),
    [migrationEnvironmentValues]
  )

  const visiblePolicyItems = useMemo(() => {
    if (!onlyShowOverrides) return migrationPolicyItems
    return migrationPolicyItems.filter((item) => !isDefaultishValue(item.value))
  }, [migrationPolicyItems, onlyShowOverrides])

  return (
    <DrawerShell
      open={open}
      onClose={onClose}
      requireCloseConfirmation={false}
      width={860}
      header={
        <Box>
          <DrawerHeader
            title="Migration Details"
            subtitle={vmName || migration?.metadata?.name || ''}
            onClose={onClose}
            actions={<StatusChip label={phaseLabel} size="small" variant="filled" />}
          />
          <NavTabs value={tab} onChange={handleTabChange} sx={{ px: 2, borderBottom: (theme) => `1px solid ${theme.palette.divider}` }}>
            <NavTab label="General" value={0} />
            <NavTab label="Advanced" value={1} />
          </NavTabs>
        </Box>
      }
      footer={
        <DrawerFooter>
          <ActionButton tone="secondary" onClick={onClose}>
            Close
          </ActionButton>
        </DrawerFooter>
      }
    >
      <Box sx={{ display: 'grid', gap: 2 }} data-testid="migration-detail">
        {!migration ? (
          <Alert severity="info">No migration selected.</Alert>
        ) : isLoading ? (
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', py: 6 }}>
            <CircularProgress size={24} sx={{ mr: 2 }} />
            <Typography variant="body2" color="text.secondary">
              Loading migration details…
            </Typography>
          </Box>
        ) : error ? (
          <Alert severity="error">Failed to load migration details.</Alert>
        ) : (
          <Section>
            {tab === 0 ? (
              <Box sx={{ display: 'grid', gap: 2 }}>
                <SurfaceCard variant="card" title="Migration Environment" subtitle="Source and destination overview">
                  <KeyValueGrid items={migrationEnvironmentItems} />
                </SurfaceCard>

                <SurfaceCard variant="card" title="General Info" subtitle="VM specifications">
                  <KeyValueGrid items={generalInfoItems} />

                  {data?.rdmDisks?.length ? (
                    <Box sx={{ mt: 2 }}>
                      <Divider sx={{ mb: 2 }} />
                      <TableContainer component={Paper} variant="outlined">
                        <Table size="small">
                          <TableHead>
                            <TableRow>
                              <TableCell>Disk</TableCell>
                              <TableCell>Size</TableCell>
                              <TableCell>Phase</TableCell>
                            </TableRow>
                          </TableHead>
                          <TableBody>
                            {data.rdmDisks.map((d) => (
                              <TableRow key={d.metadata.name}>
                                <TableCell>
                                  <Typography variant="body2" sx={{ wordBreak: 'break-word' }}>
                                    {d.spec.displayName || d.spec.diskName || d.metadata.name}
                                  </Typography>
                                </TableCell>
                                <TableCell>{formatDiskSize(d.spec.diskSize)}</TableCell>
                                <TableCell>{d.status?.phase || 'N/A'}</TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      </TableContainer>
                    </Box>
                  ) : null}
                </SurfaceCard>

                <SurfaceCard variant="card" title="Mappings" subtitle="Network and storage mappings">
                  <Box sx={{ display: 'grid', gap: 2.5 }}>
                    <Box sx={{ display: 'grid', gap: 1 }}>
                      <FieldLabel label="Network Mapping" />
                      <MappingTable
                        rows={networkMappings}
                        sourceLabel="Source Network"
                        targetLabel="Target Network"
                        emptyLabel="N/A"
                        sourceIcon={<LanOutlinedIcon fontSize="small" color="action" />}
                        targetIcon={<LanOutlinedIcon fontSize="small" color="action" />}
                      />
                    </Box>

                    <Box sx={{ display: 'grid', gap: 1 }}>
                      <FieldLabel label="Storage Mapping" />
                      <MappingTable
                        rows={storageMappings}
                        sourceLabel="Source Datastore"
                        targetLabel="Target Volume Type"
                        emptyLabel="N/A"
                        sourceIcon={<StorageOutlinedIcon fontSize="small" color="action" />}
                        targetIcon={<StorageOutlinedIcon fontSize="small" color="action" />}
                      />
                    </Box>
                  </Box>
                </SurfaceCard>
              </Box>
            ) : (
              <SurfaceCard variant="card" title="Migration Policies" subtitle="Flags and post-migration actions">
                <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 1 }}>
                  <FormControlLabel
                    control={
                      <Switch
                        size="small"
                        checked={onlyShowOverrides}
                        onChange={(e) => handleOnlyShowOverridesChange(e.target.checked)}
                      />
                    }
                    label={<Typography variant="body2">View only enabled options</Typography>}
                  />
                </Box>

                {visiblePolicyItems.length ? (
                  <KeyValueGrid items={visiblePolicyItems} />
                ) : (
                  <Typography variant="body2">No advanced options configured.</Typography>
                )}

                {postMigrationScript || !onlyShowOverrides ? (
                  <Box sx={{ mt: 2, display: 'grid', gap: 1.5 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 2 }}>
                      <FieldLabel label="Post-migration script" />
                    </Box>

                    {postMigrationScript ? (
                      <Paper
                        variant="outlined"
                        sx={{
                          p: 2,
                          backgroundColor: (theme) => theme.palette.action.hover
                        }}
                      >
                        <Typography
                          variant="caption"
                          component="pre"
                          sx={{
                            m: 0,
                            whiteSpace: 'pre-wrap'
                          }}
                        >
                          {postMigrationScript}
                        </Typography>
                      </Paper>
                    ) : (
                      <Typography variant="body2">N/A</Typography>
                    )}
                  </Box>
                ) : null}
              </SurfaceCard>
            )}
          </Section>
        )}
      </Box>
    </DrawerShell>
  )
}
