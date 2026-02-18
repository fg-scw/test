import React from 'react'
import {
  styled,
  Typography,
  Box,
  FormControl,
  Select,
  MenuItem,
  ListSubheader,
  CircularProgress,
  TextField,
  InputAdornment
} from '@mui/material'
import vmwareLogo from 'src/assets/vmware.jpeg'
import { useClusterData } from './useClusterData'

import '@cds/core/icon/register.js'
import { ClarityIcons, buildingIcon, clusterIcon, searchIcon } from '@cds/core/icon'
import { Step } from 'src/shared/components'
import { FieldLabel } from 'src/components'

ClarityIcons.addIcons(buildingIcon, clusterIcon, searchIcon)

const VMwareLogoImg = styled('img')({
  width: 24,
  height: 24,
  marginRight: 8,
  objectFit: 'contain'
})

const CdsIconWrapper = styled('div')({
  marginRight: 8,
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center'
})

const ClusterSelectionStepContainer = styled('div')(({ theme }) => ({
  display: 'grid',
  gridGap: theme.spacing(1)
}))

const SideBySideContainer = styled(Box)(({ theme }) => ({
  display: 'grid',
  gridTemplateColumns: '1fr 1fr',
  gap: theme.spacing(3)
}))

interface SourceDestinationClusterSelectionProps {
  onChange: (id: string) => (value: unknown) => void
  errors: { [fieldId: string]: string }
  vmwareCluster?: string
  pcdCluster?: string
  stepNumber?: string
  stepLabel?: string
  showHeader?: boolean
  onVmwareClusterChange?: (value: string) => void
  onPcdClusterChange?: (value: string) => void
  loadingVMware?: boolean
  loadingPCD?: boolean
}

