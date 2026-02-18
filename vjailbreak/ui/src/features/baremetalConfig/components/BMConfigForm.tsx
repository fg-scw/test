import { useMemo, useState, useEffect } from 'react'
import React from 'react'
import {
  Box,
  TextField,
  Typography,
  styled,
  Select,
  MenuItem,
  FormControl,
  FormHelperText,
  SelectChangeEvent,
  CircularProgress,
  Snackbar,
  Alert,
  useTheme
} from '@mui/material'
import CodeMirror from '@uiw/react-codemirror'
import { useErrorHandler } from 'src/hooks/useErrorHandler'
import { yaml } from '@codemirror/lang-yaml'
import { EditorView } from '@codemirror/view'
import { createBmconfigSecret, deleteSecret, getSecret } from 'src/api/secrets/secrets'
import {
  createBMConfigWithSecret,
  deleteBMConfig,
  getBMConfigList,
  getBMConfig,
  fetchBootSources,
  BootSourceSelection
} from 'src/api/bmconfig/bmconfig'
import RefreshIcon from '@mui/icons-material/Refresh'
import {
  LIGHT_BG_DEFAULT,
  DARK_BG_ELEVATED,
  LIGHT_TEXT_PRIMARY,
  DARK_TEXT_PRIMARY,
  WHITE,
  BLACK
} from 'src/theme/colors'

import {
  ActionButton,
  FieldLabel,
  Row,
  Section,
  SectionHeader,
  SurfaceCard,
  ToggleField
} from 'src/components'

const LoadingRow = styled(Row)(({ theme }) => ({
  paddingTop: theme.spacing(2),
  paddingBottom: theme.spacing(2)
}))

const FormRoot = styled('form')(({ theme }) => ({
  display: 'grid',
  gap: theme.spacing(2)
}))

const ConnectionGrid = styled(Box)(({ theme }) => ({
  display: 'grid',
  gap: theme.spacing(2),
  alignItems: 'start',
  gridTemplateColumns: '1fr',
  [theme.breakpoints.up('md')]: {
    gridTemplateColumns: 'repeat(3, 1fr)'
  }
}))

const FieldBlock = styled(Box)(({ theme }) => ({
  display: 'grid',
  gap: theme.spacing(0.75),
  alignContent: 'start'
}))

const FieldLabelSlot = styled(Box)({
  minHeight: 0
})

const ToggleRow = styled(Box)(({ theme }) => ({
  marginTop: theme.spacing(1)
}))

const CloudInitField = styled(FormControl)({
  width: '100%'
})

const CodeMirrorContainer = styled(Box, {
  shouldForwardProp: (prop) => prop !== 'hasError'
})<{ hasError?: boolean }>(({ theme, hasError }) => ({
  border: `1px solid ${hasError ? theme.palette.error.main : theme.palette.divider}`,
  borderRadius: theme.shape.borderRadius,
  overflow: 'auto',
  backgroundColor: theme.palette.background.paper
}))

function validateCloudInit(cloudInit: string): string | null {
  const trimmed = cloudInit.trim()

  if (!trimmed) {
    return 'Cloud-init user-data is required.'
  }

  if (cloudInit.includes('\t')) {
    return 'Cloud-init YAML contains tab characters. YAML requires spaces for indentation.'
  }

  const firstNonEmptyLine = cloudInit
    .split(/\r?\n/)
    .map((line) => line.trim())
    .find((line) => line.length > 0)

  if (firstNonEmptyLine !== '#cloud-config') {
    return 'Cloud-init must start with #cloud-config as the first non-empty line.'
  }

  return null
}

