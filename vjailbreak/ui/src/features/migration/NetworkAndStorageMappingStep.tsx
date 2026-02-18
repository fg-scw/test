import {
  FormControl,
  FormHelperText,
  styled,
  Typography,
  Box,
  RadioGroup,
  FormControlLabel,
  Radio,
  Alert,
  Chip
} from '@mui/material'
import { useEffect, useMemo } from 'react'
import { ResourceMappingTableNew as ResourceMappingTable } from './components'
import { Step } from 'src/shared/components/forms'
import { FieldLabel } from 'src/components'
import { useArrayCredentialsQuery } from 'src/hooks/api/useArrayCredentialsQuery'

const VmsSelectionStepContainer = styled('div')(({ theme }) => ({
  display: 'grid',
  gridGap: theme.spacing(1)
}))

const FieldsContainer = styled('div')(({ theme }) => ({
  display: 'grid',
  gridGap: theme.spacing(2)
}))

export interface ResourceMap {
  source: string
  target: string
}

// Storage copy method options
export const STORAGE_COPY_METHOD_OPTIONS = [
  { value: 'normal', label: 'Standard Copy' },
  { value: 'StorageAcceleratedCopy', label: 'Storage Accelerated Copy' }
] as const

export type StorageCopyMethod = (typeof STORAGE_COPY_METHOD_OPTIONS)[number]['value']

interface NetworkAndStorageMappingStepProps {
  vmwareNetworks: string[]
  vmWareStorage: string[]
  openstackNetworks: string[]
  openstackStorage: string[]
  params: {
    networkMappings?: ResourceMap[]
    storageMappings?: ResourceMap[]
    arrayCredsMappings?: ResourceMap[]
    storageCopyMethod?: StorageCopyMethod
  }
  onChange: (key: string) => (value: any) => void
  networkMappingError?: string
  storageMappingError?: string
  stepNumber?: string
  loading?: boolean
  showHeader?: boolean
}

