import React, { useCallback, useEffect, useState } from 'react'
import { Alert, Box, Collapse, IconButton, InputAdornment, Tooltip } from '@mui/material'
import { Visibility, VisibilityOff } from '@mui/icons-material'
import CheckIcon from '@mui/icons-material/Check'
import InfoOutlined from '@mui/icons-material/InfoOutlined'

import { useForm } from 'react-hook-form'

import {
  DrawerShell,
  DrawerHeader,
  DrawerFooter,
  ActionButton,
  FormGrid,
  OperationStatus,
  Section,
  SectionHeader,
  SurfaceCard
} from 'src/components'
import { DesignSystemForm, RHFTextField, RHFToggleField } from 'src/shared/components/forms'

import { createVMwareCredsWithSecretFlow, deleteVMwareCredsWithSecretFlow } from 'src/api/helpers'
import { getVmwareCredentials } from 'src/api/vmware-creds/vmwareCreds'

import axios from 'axios'

import { useVmwareCredentialsQuery } from 'src/hooks/api/useVmwareCredentialsQuery'
import { useInterval } from 'src/hooks/useInterval'
import { useErrorHandler } from 'src/hooks/useErrorHandler'
import { useAmplitude } from 'src/hooks/useAmplitude'

import { isValidName } from 'src/utils'
import { THREE_SECONDS } from 'src/constants'
import { AMPLITUDE_EVENTS } from 'src/types/amplitude'

interface VMwareCredentialsDrawerProps {
  open: boolean
  onClose: () => void
}

interface VMwareCredentialFormValues {
  credentialName: string
  vcenterHost: string
  datacenter: string
  username: string
  password: string
  insecure: boolean
}

const defaultValues: VMwareCredentialFormValues = {
  credentialName: '',
  vcenterHost: '',
  datacenter: '',
  username: '',
  password: '',
  insecure: false
}