export default function MaasConfigForm() {
  const { reportError } = useErrorHandler({ component: 'BMConfigForm' })
  const defaultCloudInit = `#cloud-config

# Run the cloud-init script on boot
runcmd:
  - echo "Hello World" > /root/hello-cloud-init`

  interface FormDataType {
    maasUrl: string
    insecure: boolean
    apiKey: string
    os: string
    configName: string
    namespace: string
    cloudInit: string
  }

  const [formData, setFormData] = useState<FormDataType>({
    maasUrl: '',
    insecure: false,
    apiKey: '',
    os: '',
    configName: 'bmconfig',
    namespace: 'migration-system',
    cloudInit: defaultCloudInit
  })

  const [loading, setLoading] = useState(false)
  const [initialLoading, setInitialLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [bootSources, setBootSources] = useState<BootSourceSelection[]>([])
  const [notification, setNotification] = useState({
    open: false,
    message: '',
    severity: 'info' as 'error' | 'info' | 'success' | 'warning'
  })
  const [urlError, setUrlError] = useState('')

  const cloudInitError = useMemo(() => validateCloudInit(formData.cloudInit), [formData.cloudInit])

  const maasUrlRegex = /^https?:\/\/.+\/MAAS\/?$/

  const theme = useTheme()
  const isDarkMode = theme.palette.mode === 'dark'

  const extensions = React.useMemo(
    () => [
      yaml(),
      EditorView.lineWrapping,
      EditorView.theme({
        '&': {
          fontSize: '14px'
        },
        '.cm-gutters': {
          backgroundColor: isDarkMode ? DARK_BG_ELEVATED : LIGHT_BG_DEFAULT,
          color: isDarkMode ? DARK_TEXT_PRIMARY : LIGHT_TEXT_PRIMARY,
          border: 'none'
        },
        '.cm-content': {
          caretColor: isDarkMode ? WHITE : BLACK
        },
        '&.cm-focused .cm-cursor': {
          borderLeftColor: isDarkMode ? WHITE : BLACK
        },
        '.cm-line': {
          padding: '0 4px'
        }
      })
    ],
    [isDarkMode]
  )

  useEffect(() => {
    fetchExistingMaasConfig()
  }, [])

  const fetchExistingMaasConfig = async () => {
    setInitialLoading(true)
    try {
      const configs = await getBMConfigList(formData.namespace)

      if (configs && configs.length > 0) {
        const config = await getBMConfig(configs[0].metadata.name, formData.namespace)

        if (config && config.spec) {
          let maasUrl = ''
          let insecure = config.spec.insecure || false

          if (config.spec.apiUrl) {
            maasUrl = config.spec.apiUrl
            insecure = config.spec.insecure || false
          }

          let cloudInitData = ''
          if (config.spec.userDataSecretRef && config.spec.userDataSecretRef.name) {
            try {
              const secretData = await getSecret(
                config.spec.userDataSecretRef.name,
                config.spec.userDataSecretRef.namespace || formData.namespace
              )

              if (secretData && secretData.data && secretData.data['user-data']) {
                cloudInitData = secretData.data['user-data']
              }
            } catch (error) {
              console.warn('Error fetching user-data secret:', error)
            }
          }

          setFormData({
            maasUrl,
            insecure,
            apiKey: config.spec.apiKey || '',
            os: config.spec.os || '',
            configName: config.metadata.name,
            namespace: config.metadata.namespace,
            cloudInit: cloudInitData
          })
        }
      }
    } catch (error) {
      console.error('Error fetching existing Bare Metal Config:', error)
    } finally {
      setInitialLoading(false)

      if (formData.maasUrl && formData.apiKey) {
        handleFetchBootSources()
      }
    }
  }

  useEffect(() => {
    if (formData.maasUrl && formData.apiKey && !urlError) {
      handleFetchBootSources()
    }
  }, [formData.maasUrl, formData.apiKey, urlError])

  const handleFetchBootSources = async () => {
    if (!formData.maasUrl || !formData.apiKey || urlError) {
      return
    }

    setLoading(true)
    try {
      const response = await fetchBootSources(formData.maasUrl, formData.apiKey, formData.insecure)

      setBootSources(response.bootSourceSelections)

      if (response.bootSourceSelections.length > 0) {
        const ubuntuJammy = response.bootSourceSelections.find(
          (source) => source.OS === 'ubuntu' && source.Release === 'jammy'
        )

        if (ubuntuJammy) {
          setFormData((prev) => ({
            ...prev,
            os: ubuntuJammy.Release
          }))
        } else if (response.bootSourceSelections.length > 0) {
          const firstSource = response.bootSourceSelections[0]
          setFormData((prev) => ({
            ...prev,
            os: firstSource.Release
          }))
        }
      }
    } catch (error) {
      console.error('Error fetching boot sources:', error)
      setNotification({
        open: true,
        message: 'Failed to fetch boot sources',
        severity: 'error'
      })
    } finally {
      setLoading(false)
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const { name, value } = e.target

    if (name === 'maasUrl') {
      const isValid = maasUrlRegex.test(value)
      setUrlError(value && !isValid ? 'URL must be in format: http(s)://hostname/MAAS' : '')
    }

    setFormData((prev) => ({
      ...prev,
      [name]: value
    }))
  }

  const handleSwitchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, checked } = e.target
    setFormData((prev) => ({
      ...prev,
      [name]: checked
    }))
  }

  const handleSelectChange = (e: SelectChangeEvent<string>) => {
    const { name, value } = e.target
    setFormData((prev) => ({
      ...prev,
      [name]: value
    }))
  }

  const handleCloseNotification = () => {
    setNotification((prev) => ({
      ...prev,
      open: false
    }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (cloudInitError) {
      setNotification({
        open: true,
        message: cloudInitError,
        severity: 'error'
      })
      return
    }

    if (urlError) {
      setNotification({
        open: true,
        message: 'Please fix the Bare Metal Provider URL format',
        severity: 'error'
      })
      return
    }

    setSubmitting(true)

    try {
      const secretName = 'user-data-secret'

      const existingConfigs = await getBMConfigList(formData.namespace)

      if (existingConfigs && existingConfigs.length > 0) {
        for (const config of existingConfigs) {
          const fullConfig = await getBMConfig(config.metadata.name, formData.namespace)

          if (
            fullConfig &&
            fullConfig.spec &&
            fullConfig.spec.userDataSecretRef &&
            fullConfig.spec.userDataSecretRef.name
          ) {
            try {
              const secretName = fullConfig.spec.userDataSecretRef.name
              await deleteSecret(secretName, formData.namespace)
            } catch (secretError) {
              console.warn('Could not delete associated secret:', secretError)
            }
          }

          await deleteBMConfig(config.metadata.name, formData.namespace)
        }
      }

      await createBmconfigSecret(secretName, formData.cloudInit, formData.namespace)

      await createBMConfigWithSecret(
        formData.configName,
        'maas',
        formData.maasUrl,
        formData.apiKey,
        secretName,
        formData.namespace,
        formData.insecure,
        formData.os
      )

      setNotification({
        open: true,
        message: 'Bare Metal Config saved successfully',
        severity: 'success'
      })
    } catch (error) {
      console.error('Error submitting form:', error)
      reportError(error as Error, {
        context: 'baremetal-config-submission',
        metadata: {
          configName: formData.configName,
          maasUrl: formData.maasUrl,
          os: formData.os,
          namespace: formData.namespace,
          action: 'create-baremetal-config'
        }
      })
      setNotification({
        open: true,
        message: 'Failed to save Bare Metal Config',
        severity: 'error'
      })
    } finally {
      setSubmitting(false)
    }
  }

  const handleCancel = () => {
    // Reset form or navigate away
  }

  const handleResetCloudInit = () => {
    setFormData((prev) => ({
      ...prev,
      cloudInit: defaultCloudInit
    }))
  }

  return (
    <SurfaceCard
      title="Bare metal config"
      subtitle="Configure MAAS connection, boot source selection, and cloud-init user-data."
      data-testid="bm-config-card"
      sx={{ borderRadius: 'none' }}
      footer={
        <Row justifyContent="flex-end" gap={2}>
          <ActionButton tone="secondary" onClick={handleCancel} data-testid="bm-config-cancel">
            Cancel
          </ActionButton>
          <ActionButton
            tone="primary"
            type="submit"
            form="bm-config-form"
            loading={submitting}
            disabled={submitting || !formData.os || !!urlError || !!cloudInitError}
            data-testid="bm-config-save"
          >
            Save
          </ActionButton>
        </Row>
      }
    >
      {initialLoading ? (
        <LoadingRow gap={2} alignItems="center">
          <CircularProgress size={20} />
          <Typography variant="body2" color="text.secondary">
            Loading existing configuration...
          </Typography>
        </LoadingRow>
      ) : (
        <FormRoot id="bm-config-form" onSubmit={handleSubmit} data-testid="bm-config-form">
          <Section>
            {/* <SectionHeader
              title="Connection"
              subtitle="MAAS endpoint and credentials used to fetch boot sources."
            /> */}

            <ConnectionGrid>
              <FieldBlock>
                <FieldLabelSlot>
                  <FieldLabel
                    label="Bare Metal Provider URL"
                    required
                    // helperText="Format: http(s)://hostname/MAAS"
                  />
                </FieldLabelSlot>
                <TextField
                  fullWidth
                  name="maasUrl"
                  value={formData.maasUrl}
                  onChange={handleChange}
                  size="small"
                  variant="outlined"
                  placeholder="http://10.9.4.234:5240/MAAS"
                  error={!!urlError}
                  helperText={urlError}
                  data-testid="bm-config-maas-url"
                />
              </FieldBlock>

              <FieldBlock>
                <FieldLabelSlot>
                  <FieldLabel
                    label="API Key"
                    tooltip="The API key for your MAAS account. You can generate this from the MAAS UI under 'API Keys'."
                    required
                    helperText=" "
                  />
                </FieldLabelSlot>
                <TextField
                  fullWidth
                  name="apiKey"
                  value={formData.apiKey}
                  onChange={handleChange}
                  size="small"
                  variant="outlined"
                  type="password"
                  data-testid="bm-config-api-key"
                />
              </FieldBlock>

              <FieldBlock>
                <FieldLabelSlot>
                  <FieldLabel
                    label="OS"
                    tooltip="Ubuntu Jammy is only supported for now."
                    helperText=" "
                  />
                </FieldLabelSlot>

                {loading ? (
                  <Row gap={1} alignItems="center">
                    <CircularProgress size={16} />
                    <Typography variant="body2" color="text.secondary">
                      Loading OS options...
                    </Typography>
                  </Row>
                ) : (
                  <FormControl fullWidth size="small">
                    <Select
                      name="os"
                      value={formData.os}
                      onChange={handleSelectChange}
                      displayEmpty
                      disabled={bootSources?.length === 0}
                      data-testid="bm-config-os-select"
                    >
                      <MenuItem value="" disabled>
                        Select an OS
                      </MenuItem>
                      {bootSources?.map((source) => (
                        <MenuItem
                          key={`${source.OS} (${source.Release})`}
                          value={source.Release}
                          disabled={!(source.OS === 'ubuntu' && source.Release === 'jammy')}
                        >
                          {`${source.OS} (${source.Release})`}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                )}
              </FieldBlock>
            </ConnectionGrid>

            <ToggleRow>
              <ToggleField
                name="insecure"
                checked={formData.insecure}
                onChange={handleSwitchChange}
                label="Insecure"
                tooltip="Disable TLS certificate verification. Use only in trusted networks or lab environments."
                description="Skip SSL verification when connecting to MAAS."
                data-testid="bm-config-insecure-toggle"
              />
            </ToggleRow>
          </Section>

          <Section>
            <SectionHeader
              title="Cloud-init"
              subtitle="User-data YAML applied when machines boot."
              actions={
                <ActionButton
                  tone="secondary"
                  onClick={handleResetCloudInit}
                  startIcon={<RefreshIcon fontSize="small" />}
                  data-testid="bm-config-cloudinit-reset"
                >
                  Reset to Default
                </ActionButton>
              }
            />

            <CloudInitField error={!!cloudInitError}>
              <CodeMirrorContainer
                hasError={!!cloudInitError}
                data-testid="bm-config-cloudinit-editor"
              >
                <CodeMirror
                  value={formData.cloudInit}
                  height="300px"
                  extensions={extensions}
                  theme={isDarkMode ? 'dark' : 'light'}
                  onChange={(value) => {
                    setFormData((prev) => ({
                      ...prev,
                      cloudInit: value
                    }))
                  }}
                  basicSetup={{
                    lineNumbers: true,
                    highlightActiveLine: true,
                    highlightSelectionMatches: true,
                    syntaxHighlighting: true
                  }}
                />
              </CodeMirrorContainer>
              {cloudInitError ? <FormHelperText>{cloudInitError}</FormHelperText> : null}
            </CloudInitField>
          </Section>
        </FormRoot>
      )}

      <Snackbar open={notification.open} autoHideDuration={6000} onClose={handleCloseNotification}>
        <Alert
          onClose={handleCloseNotification}
          severity={notification.severity}
          sx={{ width: '100%' }}
        >
          {notification.message}
        </Alert>
      </Snackbar>
    </SurfaceCard>
  )
}