export default function NetworkAndStorageMappingStep({
  vmwareNetworks = [],
  vmWareStorage = [],
  openstackNetworks = [],
  openstackStorage = [],
  params,
  onChange,
  networkMappingError,
  storageMappingError,
  stepNumber = '3',
  loading = false,
  showHeader = true
}: NetworkAndStorageMappingStepProps) {
  const storageCopyMethod = params.storageCopyMethod || 'normal'

  // Fetch validated array credentials for StorageAcceleratedCopy
  const { data: arrayCredentials, isLoading: arrayCredsLoading } = useArrayCredentialsQuery(
    undefined,
    {
      enabled: storageCopyMethod === 'StorageAcceleratedCopy'
    }
  )

  // Filter to only validated array credentials
  const validatedArrayCreds = useMemo(
    () =>
      (arrayCredentials || []).filter((cred) => cred.status?.arrayValidationStatus === 'Succeeded'),
    [arrayCredentials]
  )

  // Get available ArrayCreds names for dropdown
  const arrayCredsNames = useMemo(
    () => validatedArrayCreds.map((ac) => ac.metadata.name),
    [validatedArrayCreds]
  )

  // Filter out any mappings that don't match the available networks/storage
  const filteredNetworkMappings = useMemo(
    () =>
      (params.networkMappings || []).filter(
        (mapping) =>
          vmwareNetworks.includes(mapping.source) && openstackNetworks.includes(mapping.target)
      ),
    [params.networkMappings, vmwareNetworks, openstackNetworks]
  )

  const filteredStorageMappings = useMemo(
    () =>
      (params.storageMappings || []).filter(
        (mapping) =>
          vmWareStorage.includes(mapping.source) && openstackStorage.includes(mapping.target)
      ),
    [params.storageMappings, vmWareStorage, openstackStorage]
  )

  const filteredArrayCredsMappings = useMemo(
    () =>
      (params.arrayCredsMappings || []).filter(
        (mapping) =>
          vmWareStorage.includes(mapping.source) && arrayCredsNames.includes(mapping.target)
      ),
    [params.arrayCredsMappings, vmWareStorage, arrayCredsNames]
  )

  useEffect(() => {
    if (filteredNetworkMappings.length !== params.networkMappings?.length) {
      onChange('networkMappings')(filteredNetworkMappings)
    }
  }, [filteredNetworkMappings, onChange, params.networkMappings])

  useEffect(() => {
    if (
      storageCopyMethod === 'normal' &&
      filteredStorageMappings.length !== params.storageMappings?.length
    ) {
      onChange('storageMappings')(filteredStorageMappings)
    }
  }, [filteredStorageMappings, onChange, params.storageMappings, storageCopyMethod])

  useEffect(() => {
    if (
      storageCopyMethod === 'StorageAcceleratedCopy' &&
      filteredArrayCredsMappings.length !== params.arrayCredsMappings?.length
    ) {
      onChange('arrayCredsMappings')(filteredArrayCredsMappings)
    }
  }, [filteredArrayCredsMappings, onChange, params.arrayCredsMappings, storageCopyMethod])

  // Auto-map datastores to ArrayCreds based on dataStore information in ArrayCreds status
  useEffect(() => {
    if (
      storageCopyMethod !== 'StorageAcceleratedCopy' ||
      !validatedArrayCreds.length ||
      !vmWareStorage.length
    ) {
      return
    }

    // Create a map of datastore names to ArrayCreds names
    const datastoreToArrayCredsMap = new Map<string, string>()
    validatedArrayCreds.forEach((cred) => {
      const datastores = cred.status?.dataStore || []
      datastores.forEach((ds) => {
        // Map datastore name to ArrayCreds name
        datastoreToArrayCredsMap.set(ds.name, cred.metadata.name)
      })
    })

    // Get current mappings
    const currentMappings = params.arrayCredsMappings || []
    const currentMappedSources = new Set(currentMappings.map((m) => m.source))

    // Find datastores that can be auto-mapped but aren't already mapped
    const autoMappings: ResourceMap[] = []
    vmWareStorage.forEach((datastore) => {
      if (!currentMappedSources.has(datastore) && datastoreToArrayCredsMap.has(datastore)) {
        autoMappings.push({
          source: datastore,
          target: datastoreToArrayCredsMap.get(datastore)!
        })
      }
    })

    // If we found any auto-mappings, add them to existing mappings
    if (autoMappings.length > 0) {
      onChange('arrayCredsMappings')([...currentMappings, ...autoMappings])
    }
  }, [storageCopyMethod, validatedArrayCreds, vmWareStorage, params.arrayCredsMappings, onChange])

  // Calculate unmapped networks and storage
  const unmappedNetworks = useMemo(
    () =>
      vmwareNetworks.filter(
        (network) => !params.networkMappings?.some((mapping) => mapping.source === network)
      ),
    [vmwareNetworks, params.networkMappings]
  )

  const unmappedStorage = useMemo(() => {
    if (storageCopyMethod === 'StorageAcceleratedCopy') {
      return vmWareStorage.filter(
        (storage) => !params.arrayCredsMappings?.some((mapping) => mapping.source === storage)
      )
    }
    return vmWareStorage.filter(
      (storage) => !params.storageMappings?.some((mapping) => mapping.source === storage)
    )
  }, [vmWareStorage, params.storageMappings, params.arrayCredsMappings, storageCopyMethod])

  // Calculate completion status
  const networksFullyMapped = unmappedNetworks.length === 0 && vmwareNetworks.length > 0
  const storageFullyMapped = unmappedStorage.length === 0 && vmWareStorage.length > 0

  return (
    <VmsSelectionStepContainer>
      {showHeader ? <Step stepNumber={stepNumber} label="Network And Storage Mapping" /> : null}
      <FieldsContainer>
        {loading ? (
          <Typography variant="body2" color="text.secondary">
            Loading PCD networks and storage options...
          </Typography>
        ) : (
          <>
            <FormControl error={!!networkMappingError}>
              <Box
                sx={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  mb: 1
                }}
              >
                <FieldLabel label="Map Networks" required align="flex-start" />
                {networksFullyMapped ? (
                  <Typography variant="body2" color="success.main">
                    All networks mapped ✓
                  </Typography>
                ) : (
                  <Typography variant="body2" color="warning.main">
                    {unmappedNetworks.length} of {vmwareNetworks.length} networks unmapped
                  </Typography>
                )}
              </Box>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Select source and target networks to automatically create mappings. All networks
                must be mapped to proceed.
              </Typography>
              <ResourceMappingTable
                sourceItems={vmwareNetworks}
                targetItems={openstackNetworks}
                sourceLabel="VMware Network"
                targetLabel="PCD Network"
                values={params.networkMappings || []}
                onChange={(value) => onChange('networkMappings')(value)}
                oneToManyMapping
                fieldPrefix="networkMapping"
              />
              {networkMappingError && <FormHelperText error>{networkMappingError}</FormHelperText>}
            </FormControl>
            <FormControl error={!!storageMappingError}>
              <Box
                sx={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  mb: 1
                }}
              >
                <FieldLabel label="Map Storage" required align="flex-start" />
                {storageFullyMapped ? (
                  <Typography variant="body2" color="success.main">
                    All storage mapped ✓
                  </Typography>
                ) : (
                  <Typography variant="body2" color="warning.main">
                    {unmappedStorage.length} of {vmWareStorage.length} storage devices unmapped
                  </Typography>
                )}
              </Box>

              {/* Storage Copy Method Selection */}
              <Box sx={{ mb: 2 }}>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                  Select the data copy method for storage migration:
                </Typography>
                <RadioGroup
                  value={storageCopyMethod}
                  onChange={(e) => onChange('storageCopyMethod')(e.target.value)}
                  row
                >
                  {STORAGE_COPY_METHOD_OPTIONS.map((option) => (
                    <FormControlLabel
                      key={option.value}
                      value={option.value}
                      control={<Radio />}
                      label={
                        option.value === 'StorageAcceleratedCopy' ? (
                          <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 1 }}>
                            <Box component="span">{option.label}</Box>
                            <Chip
                              label="Beta"
                              size="small"
                              color="warning"
                              variant="outlined"
                              sx={{
                                transform: 'translateY(-6px)',
                                height: 16,
                                '& .MuiChip-label': {
                                  px: 0.75,
                                  fontSize: '0.65rem',
                                  lineHeight: '16px',
                                  display: 'flex',
                                  alignItems: 'center'
                                }
                              }}
                            />
                          </Box>
                        ) : (
                          option.label
                        )
                      }
                    />
                  ))}
                </RadioGroup>
              </Box>

              {storageCopyMethod === 'normal' ? (
                <>
                  <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                    Select source and target storage to automatically create mappings. All storage
                    devices must be mapped in order to proceed.
                  </Typography>
                  <ResourceMappingTable
                    sourceItems={vmWareStorage}
                    targetItems={openstackStorage}
                    sourceLabel="VMware Datastore"
                    targetLabel="PCD Volume Type"
                    values={params.storageMappings || []}
                    onChange={(value) => onChange('storageMappings')(value)}
                    oneToManyMapping
                    fieldPrefix="storageMapping"
                  />
                </>
              ) : (
                <>
                  <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                    Map datastores to storage array credentials for storage array data copy.
                  </Typography>

                  {arrayCredsLoading ? (
                    <Alert severity="info" sx={{ mb: 2 }}>
                      Loading array credentials...
                    </Alert>
                  ) : validatedArrayCreds.length === 0 ? (
                    <Alert severity="warning" sx={{ mb: 2 }}>
                      No validated array credentials found. Please configure and validate array
                      credentials in the Storage Management page first.
                    </Alert>
                  ) : (
                    <>
                      {unmappedStorage.length > 0 && (
                        <Alert severity="info" sx={{ mb: 2 }}>
                          {unmappedStorage.length} datastore(s) need to be mapped. Please map them
                          below.
                        </Alert>
                      )}
                      <ResourceMappingTable
                        sourceItems={vmWareStorage}
                        targetItems={arrayCredsNames}
                        sourceLabel="VMware Datastore"
                        targetLabel="Array Credentials"
                        values={params.arrayCredsMappings || []}
                        onChange={(value) => onChange('arrayCredsMappings')(value)}
                        oneToManyMapping
                        fieldPrefix="arrayCredsMapping"
                      />
                    </>
                  )}
                </>
              )}

              {storageMappingError && <FormHelperText error>{storageMappingError}</FormHelperText>}
            </FormControl>
          </>
        )}
      </FieldsContainer>
    </VmsSelectionStepContainer>
  )
}
