import { Box, FormControl, MenuItem, Select, Checkbox, ListItemText } from '@mui/material'
import { useState, useCallback, useEffect, useMemo } from 'react'
import { useForm, SubmitHandler } from 'react-hook-form'
import {
  DrawerShell,
  DrawerHeader,
  DrawerFooter,
  ActionButton,
  OperationStatus,
  FormGrid,
  SurfaceCard,
  FieldLabel
} from 'src/components'

import { DesignSystemForm, RHFTextField, RHFSelect } from 'src/shared/components/forms'
import { OpenstackCredentialsForm } from 'src/features/credentials/components'

import { getOpenstackCredentials } from 'src/api/openstack-creds/openstackCreds'
import { createNodes } from 'src/api/nodes/nodeMappings'
import axios from 'axios'
import { OpenstackCreds, OpenstackFlavor } from 'src/api/openstack-creds/model'
import { NodeItem } from 'src/api/nodes/model'
import { useOpenstackCredentialsQuery } from 'src/hooks/api/useOpenstackCredentialsQuery'
import { useErrorHandler } from 'src/hooks/useErrorHandler'
import { useAmplitude } from 'src/hooks/useAmplitude'

interface ScaleUpDrawerProps {
  open: boolean
  onClose: () => void
  masterNode: NodeItem | null
}

interface ScaleUpFormValues {
  openstackCredential: string
  flavor: string
  nodeCount: number
  masterAgentImage?: string
}