export default function SourceDestinationClusterSelection({
  onChange,
  errors,
  vmwareCluster = '',
  pcdCluster = '',
  stepNumber = '1',
  stepLabel = 'Source and Destination Clusters',
  showHeader = true,
  onVmwareClusterChange,
  onPcdClusterChange,
  loadingVMware: externalLoadingVMware,
  loadingPCD: externalLoadingPCD
}: SourceDestinationClusterSelectionProps) {
  // Use the centralized cluster data hook
  const {
    sourceData,
    pcdData,
    loadingVMware: hookLoadingVMware,
    loadingPCD: hookLoadingPCD
  } = useClusterData()

  // State for PCD search
  const [pcdSearchTerm, setPcdSearchTerm] = React.useState('')
  // State for VMware search
  const [vmwareSearchTerm, setVmwareSearchTerm] = React.useState('')
  // State to track dropdown open state
  const [vmwareDropdownOpen, setVmwareDropdownOpen] = React.useState(false)
  const [pcdDropdownOpen, setPcdDropdownOpen] = React.useState(false)

  // Filter PCD data based on search term
  const filteredPcdData = React.useMemo(() => {
    if (!pcdSearchTerm) return pcdData
    const term = pcdSearchTerm.toLowerCase().trim()
    return pcdData.filter((pcd) => {
      const clusterName = (pcd.name || '').toLowerCase()
      const credName = (pcd.openstackCredName || '').toLowerCase()
      const tenantName = (pcd.tenantName || '').toLowerCase()
      return clusterName.includes(term) || credName.includes(term) || tenantName.includes(term)
    })
  }, [pcdData, pcdSearchTerm])

  // Use external loading states if provided, otherwise use hook loading states
  const loadingVMware =
    externalLoadingVMware !== undefined ? externalLoadingVMware : hookLoadingVMware
  const loadingPCD = externalLoadingPCD !== undefined ? externalLoadingPCD : hookLoadingPCD

  // Use alternative prop names if provided
  const currentVmwareCluster = vmwareCluster
  const currentPcdCluster = pcdCluster

  const handleVMwareClusterChange = (event) => {
    const value = event.target.value
    onChange('vmwareCluster')(value)

    if (value) {
      const parts = value.split(':')
      const credName = parts[0]

      onChange('vmwareCreds')({
        existingCredName: credName
      })
    } else {
      onChange('vmwareCreds')({})
    }
  }

  const handlePcdClusterChange = (event) => {
    const value = event.target.value
    onChange('pcdCluster')(value)

    if (value) {
      const selectedPCD = pcdData.find((p) => p.id === value)
      if (selectedPCD) {
        onChange('openstackCreds')({
          existingCredName: selectedPCD.openstackCredName
        })
      }
    } else {
      onChange('openstackCreds')({})
    }
  }

  const handleVMwareClusterChangeWrapper = (event) => {
    const value = event.target.value
    if (onVmwareClusterChange) {
      onVmwareClusterChange(value)
    } else {
      handleVMwareClusterChange(event)
    }
  }

  const handlePcdClusterChangeWrapper = (event) => {
    const value = event.target.value
    if (onPcdClusterChange) {
      onPcdClusterChange(value)
    } else {
      handlePcdClusterChange(event)
    }
  }

  return (
    <ClusterSelectionStepContainer>
      {showHeader ? <Step stepNumber={stepNumber} label={stepLabel} /> : null}
      <SideBySideContainer>
        <Box>
          <Box sx={{ mb: 1 }}>
            <FieldLabel label="VMware Source Cluster" required align="flex-start" />
          </Box>
          <FormControl fullWidth variant="outlined" size="small">
            <Select
              value={currentVmwareCluster}
              onChange={handleVMwareClusterChangeWrapper}
              onOpen={() => setVmwareDropdownOpen(true)}
              onClose={() => {
                setVmwareDropdownOpen(false)
                setVmwareSearchTerm('') // Clear search on close
              }}
              open={vmwareDropdownOpen}
              displayEmpty
              disabled={loadingVMware}
              error={!!errors['vmwareCluster']}
              renderValue={(selected) => {
                if (!selected) return <em>Select VMware Cluster</em>
                const parts = selected.split(':')
                const credName = parts[0]

                const sourceItem = sourceData.find(
                  (item) =>
                    item.credName === credName && item.clusters.some((c) => c.id === selected)
                )
                const cluster = sourceItem?.clusters.find((c) => c.id === selected)
                const vcenterName = sourceItem?.vcenterName || credName
                const datacenterDisplay = sourceItem?.datacenter || ''

                return datacenterDisplay && datacenterDisplay !== 'All Datacenters'
                  ? `${vcenterName} - ${datacenterDisplay} - ${
                      cluster?.displayName || cluster?.name || 'Unknown Cluster'
                    }`
                  : `${vcenterName} - ${cluster?.displayName || cluster?.name || 'Unknown Cluster'}`
              }}
              endAdornment={
                loadingVMware ? (
                  <CircularProgress
                    size={25}
                    sx={{ mr: 3, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
                  />
                ) : null
              }
              MenuProps={{
                PaperProps: {
                  style: {
                    maxHeight: 300
                  }
                },
                MenuListProps: {
                  autoFocus: false
                }
              }}
            >
              <Box
                sx={{ p: 1, position: 'sticky', top: 0, bgcolor: 'background.paper', zIndex: 1 }}
              >
                <TextField
                  size="small"
                  placeholder="Search clusters or datacenters"
                  fullWidth
                  value={vmwareSearchTerm}
                  onChange={(e) => {
                    e.stopPropagation()
                    setVmwareSearchTerm(e.target.value)
                  }}
                  onClick={(e) => {
                    e.stopPropagation()
                    if (!vmwareDropdownOpen) {
                      setVmwareDropdownOpen(true)
                    }
                  }}
                  onKeyDown={(e) => {
                    e.stopPropagation()
                    // Prevent backspace from closing the dropdown
                    if (e.key === 'Backspace') {
                      e.nativeEvent.stopImmediatePropagation()
                    }
                  }}
                  InputProps={{
                    startAdornment: (
                      <InputAdornment position="start">
                        {/* eslint-disable-next-line @typescript-eslint/ban-ts-comment */}
                        {/* @ts-ignore */}
                        <cds-icon shape="search" size="sm"></cds-icon>
                      </InputAdornment>
                    )
                  }}
                />
              </Box>
              <MenuItem value="" disabled>
                <em>Select VMware Cluster</em>
              </MenuItem>

              {sourceData.length === 0 ? (
                <MenuItem disabled>No clusters found</MenuItem>
              ) : (
                (() => {
                  const term = vmwareSearchTerm.trim().toLowerCase()
                  const grouped = sourceData.reduce(
                    (acc, item) => {
                      if (!acc[item.vcenterName]) {
                        acc[item.vcenterName] = {
                          credName: item.credName,
                          datacenters: {}
                        } as {
                          credName: string
                          datacenters: Record<
                            string,
                            { id: string; name: string; displayName: string }[]
                          >
                        }
                      }
                      acc[item.vcenterName].datacenters[item.datacenter] = item.clusters
                      return acc
                    },
                    {} as Record<
                      string,
                      {
                        credName: string
                        datacenters: Record<
                          string,
                          { id: string; name: string; displayName: string }[]
                        >
                      }
                    >
                  )

                  const items: JSX.Element[] = []
                  Object.entries(grouped).forEach(([vcenterName, { credName, datacenters }]) => {
                    let vcenterHasMatches = false
                    const vcenterHeader = (
                      <ListSubheader key={`vc-${vcenterName}`} sx={{ fontWeight: 700 }}>
                        <Box sx={{ display: 'flex', alignItems: 'center' }}>
                          <VMwareLogoImg src={vmwareLogo} alt="VMware" />
                          {vcenterName}
                        </Box>
                      </ListSubheader>
                    )

                    Object.entries(datacenters).forEach(([datacenterName, clusters]) => {
                      const filteredClusters = term
                        ? clusters.filter((cluster) => {
                            const clusterName = cluster.displayName?.toLowerCase() || ''
                            return (
                              clusterName.includes(term) ||
                              vcenterName.toLowerCase().includes(term) ||
                              datacenterName.toLowerCase().includes(term)
                            )
                          })
                        : clusters

                      if (filteredClusters.length > 0) {
                        if (!vcenterHasMatches) {
                          items.push(vcenterHeader)
                          vcenterHasMatches = true
                        }
                        items.push(
                          <ListSubheader
                            key={`dc-${credName}-${datacenterName}`}
                            sx={{ fontWeight: 600, pl: 4 }}
                          >
                            <Box sx={{ display: 'flex', alignItems: 'center' }}>
                              <CdsIconWrapper>
                                {/* eslint-disable-next-line @typescript-eslint/ban-ts-comment */}
                                {/* @ts-ignore */}
                                <cds-icon shape="building" size="md" solid></cds-icon>
                              </CdsIconWrapper>
                              {datacenterName}
                            </Box>
                          </ListSubheader>
                        )
                        filteredClusters.forEach((cluster) => {
                          items.push(
                            <MenuItem key={cluster.id} value={cluster.id} sx={{ pl: 7 }}>
                              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                                <CdsIconWrapper>
                                  {/* eslint-disable-next-line @typescript-eslint/ban-ts-comment */}
                                  {/* @ts-ignore */}
                                  <cds-icon shape="cluster" size="md"></cds-icon>
                                </CdsIconWrapper>
                                {cluster.displayName}
                              </Box>
                            </MenuItem>
                          )
                        })
                      }
                    })
                  })

                  if (items.length === 0) {
                    return <MenuItem disabled>No matching clusters found</MenuItem>
                  }
                  return items
                })()
              )}
            </Select>
          </FormControl>
          {errors['vmwareCluster'] && (
            <Typography variant="caption" color="error" sx={{ mt: 1, display: 'block' }}>
              {errors['vmwareCluster']}
            </Typography>
          )}
        </Box>

        <Box>
          <Box sx={{ mb: 1 }}>
            <FieldLabel label="PCD Destination Cluster" required align="flex-start" />
          </Box>
          <FormControl fullWidth variant="outlined" size="small">
            <Select
              value={currentPcdCluster}
              onChange={handlePcdClusterChangeWrapper}
              onOpen={() => setPcdDropdownOpen(true)}
              onClose={() => {
                setPcdDropdownOpen(false)
                setPcdSearchTerm('') // Clear search on close
              }}
              open={pcdDropdownOpen}
              displayEmpty
              disabled={loadingPCD}
              error={!!errors['pcdCluster']}
              renderValue={(selected) => {
                if (!selected) return <em>Select PCD Cluster</em>
                const pcd = pcdData.find((p) => p.id === selected)
                return (
                  <Box sx={{ display: 'flex', flexDirection: 'row', alignItems: 'center' }}>
                    <Typography variant="body2" noWrap>
                      {pcd?.name || selected}
                    </Typography>
                    {pcd?.tenantName && (
                      <Typography variant="caption" color="text.secondary" noWrap sx={{ ml: 1 }}>
                        | Tenant: {pcd.tenantName}
                      </Typography>
                    )}
                  </Box>
                )
              }}
              endAdornment={
                loadingPCD ? (
                  <CircularProgress
                    size={25}
                    sx={{ mr: 3, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
                  />
                ) : null
              }
              MenuProps={{
                PaperProps: {
                  style: {
                    maxHeight: 300
                  }
                },
                MenuListProps: {
                  autoFocus: false
                }
              }}
            >
              <Box
                sx={{ p: 1, position: 'sticky', top: 0, bgcolor: 'background.paper', zIndex: 1 }}
              >
                <TextField
                  size="small"
                  placeholder="Search by cluster, credential, or tenant"
                  fullWidth
                  value={pcdSearchTerm}
                  onChange={(e) => {
                    e.stopPropagation()
                    setPcdSearchTerm(e.target.value)
                  }}
                  onClick={(e) => {
                    e.stopPropagation()
                    if (!pcdDropdownOpen) {
                      setPcdDropdownOpen(true)
                    }
                  }}
                  onKeyDown={(e) => {
                    e.stopPropagation()
                    // Prevent backspace from closing the dropdown
                    if (e.key === 'Backspace') {
                      e.nativeEvent.stopImmediatePropagation()
                    }
                  }}
                  InputProps={{
                    startAdornment: (
                      <InputAdornment position="start">
                        {/* eslint-disable-next-line @typescript-eslint/ban-ts-comment */}
                        {/* @ts-ignore */}
                        <cds-icon shape="search" size="sm"></cds-icon>
                      </InputAdornment>
                    )
                  }}
                />
              </Box>
              <MenuItem value="" disabled>
                <em>Select PCD Cluster</em>
              </MenuItem>

              {pcdData.length === 0 ? (
                <MenuItem disabled>No PCD clusters found</MenuItem>
              ) : filteredPcdData.length === 0 ? (
                <MenuItem disabled>No matching clusters found</MenuItem>
              ) : (
                filteredPcdData.map((pcd) => (
                  <MenuItem key={pcd.id} value={pcd.id}>
                    <Box sx={{ display: 'flex', alignItems: 'center' }}>
                      <CdsIconWrapper>
                        {/* eslint-disable-next-line @typescript-eslint/ban-ts-comment */}
                        {/* @ts-ignore */}
                        <cds-icon shape="cluster" size="md"></cds-icon>
                      </CdsIconWrapper>
                      <Box sx={{ display: 'flex', flexDirection: 'column' }}>
                        <Typography variant="body1">{pcd.name}</Typography>
                        <Typography variant="caption" color="text.secondary">
                          Credential: {pcd.openstackCredName} | Tenant: {pcd.tenantName}
                        </Typography>
                      </Box>
                    </Box>
                  </MenuItem>
                ))
              )}
            </Select>
          </FormControl>
          {errors['pcdCluster'] && (
            <Typography variant="caption" color="error" sx={{ mt: 1, display: 'block' }}>
              {errors['pcdCluster']}
            </Typography>
          )}
        </Box>
      </SideBySideContainer>
    </ClusterSelectionStepContainer>
  )
}