export default function VMwareCredentialsDrawer({ open, onClose }: VMwareCredentialsDrawerProps) {
  const { reportError } = useErrorHandler({ component: 'VMwareCredentialsDrawer' })
  const { track } = useAmplitude({ component: 'VMwareCredentialsDrawer' })
  const { refetch: refetchVmwareCreds } = useVmwareCredentialsQuery()

  const form = useForm<VMwareCredentialFormValues>({
    defaultValues,
    mode: 'onChange',
    reValidateMode: 'onChange'
  })

  const {
    watch,
    reset,
    formState: { isValid }
  } = form

  const values = watch()
  const {
    credentialName: credentialNameValue,
    vcenterHost: vcenterHostValue,
    datacenter: datacenterValue,
    username: usernameValue,
    password: passwordValue,
    insecure: insecureValue
  } = values

  const formId = 'vmware-credentials-form'

  const [showPassword, setShowPassword] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [validatingVmwareCreds, setValidatingVmwareCreds] = useState(false)
  const [vmwareCredsValidated, setVmwareCredsValidated] = useState<boolean | null>(null)
  const [formError, setFormError] = useState<string | null>(null)
  const [createdCredentialName, setCreatedCredentialName] = useState<string | null>(null)

  const shouldPollVmwareCreds = Boolean(createdCredentialName && validatingVmwareCreds)

  const handleClickShowPassword = useCallback(() => setShowPassword((p) => !p), [])
  const handleMouseDownPassword = useCallback((e: React.MouseEvent<HTMLButtonElement>) => {
    e.preventDefault()
  }, [])

  const resetDrawerState = useCallback(() => {
    reset(defaultValues)
    setCreatedCredentialName(null)
    setValidatingVmwareCreds(false)
    setVmwareCredsValidated(null)
    setFormError(null)
    setSubmitting(false)
    setShowPassword(false)

    onClose()
  }, [onClose, reset])

  const closeDrawer = useCallback(() => {
    if (createdCredentialName) {
      deleteVMwareCredsWithSecretFlow(createdCredentialName).catch((err) => {
        console.error(`Error deleting cancelled credential: ${createdCredentialName}`, err)
      })
    }

    resetDrawerState()
  }, [createdCredentialName, resetDrawerState])

  useEffect(() => {
    setVmwareCredsValidated(null)
    setFormError(null)
  }, [
    credentialNameValue,
    vcenterHostValue,
    datacenterValue,
    usernameValue,
    passwordValue,
    insecureValue
  ])

  const handleValidationStatus = useCallback(
    (status: string, message?: string) => {
      if (status === 'Succeeded') {
        setVmwareCredsValidated(true)
        setValidatingVmwareCreds(false)

        track(AMPLITUDE_EVENTS.CREDENTIALS_ADDED, {
          credentialType: 'vmware',
          credentialName: createdCredentialName,
          stage: 'validation_success'
        })

        setTimeout(() => {
          refetchVmwareCreds()
          resetDrawerState()
        }, 1500)
      } else if (status === 'Failed') {
        setVmwareCredsValidated(false)
        setValidatingVmwareCreds(false)
        setFormError(message || 'Validation failed')

        track(AMPLITUDE_EVENTS.CREDENTIALS_FAILED, {
          credentialType: 'vmware',
          credentialName: createdCredentialName,
          errorMessage: message || 'Validation failed',
          stage: 'validation'
        })

        reportError(
          new Error(`VMware credential validation failed: ${message || 'Unknown reason'}`),
          {
            context: 'vmware-validation-failure',
            metadata: {
              credentialName: createdCredentialName,
              validationMessage: message,
              action: 'vmware-validation-failed'
            }
          }
        )

        if (createdCredentialName) {
          deleteVMwareCredsWithSecretFlow(createdCredentialName).catch((deleteErr) => {
            console.error(`Error deleting failed credential: ${createdCredentialName}`, deleteErr)
            reportError(deleteErr as Error, {
              context: 'vmware-credential-deletion',
              metadata: {
                credentialName: createdCredentialName,
                action: 'delete-failed-credential'
              }
            })
          })
        }
      }

      setSubmitting(false)
    },
    [createdCredentialName, refetchVmwareCreds, reportError, resetDrawerState, track]
  )

  useInterval(
    async () => {
      if (!createdCredentialName) return
      try {
        const response = await getVmwareCredentials(createdCredentialName)
        if (response?.status?.vmwareValidationStatus) {
          handleValidationStatus(
            response.status.vmwareValidationStatus,
            response.status.vmwareValidationMessage
          )
        }
      } catch (err) {
        console.error('Error validating VMware credentials', err)
        reportError(err as Error, {
          context: 'vmware-validation-polling',
          metadata: {
            credentialName: createdCredentialName,
            action: 'vmware-validation-status-polling'
          }
        })
        setFormError('Error validating VMware credentials')
        setValidatingVmwareCreds(false)
        setSubmitting(false)
      }
    },
    THREE_SECONDS,
    shouldPollVmwareCreds
  )

  const onSubmit = useCallback(
    async (vals: VMwareCredentialFormValues) => {
      try {
        setSubmitting(true)
        setValidatingVmwareCreds(true)
        setFormError(null)

        track(AMPLITUDE_EVENTS.CREDENTIALS_ADDED, {
          credentialType: 'vmware',
          credentialName: vals.credentialName,
          vcenterHost: vals.vcenterHost,
          stage: 'creation_start'
        })

        const response = await createVMwareCredsWithSecretFlow(vals.credentialName, {
          VCENTER_HOST: vals.vcenterHost,
          VCENTER_DATACENTER: vals.datacenter || '',
          VCENTER_USERNAME: vals.username,
          VCENTER_PASSWORD: vals.password,
          ...(vals.insecure && { VCENTER_INSECURE: true })
        })

        setCreatedCredentialName(response.metadata.name)

        track(AMPLITUDE_EVENTS.CREDENTIALS_ADDED, {
          credentialType: 'vmware',
          credentialName: vals.credentialName,
          vcenterHost: vals.vcenterHost,
          stage: 'creation_success'
        })
      } catch (error) {
        console.error('Error creating VMware credentials:', error)

        track(AMPLITUDE_EVENTS.CREDENTIALS_FAILED, {
          credentialType: 'vmware',
          credentialName: vals.credentialName,
          vcenterHost: vals.vcenterHost,
          errorMessage: error instanceof Error ? error.message : String(error),
          stage: 'creation'
        })

        reportError(error as Error, {
          context: 'vmware-credential-creation',
          metadata: {
            credentialName: vals.credentialName,
            vcenterHost: vals.vcenterHost,
            username: vals.username,
            action: 'create-vmware-credential'
          }
        })

        setVmwareCredsValidated(false)
        setValidatingVmwareCreds(false)
        setFormError(
          'Error creating VMware credentials: ' +
            (axios.isAxiosError(error) ? error?.response?.data?.message : String(error))
        )
        setSubmitting(false)
      }
    },
    [reportError, track]
  )

  const isSubmitDisabled = submitting || validatingVmwareCreds || !isValid

  const renderBasicsSection = () => (
    <Section>
      <SectionHeader
        title="Connection Details"
        subtitle="Provide the name and endpoint details for the VMware environment you want to connect."
      />

      <FormGrid gap={2} minWidth={300}>
        <RHFTextField
          name="credentialName"
          label="Credential name"
          placeholder="production-vcenter"
          required
          fullWidth
          size="small"
          rules={{
            required: 'Credential name is required',
            validate: (value: string) =>
              isValidName(value) ||
              'Credential name must start with a lowercase letter/number and use only lowercase letters, numbers, or hyphens.'
          }}
        />

        <RHFTextField
          name="vcenterHost"
          label="vCenter server"
          placeholder="https://vcenter.example.com"
          required
          fullWidth
          size="small"
          rules={{ required: 'vCenter server is required' }}
          InputProps={{
            endAdornment: (
              <InputAdornment position="end">
                <Tooltip
                  title="Enter the FQDN or full URL for the vCenter endpoint."
                  arrow
                  placement="left"
                >
                  <IconButton size="small" tabIndex={-1}>
                    <InfoOutlined fontSize="small" />
                  </IconButton>
                </Tooltip>
              </InputAdornment>
            )
          }}
        />

        <RHFTextField
          name="datacenter"
          label="Datacenter name"
          placeholder="Primary-DC"
          fullWidth
          size="small"
        />
      </FormGrid>
    </Section>
  )

  const renderAuthSection = () => (
    <Section>
      <SectionHeader
        title="User Credentials"
        subtitle="These credentials need permission to manage inventory and read cluster configuration."
      />

      <FormGrid gap={2} minWidth={300}>
        <RHFTextField
          name="username"
          label="Username"
          placeholder="administrator@vsphere.local"
          required
          fullWidth
          size="small"
          rules={{ required: 'Username is required' }}
        />

        <RHFTextField
          name="password"
          label="Password"
          type={showPassword ? 'text' : 'password'}
          required
          fullWidth
          size="small"
          rules={{ required: 'Password is required' }}
          InputProps={{
            endAdornment: (
              <InputAdornment position="end">
                <IconButton
                  onClick={handleClickShowPassword}
                  onMouseDown={handleMouseDownPassword}
                  edge="end"
                  size="small"
                >
                  {showPassword ? <VisibilityOff /> : <Visibility />}
                </IconButton>
              </InputAdornment>
            )
          }}
        />
      </FormGrid>
    </Section>
  )

  const renderSecuritySection = () => (
    <Section>
      <SectionHeader
        title="Connection Options"
        subtitle="Prefer TLS-secure connections. Only disable SSL verification if your environment requires it."
      />

      <RHFToggleField
        name="insecure"
        label="Allow insecure connection"
        description="Skip SSL verification for self-signed or lab environments."
        helperText="Disabling verification may expose credentials in transit."
      />

      <Collapse in={Boolean(insecureValue)} unmountOnExit sx={{ mt: 1.5 }}>
        <Alert severity="warning" variant="outlined">
          Use this option only when you fully trust the network between Platform9 and the vCenter
          host.
        </Alert>
      </Collapse>
    </Section>
  )

  const renderStatusRow = () => (
    <OperationStatus
      display="flex"
      flexDirection="column"
      gap={2}
      loading={validatingVmwareCreds}
      loadingMessage="Validating VMware credentials…"
      success={vmwareCredsValidated === true && Boolean(credentialNameValue)}
      successMessage="VMware credentials created and validated."
      successIcon={<CheckIcon color="success" fontSize="small" />}
      error={formError}
    />
  )

  return (
    <DrawerShell
      open={open}
      onClose={closeDrawer}
      header={
        <DrawerHeader
          title="Add VMware Credentials"
          subtitle="Provide vCenter connection details and validate access"
          onClose={closeDrawer}
        />
      }
      footer={
        <DrawerFooter>
          <ActionButton tone="secondary" onClick={closeDrawer} data-testid="vmware-cred-cancel">
            Cancel
          </ActionButton>
          <ActionButton
            tone="primary"
            type="submit"
            form={formId}
            loading={submitting}
            disabled={isSubmitDisabled}
            data-testid="vmware-cred-submit"
          >
            Save Credential
          </ActionButton>
        </DrawerFooter>
      }
    >
      <DesignSystemForm
        form={form}
        onSubmit={onSubmit}
        keyboardSubmitProps={{ open, onClose: closeDrawer, isSubmitDisabled }}
        data-testid="vmware-cred-form"
        id={formId}
      >
        <SurfaceCard>
          <Box sx={{ display: 'grid', gap: 2 }}>
            <Box
              sx={{
                display: 'grid',
                gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' },
                gap: 2,
                alignItems: 'start'
              }}
            >
              {renderBasicsSection()}
              {renderAuthSection()}
            </Box>

            {renderSecuritySection()}

            {renderStatusRow()}
          </Box>
        </SurfaceCard>
      </DesignSystemForm>
    </DrawerShell>
  )
}