export default function ScaleUpDrawer({ open, onClose, masterNode }: ScaleUpDrawerProps) {
  const { reportError } = useErrorHandler({ component: 'ScaleUpDrawer' })
  const { track } = useAmplitude({ component: 'ScaleUpDrawer' })

  const form = useForm<ScaleUpFormValues>({
    defaultValues: {
      openstackCredential: '',
      flavor: '',
      nodeCount: 1,
      masterAgentImage: masterNode?.spec.openstackImageID || 'No image found on master node'
    }
  })

  const {
    watch,
    setValue,
    reset,
    formState: { errors }
  } = form

  const [openstackCredentials, setOpenstackCredentials] = useState<OpenstackCreds | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [openstackError, setOpenstackError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  const [flavors, setFlavors] = useState<Array<OpenstackFlavor>>([])
  const [loadingFlavors, setLoadingFlavors] = useState(false)
  const [flavorsError, setFlavorsError] = useState<string | null>(null)

  const watchedValues = watch()
  const selectedOpenstackCred = watchedValues.openstackCredential

  const { data: openstackCredsList = [], isLoading: loadingOpenstackCreds } =
    useOpenstackCredentialsQuery()

  const openstackCredsValidated =
    openstackCredentials?.status?.openstackValidationStatus === 'Succeeded'

  const [volumeTypes, setVolumeTypes] = useState<Array<string>>([])
  const [selectedVolumeType, setSelectedVolumeType] = useState('')

  const [securityGroups, setSecurityGroups] = useState<Array<{ name: string; id: string }>>([])
  const [selectedSecurityGroups, setSelectedSecurityGroups] = useState<string[]>([])
  const [useMasterSecurityGroups, setUseMasterSecurityGroups] = useState(true)

  const flavorOptions = useMemo(() => {
    return flavors
      .filter((flavor) => flavor.disk >= 60)
      .map((flavor) => ({
        label: `${flavor.name} — ${flavor.vcpus} vCPU · ${flavor.ram / 1024}GB RAM · ${flavor.disk}GB disk`,
        value: flavor.id
      }))
  }, [flavors])

  const flavorById = useMemo(() => {
    return new Map(flavors.map((flavor) => [flavor.id, flavor]))
  }, [flavors])

  useEffect(() => {
    setValue('masterAgentImage', masterNode?.spec.openstackImageID)
  }, [masterNode, setValue])

  const clearStates = useCallback(() => {
    reset({
      openstackCredential: '',
      flavor: '',
      nodeCount: 1,
      masterAgentImage: masterNode?.spec.openstackImageID || 'No image found on master node'
    })
    setOpenstackCredentials(null)
    setOpenstackError(null)
    setError(null)
    setSuccess(false)
    setFlavors([])
    setLoadingFlavors(false)
    setFlavorsError(null)
    setVolumeTypes([])
    setSelectedVolumeType('')
    setSecurityGroups([])
    setSelectedSecurityGroups([])
    setUseMasterSecurityGroups(true)
  }, [reset])

  const handleClose = useCallback(() => {
    clearStates()
    onClose()
  }, [clearStates, onClose])

  const handleOpenstackCredSelect = useCallback(
    async (credId: string | null) => {
      setValue('openstackCredential', credId || '')
      setValue('flavor', '')
      setSelectedVolumeType('')
      setUseMasterSecurityGroups(true)
      setSelectedSecurityGroups([])

      if (credId) {
        try {
          const response = await getOpenstackCredentials(credId)
          setOpenstackCredentials(response)
          setOpenstackError(null)
        } catch (err) {
          console.error('Error fetching OpenStack credentials:', err)
          reportError(err as Error, {
            context: 'scaleup-fetch-openstack-credential',
            metadata: {
              credentialName: credId,
              action: 'get-openstack-credential'
            }
          })
          setOpenstackError(
            'Error fetching PCD credentials: ' +
              (axios.isAxiosError(err) ? err?.response?.data?.message : String(err))
          )
        }
      } else {
        setOpenstackCredentials(null)
        setOpenstackError(null)
      }
    },
    [reportError, setValue]
  )

  useEffect(() => {
    const fetchFlavours = async () => {
      if (openstackCredsValidated && openstackCredentials) {
        setLoadingFlavors(true)
        setFlavorsError(null)
        try {
          const flavours = openstackCredentials?.spec.flavors
          setFlavors(flavours || [])
        } catch (err) {
          console.error('Failed to fetch flavors:', err)
          setFlavorsError('Failed to fetch PCD flavors')
        } finally {
          setLoadingFlavors(false)
        }
      } else {
        setFlavors([])
        setValue('flavor', '')
      }
    }
    fetchFlavours()
  }, [openstackCredsValidated, openstackCredentials, setValue])

  useEffect(() => {
    if (openstackCredsValidated && openstackCredentials) {
      const types = openstackCredentials?.status?.openstack?.volumeTypes || []
      setVolumeTypes(types)
      // Set default to use master's volume type
      setSelectedVolumeType('USE_MASTER')

      const sgs = openstackCredentials?.status?.openstack?.securityGroups || []
      setSecurityGroups(sgs)
      // Default to using master's security groups
      setUseMasterSecurityGroups(true)
      setSelectedSecurityGroups([])
    } else {
      setVolumeTypes([])
      setSelectedVolumeType('')
      setSecurityGroups([])
      setSelectedSecurityGroups([])
      setUseMasterSecurityGroups(true)
    }
  }, [openstackCredsValidated, openstackCredentials])

  const handleSubmit: SubmitHandler<ScaleUpFormValues> = useCallback(
    async (values) => {
      const nodeCountNum = Number(values.nodeCount)
      if (
        !masterNode?.spec.openstackImageID ||
        !values.flavor ||
        !nodeCountNum ||
        isNaN(nodeCountNum) ||
        !openstackCredentials?.metadata?.name
      ) {
        setError('Please fill in all required fields')
        return
      }

      const selectedFlavor = flavorById.get(values.flavor)
      if (selectedFlavor && selectedFlavor.disk < 60) {
        setError('Selected flavor has insufficient disk size')
        return
      }

      try {
        setLoading(true)
        setError(null)

        track('Agents Scale Up', {
          stage: 'start',
          nodeCount: nodeCountNum,
          flavorId: values.flavor,
          credentialName: openstackCredentials?.metadata?.name
        })

        await createNodes({
          imageId: masterNode.spec.openstackImageID,
          openstackCreds: {
            kind: 'openstackcreds' as const,
            name: openstackCredentials.metadata.name,
            namespace: 'migration-system'
          },
          count: nodeCountNum,
          flavorId: values.flavor,
          volumeType: selectedVolumeType === 'USE_MASTER' ? undefined : selectedVolumeType,
          securityGroups: useMasterSecurityGroups ? undefined : selectedSecurityGroups
        })

        track('Agents Scale Up', {
          stage: 'success',
          nodeCount: nodeCountNum,
          flavorId: values.flavor,
          credentialName: openstackCredentials?.metadata?.name
        })

        setSuccess(true)
        setTimeout(() => {
          handleClose()
        }, 1500)
      } catch (err) {
        console.error('Error scaling up nodes:', err)

        track('Agents Scale Up', {
          stage: 'failure',
          nodeCount: nodeCountNum,
          flavorId: values.flavor,
          credentialName: openstackCredentials?.metadata?.name,
          errorMessage: err instanceof Error ? err.message : String(err)
        })

        reportError(err as Error, {
          context: 'scaleup-create-nodes',
          metadata: {
            nodeCount: nodeCountNum,
            flavorId: values.flavor,
            credentialName: openstackCredentials?.metadata?.name,
            action: 'create-nodes'
          }
        })

        setError(err instanceof Error ? err.message : 'Failed to scale up nodes')
      } finally {
        setLoading(false)
      }
    },
    [
      flavorById,
      handleClose,
      masterNode,
      openstackCredentials,
      reportError,
      track,
      selectedSecurityGroups,
      selectedVolumeType,
      useMasterSecurityGroups
    ]
  )

  const isSubmitDisabled =
    !masterNode ||
    !watchedValues.flavor ||
    loading ||
    !openstackCredsValidated ||
    !!errors.openstackCredential ||
    !!errors.flavor ||
    !!errors.nodeCount

  return (
    <DrawerShell
      open={open}
      onClose={handleClose}
      header={
        <DrawerHeader
          title="Scale Up Agents"
          subtitle="Create additional worker agents using a PCD credential and flavor"
          onClose={handleClose}
        />
      }
      footer={
        <DrawerFooter>
          <ActionButton tone="secondary" onClick={handleClose} data-testid="scaleup-cancel">
            Cancel
          </ActionButton>
          <ActionButton
            tone="primary"
            type="submit"
            form="scaleup-form"
            loading={loading}
            disabled={isSubmitDisabled}
            data-testid="scaleup-submit"
          >
            Scale Up
          </ActionButton>
        </DrawerFooter>
      }
    >
      <DesignSystemForm
        form={form}
        id="scaleup-form"
        onSubmit={handleSubmit}
        keyboardSubmitProps={{
          open,
          onClose: handleClose,
          isSubmitDisabled
        }}
      >
        <Box sx={{ display: 'grid', gap: 2 }} data-testid="scaleup-form">
          <SurfaceCard
            title="1. PCD Credentials"
            subtitle="Select an existing PCD credential. The credential must be validated."
          >
            <OpenstackCredentialsForm
              fullWidth
              size="small"
              credentialsList={openstackCredsList}
              loadingCredentials={loadingOpenstackCreds}
              error={openstackError ?? undefined}
              onCredentialSelect={handleOpenstackCredSelect}
              selectedCredential={selectedOpenstackCred}
              showCredentialSelector
            />
          </SurfaceCard>

          <SurfaceCard
            title="2. Agent Template"
            subtitle="Choose the flavor and security group for newly created agents."
          >
            <FormGrid minWidth={260} gap={2}>
              <RHFTextField name="masterAgentImage" label="Master Agent Image" disabled={true} />

              <RHFSelect
                name="flavor"
                label="Flavor"
                options={flavorOptions}
                placeholder={loadingFlavors ? 'Loading flavors...' : 'Select a flavor'}
                disabled={loadingFlavors || !openstackCredsValidated || !openstackCredentials}
                searchable
                searchPlaceholder="Search flavors by name, vCPU, RAM or disk"
                rules={{
                  required: 'Flavor selection is required',
                  validate: (value) => {
                    const f = flavorById.get(String(value))
                    if (f && f.disk < 60) return 'Selected flavor has insufficient disk size'
                    return true
                  }
                }}
                helperText={flavorsError ?? undefined}
                error={!!flavorsError}
              />
            </FormGrid>

            <FormGrid minWidth={260} gap={2}>
              <FormControl
                fullWidth
                size="small"
                disabled={!openstackCredsValidated || !openstackCredentials}
              >
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                  <FieldLabel label="Volume Type" align="flex-start" />
                  <FormControl
                    variant="outlined"
                    size="small"
                    disabled={!openstackCredsValidated || !openstackCredentials}
                  >
                    <Select
                      value={selectedVolumeType}
                      label=""
                      onChange={(e) => setSelectedVolumeType(e.target.value)}
                      size="small"
                    >
                      <MenuItem value="USE_MASTER">
                        Use volume type of primary VJB instance
                      </MenuItem>
                      {volumeTypes.map((volumeType) => (
                        <MenuItem key={volumeType} value={volumeType}>
                          {volumeType}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Box>
              </FormControl>

              <FormControl
                fullWidth
                size="small"
                disabled={
                  !openstackCredsValidated || !openstackCredentials || securityGroups.length === 0
                }
              >
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                  <FieldLabel label="Security Groups" align="flex-start" />

                  <Select
                    multiple
                    value={useMasterSecurityGroups ? ['USE_MASTER'] : selectedSecurityGroups}
                    label=""
                    onChange={(e) => {
                      const value = e.target.value as string[]
                      if (value.includes('USE_MASTER')) {
                        setUseMasterSecurityGroups(true)
                        setSelectedSecurityGroups([])
                      } else {
                        setUseMasterSecurityGroups(false)
                        setSelectedSecurityGroups(value)
                      }
                    }}
                    size="small"
                    renderValue={(selected) => {
                      if (useMasterSecurityGroups) {
                        return 'Use security groups of primary VJB instance'
                      }
                      return (selected as string[])
                        .map((id) => securityGroups.find((sg) => sg.id === id)?.name || id)
                        .join(', ')
                    }}
                  >
                    <MenuItem value="USE_MASTER">
                      <Checkbox checked={useMasterSecurityGroups} />
                      <ListItemText primary="Use security groups of primary VJB instance" />
                    </MenuItem>
                    {securityGroups.map((sg) => (
                      <MenuItem key={sg.id} value={sg.id} disabled={useMasterSecurityGroups}>
                        <Checkbox checked={selectedSecurityGroups.includes(sg.id)} />
                        <ListItemText primary={sg.name} />
                      </MenuItem>
                    ))}
                  </Select>
                </Box>
              </FormControl>
            </FormGrid>

            <Box sx={{ display: 'grid', gap: 1.5 }}>
              {/* <InlineHelp tone="warning" icon="warning" variant="outline">
                Select a flavor with disk &gt; 16GB for production workloads.
              </InlineHelp> */}

              <OperationStatus
                loading={loadingFlavors}
                loadingMessage="Loading available flavors…"
              />
            </Box>
          </SurfaceCard>

          <SurfaceCard
            title="3. Agent Count & Review"
            subtitle="Set the number of agents to create and review your selections before scaling up."
          >
            <FormGrid minWidth={260} gap={2}>
              <RHFTextField
                name="nodeCount"
                label="Number of Agents"
                type="number"
                rules={{
                  required: 'Number of agents is required',
                  validate: (value) => {
                    const num = Number(value)
                    if (isNaN(num)) return 'Please enter a valid number'
                    if (num < 1) return 'Minimum 1 agent required'
                    if (num > 5) return 'Maximum 5 agents allowed'
                    return true
                  }
                }}
                inputProps={{ min: 1, max: 5 }}
                fullWidth
                labelHelperText="Min: 1 · Max: 5"
                required
              />
            </FormGrid>

            <OperationStatus
              sx={{ display: 'grid', gap: 2 }}
              loading={loading}
              loadingMessage="Scaling up agents…"
              success={success}
              successMessage="Agents successfully created."
              error={error}
            />
          </SurfaceCard>
        </Box>
      </DesignSystemForm>
    </DrawerShell>
  )
}
