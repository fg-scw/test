import React, { SyntheticEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { FormProvider, useForm } from 'react-hook-form'
import { useLocation } from 'react-router-dom'
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  FormControl,
  FormHelperText,
  MenuItem,
  Select,
  SelectChangeEvent,
  Snackbar,
  Tab,
  Tabs,
  Typography,
  styled,
  useTheme
} from '@mui/material'
import type { SnackbarCloseReason } from '@mui/material'
import RefreshIcon from '@mui/icons-material/Refresh'
import SettingsOutlinedIcon from '@mui/icons-material/SettingsOutlined'
import HistoryToggleOffOutlinedIcon from '@mui/icons-material/HistoryToggleOffOutlined'
import TuneOutlinedIcon from '@mui/icons-material/TuneOutlined'
import LanOutlinedIcon from '@mui/icons-material/LanOutlined'
import FieldLabel from 'src/components/design-system/ui/FieldLabel'
import FormGrid from 'src/components/design-system/ui/FormGrid'
import InlineHelp from 'src/components/design-system/ui/InlineHelp'
import ToggleField from 'src/components/design-system/ui/ToggleField'
import VDDKUploadTab from './VDDKUploadTab'
import type { VddkUploadStatus } from './VDDKUploadTab'
import { IntervalField as SharedIntervalField, RHFTextField } from 'src/shared/components/forms'
import { getGlobalSettingsHelpers, type SettingsForm } from 'src/features/globalSettings/helpers'
import {
  getSettingsConfigMap,
  updateSettingsConfigMap,
  VERSION_CONFIG_MAP_NAME,
  VERSION_NAMESPACE
} from 'src/api/settings/settings'
import { getPf9EnvConfig, injectEnvVariables } from 'src/api/helpers'
import { CloudUploadOutlined } from '@mui/icons-material'
import { uploadVddkFile } from 'src/api/vddk'
import { useVddkStatusQuery } from 'src/hooks/api/useVddkStatusQuery'

const VDDK_UPLOADED_KEY = 'vddk-uploaded'

const StyledPaper = styled(Box)(({ theme }) => ({
  width: '100%',
  height: '100%',
  minHeight: 0,
  padding: theme.spacing(4),
  boxSizing: 'border-box',
  display: 'flex',
  flexDirection: 'column'
}))

const Footer = styled(Box)(({ theme }) => ({
  display: 'flex',
  justifyContent: 'flex-end',
  gap: theme.spacing(2),
  marginTop: theme.spacing(3),
  paddingTop: theme.spacing(2),
  borderTop: `1px solid ${theme.palette.divider}`
}))

const DEFAULTS: SettingsForm = {
  CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD: 20,
  PERIODIC_SYNC_INTERVAL: '1h',
  VM_ACTIVE_WAIT_INTERVAL_SECONDS: 20,
  VM_ACTIVE_WAIT_RETRY_LIMIT: 15,
  DEFAULT_MIGRATION_METHOD: 'hot',
  VCENTER_SCAN_CONCURRENCY_LIMIT: 10,
  CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE: false,
  CLEANUP_PORTS_AFTER_MIGRATION_FAILURE: false,
  POPULATE_VMWARE_MACHINE_FLAVORS: true,
  VOLUME_AVAILABLE_WAIT_INTERVAL_SECONDS: 10,
  VOLUME_AVAILABLE_WAIT_RETRY_LIMIT: 15,
  VCENTER_LOGIN_RETRY_LIMIT: 5,
  OPENSTACK_CREDS_REQUEUE_AFTER_MINUTES: 60,
  VMWARE_CREDS_REQUEUE_AFTER_MINUTES: 60,
  VALIDATE_RDM_OWNER_VMS: true,
  AUTO_FSTAB_UPDATE: false,
  DEPLOYMENT_NAME: 'vJailbreak',
  PROXY_ENABLED: false,
  PROXY_HTTP_SCHEME: 'http',
  PROXY_HTTP_HOST: '',
  PROXY_HTTP_PORT: '',
  PROXY_HTTPS_SCHEME: 'http',
  PROXY_HTTPS_HOST: '',
  PROXY_HTTPS_PORT: '',
  NO_PROXY: 'localhost,127.0.0.1'
}

const helpers = getGlobalSettingsHelpers(DEFAULTS)
type TabKey = 'general' | 'retry' | 'network' | 'advanced' | 'vddk'

const TAB_FIELD_KEYS: Record<TabKey, Array<keyof SettingsForm>> = {
  general: ['DEPLOYMENT_NAME', 'CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD', 'PERIODIC_SYNC_INTERVAL'],
  retry: [
    'VM_ACTIVE_WAIT_INTERVAL_SECONDS',
    'VM_ACTIVE_WAIT_RETRY_LIMIT',
    'VOLUME_AVAILABLE_WAIT_INTERVAL_SECONDS',
    'VOLUME_AVAILABLE_WAIT_RETRY_LIMIT',
    'VCENTER_LOGIN_RETRY_LIMIT',
    'VCENTER_SCAN_CONCURRENCY_LIMIT'
  ],
  network: [
    'PROXY_ENABLED',
    'PROXY_HTTP_SCHEME',
    'PROXY_HTTP_HOST',
    'PROXY_HTTP_PORT',
    'PROXY_HTTPS_SCHEME',
    'PROXY_HTTPS_HOST',
    'PROXY_HTTPS_PORT',
    'NO_PROXY'
  ],
  advanced: [
    'OPENSTACK_CREDS_REQUEUE_AFTER_MINUTES',
    'VMWARE_CREDS_REQUEUE_AFTER_MINUTES',
    'DEFAULT_MIGRATION_METHOD',
    'CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE',
    'CLEANUP_PORTS_AFTER_MIGRATION_FAILURE',
    'POPULATE_VMWARE_MACHINE_FLAVORS',
    'VALIDATE_RDM_OWNER_VMS',
    'AUTO_FSTAB_UPDATE'
  ],
  vddk: []
}

const TAB_ORDER: TabKey[] = ['general', 'retry', 'network', 'advanced', 'vddk']

const TAB_META: Record<TabKey, { label: string; helper: string; icon: React.ReactNode }> = {
  general: {
    label: 'General',
    helper: 'Keep the deployment identity and cadence consistent across the platform.',
    icon: <SettingsOutlinedIcon fontSize="small" />
  },
  retry: {
    label: 'Intervals',
    helper:
      'Control wait intervals, retry tolerances, and concurrency to balance speed vs. safety.',
    icon: <HistoryToggleOffOutlinedIcon fontSize="small" />
  },
  network: {
    label: 'Network',
    helper: 'Configure proxy used by the migration system components.',
    icon: <LanOutlinedIcon fontSize="small" />
  },
  advanced: {
    label: 'Advanced',
    helper: 'Tune integration defaults and automation flags for PCD and VMware flows.',
    icon: <TuneOutlinedIcon fontSize="small" />
  },
  vddk: {
    label: 'VDDK Upload',
    helper: 'Upload and manage VDDK (Virtual Disk Development Kit) files for VMware integration.',
    icon: <CloudUploadOutlined fontSize="small" />
  }
}

const TabLabel = ({
  label,
  showError,
  icon
}: {
  label: string
  showError: boolean
  icon: React.ReactNode
}) => (
  <Box display="flex" alignItems="center" gap={0.75}>
    <Box component="span" sx={{ display: 'flex', alignItems: 'center', color: 'text.secondary' }}>
      {icon}
    </Box>
    <Typography variant="body2" fontWeight={600}>
      {label}
    </Typography>
    {showError ? (
      <Box
        component="span"
        sx={{ width: 8, height: 8, bgcolor: 'error.main', borderRadius: '50%' }}
      />
    ) : null}
  </Box>
)

const TabPanel = ({
  children,
  activeTab,
  value
}: {
  children: React.ReactNode
  activeTab: TabKey
  value: TabKey
}) => {
  const isActive = activeTab === value
  return (
    <Box
      role="tabpanel"
      id={`settings-tabpanel-${value}`}
      aria-labelledby={`settings-tab-${value}`}
      hidden={!isActive}
      sx={{ pt: isActive ? 3 : 0, display: isActive ? 'block' : 'none' }}
    >
      {children}
    </Box>
  )
}

const FIELD_TOOLTIPS: Record<keyof SettingsForm, string> = {
  DEPLOYMENT_NAME: 'Display name shown across dashboards and exported workflows.',
  CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD: 'Number of iterations to copy changed blocks.',
  PERIODIC_SYNC_INTERVAL: 'Frequency for background periodic sync jobs (minimum 5 minutes).',
  VM_ACTIVE_WAIT_INTERVAL_SECONDS: 'Interval to wait for VM to become active (in seconds).',
  VM_ACTIVE_WAIT_RETRY_LIMIT: 'Number of retries to wait for VM to become active.',
  VOLUME_AVAILABLE_WAIT_INTERVAL_SECONDS:
    'Delay between retries while tracking volume availability.',
  VOLUME_AVAILABLE_WAIT_RETRY_LIMIT: 'Maximum attempts to wait for volumes before failing the job.',
  VCENTER_LOGIN_RETRY_LIMIT:
    'Number of login retries before the workflow surfaces an authentication error.',
  VCENTER_SCAN_CONCURRENCY_LIMIT: 'Maximum number of vCenter VMs to scan concurrently.',
  OPENSTACK_CREDS_REQUEUE_AFTER_MINUTES:
    'Time before failed PCD credentials are re-queued for another attempt.',
  VMWARE_CREDS_REQUEUE_AFTER_MINUTES: 'Time before VMware credential rotations are retried.',
  DEFAULT_MIGRATION_METHOD:
    'Default method for VM migration (placeholder for future releases, not currently used).',
  CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE:
    'Automatically delete intermediate volumes when a conversion fails.',
  CLEANUP_PORTS_AFTER_MIGRATION_FAILURE:
    'Automatically delete network ports when a migration fails.',
  POPULATE_VMWARE_MACHINE_FLAVORS:
    'Fetch VMware hardware flavors to enrich instance sizing details.',
  VALIDATE_RDM_OWNER_VMS: 'Ensure Raw Device Mapping owners are validated before migration.',
  AUTO_FSTAB_UPDATE: 'Automatically update fstab entries during VM migration.',
  PROXY_ENABLED: 'Turn on to route outbound HTTP/HTTPS traffic via the configured proxy.',
  PROXY_HTTP_SCHEME:
    "Protocol to use when constructing the HTTP proxy URL (default: 'http'). Many proxies expect http://.",
  PROXY_HTTP_HOST:
    'FQDN or IP of the HTTP proxy server (e.g. proxy.example.com). You may also paste a full URL like http://proxy.example.com:3128 to auto-fill.',
  PROXY_HTTP_PORT: 'TCP port of the HTTP proxy server (e.g. 3128).',
  PROXY_HTTPS_SCHEME:
    "Protocol to use when constructing the HTTPS proxy URL (default: 'http'). Some environments set https_proxy using http://.",
  PROXY_HTTPS_HOST:
    'FQDN or IP of the HTTPS proxy server (e.g. proxy.example.com). You may also paste a full URL like http://proxy.example.com:3128 to auto-fill.',
  PROXY_HTTPS_PORT: 'TCP port of the HTTPS proxy server (e.g. 3129).',
  NO_PROXY:
    'Comma-separated hosts or CIDRs that should bypass the proxy (e.g. localhost,127.0.0.1).'
}

type ToggleKey = Extract<
  keyof SettingsForm,
  | 'CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE'
  | 'CLEANUP_PORTS_AFTER_MIGRATION_FAILURE'
  | 'POPULATE_VMWARE_MACHINE_FLAVORS'
  | 'VALIDATE_RDM_OWNER_VMS'
  | 'AUTO_FSTAB_UPDATE'
>

const TOGGLE_FIELDS: Array<{ key: ToggleKey; label: string; description: string }> = [
  {
    key: 'CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE',
    label: 'Cleanup Volumes After Conversion Failure',
    description: 'Remove orphaned storage artifacts after a failed conversion run.'
  },
  {
    key: 'CLEANUP_PORTS_AFTER_MIGRATION_FAILURE',
    label: 'Cleanup Ports After Migration Failure',
    description: 'Remove orphaned network ports after a failed migration run.'
  },
  {
    key: 'POPULATE_VMWARE_MACHINE_FLAVORS',
    label: 'Populate VMware Machine Flavors',
    description: 'Sync VMware flavor data to pre-fill CPU, memory, and disk sizing hints.'
  },
  {
    key: 'VALIDATE_RDM_OWNER_VMS',
    label: 'Validate MigrationPlan for RDM VMs',
    description: 'Adds guard rails to ensure RDM devices still belong to the reported VM owner.'
  },
  {
    key: 'AUTO_FSTAB_UPDATE',
    label: 'Auto Fstab Update',
    description:
      'Automatically update fstab entries to ensure proper disk mounting after migration.'
  }
]

type NotificationSeverity = 'error' | 'info' | 'success' | 'warning'

type NotificationState = {
  open: boolean
  message: string
  severity: NotificationSeverity
}

type FieldErrorMap = Record<string, string>

const DEFAULT_NOTIFICATION: NotificationState = {
  open: false,
  message: '',
  severity: 'info'
}

const EMPTY_ERRORS: FieldErrorMap = {}

const { parseInterval, validateProxyUrl, deriveProxyState, applyProxyState } = helpers
const { toConfigMapData, fromConfigMapData, buildEnvPayload } = helpers

type UseGlobalSettingsControllerReturn = {
  form: SettingsForm
  errors: FieldErrorMap
  rhfForm: ReturnType<typeof useForm<SettingsForm>>
  loading: boolean
  saving: boolean
  activeTab: TabKey
  setActiveTab: React.Dispatch<React.SetStateAction<TabKey>>
  notification: NotificationState
  proxyUpdateSuccess: boolean
  onText: (e: React.ChangeEvent<HTMLInputElement>) => void
  onBool: (e: React.ChangeEvent<HTMLInputElement>) => void
  onSelect: (e: SelectChangeEvent<string>) => void
  tabHasError: (tab: TabKey) => boolean
  handleTabChange: (_: SyntheticEvent, value: string | number) => void
  onResetDefaults: () => void
  onCancel: () => void
  onSave: (e: React.FormEvent) => Promise<void>
  handleNotificationClose: (_: SyntheticEvent | Event, reason?: SnackbarCloseReason) => void
}

const useGlobalSettingsController = (): UseGlobalSettingsControllerReturn => {
  const [initial, setInitial] = useState<SettingsForm>(DEFAULTS)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [errors, setErrors] = useState<FieldErrorMap>(EMPTY_ERRORS)
  const [activeTab, setActiveTab] = useState<TabKey>('general')
  const [notification, setNotification] = useState<NotificationState>(DEFAULT_NOTIFICATION)
  const [proxyUpdateSuccess, setProxyUpdateSuccess] = useState(false)

  const rhfForm = useForm<SettingsForm>({
    defaultValues: DEFAULTS,
    mode: 'onChange'
  })

  const form = rhfForm.watch() as SettingsForm

  const buildErrors = useCallback((state: SettingsForm): FieldErrorMap => {
    const e: FieldErrorMap = {}

    const cb = state.CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD
    if (!Number.isInteger(cb) || cb < 1 || cb > 20) {
      e.CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD = 'Enter an integer between 1 and 20 (inclusive).'
    }

    const intervalStr = (state.PERIODIC_SYNC_INTERVAL ?? '').trim()
    const intervalError = parseInterval(intervalStr)
    if (intervalError) {
      e.PERIODIC_SYNC_INTERVAL = intervalError
    }

    if (state.DEFAULT_MIGRATION_METHOD !== 'hot' && state.DEFAULT_MIGRATION_METHOD !== 'cold') {
      e.DEFAULT_MIGRATION_METHOD = "Must be 'hot' or 'cold'."
    }

    const requiredAtLeastOne: Array<keyof SettingsForm> = [
      'VM_ACTIVE_WAIT_INTERVAL_SECONDS',
      'VM_ACTIVE_WAIT_RETRY_LIMIT',
      'VCENTER_SCAN_CONCURRENCY_LIMIT',
      'VOLUME_AVAILABLE_WAIT_INTERVAL_SECONDS',
      'VOLUME_AVAILABLE_WAIT_RETRY_LIMIT',
      'OPENSTACK_CREDS_REQUEUE_AFTER_MINUTES',
      'VMWARE_CREDS_REQUEUE_AFTER_MINUTES'
    ]

    requiredAtLeastOne.forEach((k) => {
      const v = state[k] as unknown as number
      if (!Number.isFinite(v) || !Number.isInteger(v) || v < 1) {
        e[String(k)] = 'Enter an integer >= 1.'
      }
    })

    const loginRetry = state.VCENTER_LOGIN_RETRY_LIMIT
    if (!Number.isFinite(loginRetry) || !Number.isInteger(loginRetry) || loginRetry < 0) {
      e.VCENTER_LOGIN_RETRY_LIMIT = 'Enter an integer >= 0.'
    }

    const bools: Array<keyof SettingsForm> = [
      'CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE',
      'CLEANUP_PORTS_AFTER_MIGRATION_FAILURE',
      'POPULATE_VMWARE_MACHINE_FLAVORS',
      'VALIDATE_RDM_OWNER_VMS',
      'AUTO_FSTAB_UPDATE'
    ]
    bools.forEach((k) => {
      const val = state[k]
      if (typeof val !== 'boolean') {
        e[String(k)] = 'Must be boolean: true or false.'
      }
    })

    const dn = (state.DEPLOYMENT_NAME ?? '').trim()
    if (!dn) {
      e.DEPLOYMENT_NAME = 'Required.'
    } else if (dn.length > 63) {
      e.DEPLOYMENT_NAME = 'Must be 63 characters or fewer.'
    }

    const proxyEnabled = state.PROXY_ENABLED
    const httpScheme = state.PROXY_HTTP_SCHEME
    const httpHost = (state.PROXY_HTTP_HOST ?? '').trim()
    const httpPort = (state.PROXY_HTTP_PORT ?? '').trim()
    const httpsScheme = state.PROXY_HTTPS_SCHEME
    const httpsHost = (state.PROXY_HTTPS_HOST ?? '').trim()
    const httpsPort = (state.PROXY_HTTPS_PORT ?? '').trim()
    const noProxyRaw = state.NO_PROXY ?? ''

    const fqdnRegex =
      /^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)(?:\.(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?))+$/
    const ipv4Regex =
      /^(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(?:\.(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}$/

    const isValidNoProxyEntry = (value: string) => {
      if (!value) return false
      if (/^https?:\/\//i.test(value)) return false

      if (ipv4Regex.test(value)) return true

      const cidrMatch = value.match(/^(.+)\/(\d{1,2})$/)
      if (cidrMatch) {
        const ip = cidrMatch[1]
        const mask = Number(cidrMatch[2])
        if (ipv4Regex.test(ip) && Number.isInteger(mask) && mask >= 0 && mask <= 32) {
          return true
        }
      }

      if (value === 'localhost') return true

      const hostname = value.replace(/^\*\./, '').replace(/^\./, '')
      if (!hostname) return false

      // permissive hostname / domain check (supports internal hostnames and FQDNs)
      if (!/^[a-zA-Z0-9-\.]+$/.test(hostname)) return false
      if (hostname.startsWith('-') || hostname.endsWith('-')) return false
      if (hostname.includes('..')) return false

      return hostname.split('.').every((part) => {
        if (!part) return false
        if (part.length > 63) return false
        if (!/^[a-zA-Z0-9-]+$/.test(part)) return false
        if (part.startsWith('-') || part.endsWith('-')) return false
        return true
      })
    }

    const isValidHost = (value: string) => fqdnRegex.test(value) || ipv4Regex.test(value)

    const noProxy = noProxyRaw.trim()
    if (noProxy) {
      const entries = noProxy.split(',').map((entry) => entry.trim())
      const hasEmpty = entries.some((entry) => entry.length === 0)

      if (hasEmpty) {
        e.NO_PROXY =
          'Remove empty entries. Use a comma-separated list (e.g. localhost,127.0.0.1,.example.com,10.0.0.0/8).'
      } else {
        const invalid = entries.find((entry) => !isValidNoProxyEntry(entry))
        if (invalid) {
          e.NO_PROXY = `Invalid no_proxy entry: "${invalid}". Use hosts/domains/IPs or IPv4 CIDRs (no http:// or https://).`
        }
      }
    }

    if (proxyEnabled) {
      if (httpScheme !== 'http' && httpScheme !== 'https') {
        e.PROXY_HTTP_SCHEME = "Must be 'http' or 'https'."
      }

      if (httpsScheme !== 'http' && httpsScheme !== 'https') {
        e.PROXY_HTTPS_SCHEME = "Must be 'http' or 'https'."
      }

      const validateHostPort = (
        hostKey: keyof SettingsForm,
        portKey: keyof SettingsForm,
        hostVal: string,
        portVal: string,
        scheme: 'http' | 'https'
      ) => {
        if (!hostVal && !portVal) {
          return
        }

        if (!hostVal) {
          e[String(hostKey)] =
            `${scheme.toUpperCase()} proxy server is required when proxy is enabled.`
        } else if (/^https?:\/\//i.test(hostVal)) {
          e[String(hostKey)] =
            'Enter only the FQDN or IPv4 address (or paste the full URL and it will auto-fill).'
        } else if (!isValidHost(hostVal)) {
          e[String(hostKey)] = 'Enter a valid FQDN (with at least one dot) or IPv4 address.'
        }

        if (!portVal) {
          e[String(portKey)] =
            `${scheme.toUpperCase()} proxy port is required when proxy is enabled.`
        } else {
          const portNum = Number(portVal)
          if (!Number.isInteger(portNum) || portNum < 1 || portNum > 65535) {
            e[String(portKey)] = 'Enter a valid TCP port between 1 and 65535.'
          }
        }

        if (!e[String(hostKey)] && !e[String(portKey)]) {
          const proxyUrl = `${scheme}://${hostVal}:${portVal}`
          const proxyError = validateProxyUrl(proxyUrl)
          if (proxyError) {
            e[String(hostKey)] = proxyError
          }
        }
      }

      validateHostPort('PROXY_HTTP_HOST', 'PROXY_HTTP_PORT', httpHost, httpPort, httpScheme)
      validateHostPort('PROXY_HTTPS_HOST', 'PROXY_HTTPS_PORT', httpsHost, httpsPort, httpsScheme)
    }

    return e
  }, [])

  const validateForm = useCallback(
    (state: SettingsForm) => {
      const nextErrors = buildErrors(state)
      setErrors(nextErrors)
      return Object.keys(nextErrors).length === 0
    },
    [buildErrors]
  )

  useEffect(() => {
    const subscription = rhfForm.watch((value) => {
      setErrors(buildErrors(value as SettingsForm))
    })

    return () => subscription.unsubscribe()
  }, [buildErrors, rhfForm])

  const fetchSettings = useCallback(async () => {
    setLoading(true)
    try {
      const [settingsCm, pf9Env] = await Promise.all([
        getSettingsConfigMap(),
        getPf9EnvConfig().catch((err) => {
          console.error('Failed to fetch pf9-env config:', err)
          return undefined
        })
      ])

      const base = fromConfigMapData(
        settingsCm?.data as Record<string, string | number | undefined> | undefined
      )
      const proxyState = deriveProxyState(base, pf9Env?.data)
      const merged = applyProxyState(base, proxyState)

      rhfForm.reset(merged)
      setInitial(merged)
      setErrors(buildErrors(merged))
    } catch (err) {
      console.error('Failed to load Global Settings:', err)
    } finally {
      setLoading(false)
    }
  }, [buildErrors, rhfForm])

  useEffect(() => {
    fetchSettings()
  }, [fetchSettings])

  const show = useCallback((message: string, severity: NotificationSeverity = 'info') => {
    setNotification({ open: true, message, severity })
  }, [])

  const handleNotificationClose = useCallback(
    (_: SyntheticEvent | Event, reason?: SnackbarCloseReason) => {
      if (reason === 'clickaway') return
      setNotification((prev) => ({ ...prev, open: false }))
    },
    []
  )

  const onText = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const { name, value } = e.target
      rhfForm.setValue(name as keyof SettingsForm, value as any, { shouldValidate: true })
    },
    [rhfForm]
  )

  const onBool = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const { name, checked } = e.target
      rhfForm.setValue(name as keyof SettingsForm, checked as any, { shouldValidate: true })

      if (name === 'PROXY_ENABLED' && !checked) {
        setProxyUpdateSuccess(false)
      }
    },
    [rhfForm]
  )

  const onSelect = useCallback(
    (e: SelectChangeEvent<string>) => {
      const { name, value } = e.target
      rhfForm.setValue(name as keyof SettingsForm, value as any, { shouldValidate: true })
    },
    [rhfForm]
  )

  const onResetDefaults = useCallback(() => {
    rhfForm.reset({ ...DEFAULTS })
  }, [rhfForm])

  const onCancel = useCallback(() => {
    rhfForm.reset({ ...initial })
  }, [initial, rhfForm])

  const onSave = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault()

      setProxyUpdateSuccess(false)
      if (!validateForm(form)) {
        show('Please fix the validation errors.', 'error')
        return
      }

      const proxyChanged =
        form.PROXY_ENABLED !== initial.PROXY_ENABLED ||
        form.PROXY_HTTP_SCHEME !== initial.PROXY_HTTP_SCHEME ||
        form.PROXY_HTTP_HOST !== initial.PROXY_HTTP_HOST ||
        form.PROXY_HTTP_PORT !== initial.PROXY_HTTP_PORT ||
        form.PROXY_HTTPS_SCHEME !== initial.PROXY_HTTPS_SCHEME ||
        form.PROXY_HTTPS_HOST !== initial.PROXY_HTTPS_HOST ||
        form.PROXY_HTTPS_PORT !== initial.PROXY_HTTPS_PORT ||
        form.NO_PROXY !== initial.NO_PROXY

      setSaving(true)
      try {
        const existingCm = await getSettingsConfigMap()
        const updatedData = toConfigMapData(form)

        await updateSettingsConfigMap({
          apiVersion: existingCm?.apiVersion || 'v1',
          kind: existingCm?.kind || 'ConfigMap',
          metadata: {
            ...(existingCm?.metadata as any),
            name: VERSION_CONFIG_MAP_NAME,
            namespace: VERSION_NAMESPACE
          },
          data: {
            ...(existingCm?.data as any),
            ...updatedData
          }
        } as any)

        let envInjectionFailed = false

        try {
          await injectEnvVariables(buildEnvPayload(form))
        } catch (envErr) {
          envInjectionFailed = true
          console.error('Failed to inject proxy env variables:', envErr)
        }

        let nextState = form

        if (proxyChanged) {
          try {
            const pf9Env = await getPf9EnvConfig()
            const proxyState = deriveProxyState(form, pf9Env?.data)
            nextState = applyProxyState(form, proxyState)
          } catch (refetchErr) {
            console.error('Failed to refetch pf9-env config after save:', refetchErr)
          }
        }

        rhfForm.reset(nextState)
        setInitial(nextState)
        setErrors(buildErrors(nextState))

        if (envInjectionFailed) {
          show(
            'Settings saved, but applying proxy environment variables failed. Please verify connectivity and try again.',
            'warning'
          )
        } else {
          if (proxyChanged) {
            setProxyUpdateSuccess(true)
          }
          show('Global Settings saved successfully.', 'success')
        }
      } catch (err) {
        console.error('Failed to save Global Settings ConfigMap:', err)
        show('Failed to save Global Settings. No changes were applied.', 'error')
      } finally {
        setSaving(false)
      }
    },
    [form, initial, validateForm, show, buildErrors, rhfForm]
  )

  const tabErrorFlags = useMemo(
    () =>
      TAB_ORDER.reduce<Record<TabKey, boolean>>(
        (acc, tab) => {
          acc[tab] = TAB_FIELD_KEYS[tab].some((key) => Boolean(errors[String(key)]))
          return acc
        },
        {} as Record<TabKey, boolean>
      ),
    [errors]
  )

  const tabHasError = useCallback((tab: TabKey) => tabErrorFlags[tab], [tabErrorFlags])

  const handleTabChange = useCallback((_: SyntheticEvent, value: string | number) => {
    setActiveTab(value as TabKey)
  }, [])

  return {
    form,
    errors,
    rhfForm,
    loading,
    saving,
    activeTab,
    setActiveTab,
    notification,
    proxyUpdateSuccess,
    onText,
    onBool,
    onSelect,
    tabHasError,
    handleTabChange,
    onResetDefaults,
    onCancel,
    onSave,
    handleNotificationClose
  }
}

export default function GlobalSettingsPage() {
  const theme = useTheme()
  const location = useLocation()
  const {
    form,
    errors,
    rhfForm,
    loading,
    saving,
    activeTab,
    setActiveTab,
    notification,
    proxyUpdateSuccess,
    onText,
    onBool,
    onSelect,
    tabHasError,
    handleTabChange,
    onResetDefaults,
    onCancel,
    onSave,
    handleNotificationClose
  } = useGlobalSettingsController()

  const activeTabRef = useRef(activeTab)
  useEffect(() => {
    activeTabRef.current = activeTab
  }, [activeTab])

  const appliedLocationTabRef = useRef<TabKey | null>(null)
  useEffect(() => {
    const requestedTab = (location.state as any)?.tab as TabKey | undefined
    if (!requestedTab) {
      appliedLocationTabRef.current = null
      return
    }
    if (!TAB_ORDER.includes(requestedTab)) return
    if (appliedLocationTabRef.current === requestedTab) return
    appliedLocationTabRef.current = requestedTab
    if (requestedTab === activeTabRef.current) return
    setActiveTab(requestedTab)
  }, [location.state, setActiveTab])

  const [proxyHelpDismissed, setProxyHelpDismissed] = useState(false)

  const [vddkFile, setVddkFile] = useState<File | null>(null)
  const [vddkStatus, setVddkStatus] = useState<VddkUploadStatus>('idle')
  const vddkStatusRef = useRef(vddkStatus)
  vddkStatusRef.current = vddkStatus

  const [vddkProgress, setVddkProgress] = useState(0)
  const vddkProgressRef = useRef(0)
  const [vddkMessage, setVddkMessage] = useState('')
  const vddkMessageRef = useRef('')
  const [vddkExtractedPath, setVddkExtractedPath] = useState('')

  const vddkStatusQuery = useVddkStatusQuery({ refetchOnWindowFocus: false })
  const existingVddkPath = vddkStatusQuery.data?.uploaded ? vddkStatusQuery.data?.path || '' : ''
  const existingVddkVersion = vddkStatusQuery.data?.version || ''

  const validateVddkFile = useCallback((file: File) => {
    const validExtensions = ['.tar', '.tar.gz', '.tgz']
    const isValid = validExtensions.some((ext) => file.name.toLowerCase().endsWith(ext))
    if (!isValid) {
      return 'Invalid file type. Please select a .tar or .tar.gz file.'
    }
    if (file.size > 500 * 1024 * 1024) {
      return 'File size exceeds 500MB limit.'
    }
    return null
  }, [])

  const handleVddkFileSelected = useCallback((file: File | null) => {
    if (!file) return

    setVddkFile(file)
    setVddkStatus('idle')
    setVddkProgress(0)
    vddkProgressRef.current = 0
    setVddkMessage('')
    vddkMessageRef.current = ''
    setVddkExtractedPath('')
  }, [])

  const handleVddkClear = useCallback(() => {
    setVddkFile(null)
    setVddkStatus('idle')
    setVddkProgress(0)
    vddkProgressRef.current = 0
    setVddkMessage('')
    vddkMessageRef.current = ''
    setVddkExtractedPath('')
  }, [])

  const handleSave = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault()

      if (vddkStatus === 'uploading') return

      if (vddkFile) {
        const error = validateVddkFile(vddkFile)
        if (error) {
          setVddkStatus('error')
          setVddkMessage(error)
          return
        }

        try {
          setVddkStatus('uploading')
          setVddkProgress(0)
          vddkProgressRef.current = 0
          setVddkMessage('Uploading VDDK file...')
          vddkMessageRef.current = 'Uploading VDDK file...'

          const response = await uploadVddkFile(vddkFile, {
            onProgress: (next) => {
              vddkProgressRef.current = next
              if (activeTabRef.current === 'vddk') {
                setVddkProgress(next)
              }
            }
          })

          setVddkExtractedPath(response.extracted_path || '')
          localStorage.setItem(VDDK_UPLOADED_KEY, 'true')

          // Refetch VDDK status to check if the uploaded file is a valid VDDK
          const statusResult = await vddkStatusQuery.refetch()
          const version = statusResult.data?.version

          if (!version) {
            setVddkStatus('error')
            const nextMessage =
              'Warning: The uploaded file may not be a valid VDDK. Could not detect VDDK version. Please ensure you uploaded the correct VMware VDDK tar file.'
            setVddkMessage(nextMessage)
            vddkMessageRef.current = nextMessage
          } else {
            setVddkStatus('success')
            const nextMessage =
              response.message ||
              `VDDK file uploaded and extracted successfully! Detected version: ${version}`
            setVddkMessage(nextMessage)
            setVddkFile(null)
            vddkMessageRef.current = nextMessage
          }
        } catch (err) {
          setVddkStatus('error')
          const nextMessage = err instanceof Error ? err.message : 'Upload failed'
          setVddkMessage(nextMessage)
          vddkMessageRef.current = nextMessage
          return
        }
      } else if (activeTab === 'vddk') {
        return
      }

      await onSave(e)
    },
    [onSave, validateVddkFile, vddkFile, vddkStatus]
  )

  useEffect(() => {
    if (activeTab !== 'vddk') return
    setVddkProgress(vddkProgressRef.current)
    setVddkMessage(vddkMessageRef.current)
  }, [activeTab])

  useEffect(() => {
    if (!form.PROXY_ENABLED) {
      setProxyHelpDismissed(false)
    }
  }, [form.PROXY_ENABLED])

  useEffect(() => {
    if (proxyUpdateSuccess) {
      setProxyHelpDismissed(false)
    }
  }, [proxyUpdateSuccess])

  useEffect(() => {
    if (vddkStatus !== 'uploading') return

    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault()
      e.returnValue = 'File upload is in progress. Are you sure you want to leave?'
      return e.returnValue
    }

    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [vddkStatus])

  useEffect(() => {
    ;(window as any).__VDDK_UPLOAD_IN_PROGRESS__ = vddkStatus === 'uploading'
    return () => {
      ;(window as any).__VDDK_UPLOAD_IN_PROGRESS__ = false
    }
  }, [vddkStatus])

  const tabProps = (value: TabKey) => ({
    id: `settings-tab-${value}`,
    'aria-controls': `settings-tabpanel-${value}`
  })

  if (loading) {
    return (
      <StyledPaper>
        <FormProvider {...rhfForm}>
          <Box display="flex" justifyContent="center" alignItems="center" height="400px">
            <CircularProgress />
            <Typography variant="body1" sx={{ ml: 2 }}>
              Loading Global Settings...
            </Typography>
          </Box>
        </FormProvider>
      </StyledPaper>
    )
  }

  return (
    <StyledPaper>
      <FormProvider {...rhfForm}>
        <Box
          component="form"
          onSubmit={handleSave}
          data-testid="global-settings-form"
          sx={{
            display: 'flex',
            flexDirection: 'column',
            flex: 1,
            minHeight: 0
          }}
        >
          <Tabs
            value={activeTab}
            onChange={handleTabChange}
            variant="scrollable"
            allowScrollButtonsMobile
            data-testid="global-settings-tabs"
            sx={{ borderBottom: (theme) => `1px solid ${theme.palette.divider}` }}
          >
            {TAB_ORDER.map((tab) => (
              <Tab
                key={tab}
                value={tab}
                data-testid={`global-settings-tab-${tab}`}
                data-tour={tab === 'vddk' ? 'settings-tab-vddk' : undefined}
                label={
                  <TabLabel
                    label={TAB_META[tab].label}
                    showError={tabHasError(tab)}
                    icon={TAB_META[tab].icon}
                  />
                }
                {...tabProps(tab)}
              />
            ))}
          </Tabs>

          <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
            {TAB_META[activeTab].helper}
          </Typography>

          <TabPanel activeTab={activeTab} value="general">
            <FormGrid minWidth={320} gap={2}>
              <RHFTextField
                name="DEPLOYMENT_NAME"
                label="Deployment Name"
                required
                labelProps={{ tooltip: FIELD_TOOLTIPS.DEPLOYMENT_NAME }}
                error={Boolean(errors.DEPLOYMENT_NAME)}
                helperText={errors.DEPLOYMENT_NAME}
              />

              <RHFTextField
                name="CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD"
                label="Changed Blocks Copy Iteration Threshold"
                type="number"
                labelProps={{ tooltip: FIELD_TOOLTIPS.CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD }}
                error={Boolean(errors.CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD)}
                helperText={errors.CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD}
                onValueChange={(value) => {
                  rhfForm.setValue(
                    'CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD',
                    value === '' ? ('' as any) : Number(value),
                    { shouldValidate: true }
                  )
                }}
              />

              <Box display="flex" flexDirection="column" gap={0.5}>
                <FieldLabel label="Periodic Sync" tooltip={FIELD_TOOLTIPS.PERIODIC_SYNC_INTERVAL} />
                <SharedIntervalField
                  label=""
                  name="PERIODIC_SYNC_INTERVAL"
                  value={form.PERIODIC_SYNC_INTERVAL}
                  onChange={onText}
                  error={errors.PERIODIC_SYNC_INTERVAL}
                />
              </Box>
            </FormGrid>
          </TabPanel>

          <TabPanel activeTab={activeTab} value="network">
            {proxyUpdateSuccess && !proxyHelpDismissed ? (
              <InlineHelp
                tone="warning"
                icon="warning"
                onClose={() => setProxyHelpDismissed(true)}
                sx={{ mb: 2 }}
              >
                Proxy changes may take up to 2 minutes to take effect across the system.
              </InlineHelp>
            ) : null}
            <FormGrid minWidth={320} gap={2} sx={{ mb: 2 }}>
              <ToggleField
                label="Use Proxy"
                name="PROXY_ENABLED"
                checked={form.PROXY_ENABLED}
                onChange={onBool}
                tooltip={FIELD_TOOLTIPS.PROXY_ENABLED}
                description="Route outbound HTTP/HTTPS traffic via the configured proxy."
                data-testid="global-settings-toggle-PROXY_ENABLED"
              />
            </FormGrid>
            {form.PROXY_ENABLED && (
              <>
                <Box sx={{ mb: 2 }}>
                  <FormGrid minWidth={320} gap={2}>
                    <FormControl
                      fullWidth
                      size="small"
                      error={Boolean(errors.PROXY_HTTP_SCHEME)}
                      sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}
                    >
                      <FieldLabel
                        label="HTTP Proxy Protocol"
                        tooltip={FIELD_TOOLTIPS.PROXY_HTTP_SCHEME}
                      />
                      <Select
                        name="PROXY_HTTP_SCHEME"
                        value={form.PROXY_HTTP_SCHEME}
                        onChange={onSelect}
                        data-testid="global-settings-select-PROXY_HTTP_SCHEME"
                      >
                        <MenuItem value="http">http</MenuItem>
                        {/* <MenuItem value="https">https</MenuItem> */}
                      </Select>
                      {errors.PROXY_HTTP_SCHEME ? (
                        <FormHelperText>{errors.PROXY_HTTP_SCHEME}</FormHelperText>
                      ) : null}
                    </FormControl>

                    <RHFTextField
                      name="PROXY_HTTP_HOST"
                      label="HTTP Proxy Server"
                      required
                      labelProps={{ tooltip: FIELD_TOOLTIPS.PROXY_HTTP_HOST }}
                      error={Boolean(errors.PROXY_HTTP_HOST)}
                      helperText={errors.PROXY_HTTP_HOST}
                      onValueChange={(value) => {
                        if (!/^https?:\/\//i.test(value)) return
                        try {
                          const url = new URL(value)
                          const scheme = url.protocol === 'https:' ? 'https' : 'http'
                          rhfForm.setValue('PROXY_HTTP_SCHEME', scheme, { shouldValidate: true })
                          rhfForm.setValue('PROXY_HTTP_HOST', url.hostname, {
                            shouldValidate: true
                          })
                          if (url.port) {
                            rhfForm.setValue('PROXY_HTTP_PORT', url.port, { shouldValidate: true })
                          }
                        } catch {
                          // ignore - validation will surface errors
                        }
                      }}
                    />

                    <RHFTextField
                      name="PROXY_HTTP_PORT"
                      label="HTTP Proxy Port"
                      required
                      labelProps={{ tooltip: FIELD_TOOLTIPS.PROXY_HTTP_PORT }}
                      error={Boolean(errors.PROXY_HTTP_PORT)}
                      helperText={errors.PROXY_HTTP_PORT}
                    />
                  </FormGrid>
                </Box>

                <Box sx={{ mb: 2 }}>
                  <FormGrid minWidth={320} gap={2}>
                    <FormControl
                      fullWidth
                      size="small"
                      error={Boolean(errors.PROXY_HTTPS_SCHEME)}
                      sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}
                    >
                      <FieldLabel
                        label="HTTPS Proxy Protocol"
                        tooltip={FIELD_TOOLTIPS.PROXY_HTTPS_SCHEME}
                      />
                      <Select
                        name="PROXY_HTTPS_SCHEME"
                        value={form.PROXY_HTTPS_SCHEME}
                        onChange={onSelect}
                        data-testid="global-settings-select-PROXY_HTTPS_SCHEME"
                      >
                        <MenuItem value="http">http</MenuItem>
                        <MenuItem value="https">https</MenuItem>
                      </Select>
                      {errors.PROXY_HTTPS_SCHEME ? (
                        <FormHelperText>{errors.PROXY_HTTPS_SCHEME}</FormHelperText>
                      ) : null}
                    </FormControl>

                    <RHFTextField
                      name="PROXY_HTTPS_HOST"
                      label="HTTPS Proxy Server"
                      required
                      labelProps={{ tooltip: FIELD_TOOLTIPS.PROXY_HTTPS_HOST }}
                      error={Boolean(errors.PROXY_HTTPS_HOST)}
                      helperText={errors.PROXY_HTTPS_HOST}
                      onValueChange={(value) => {
                        if (!/^https?:\/\//i.test(value)) return
                        try {
                          const url = new URL(value)
                          const scheme = url.protocol === 'https:' ? 'https' : 'http'
                          rhfForm.setValue('PROXY_HTTPS_SCHEME', scheme, { shouldValidate: true })
                          rhfForm.setValue('PROXY_HTTPS_HOST', url.hostname, {
                            shouldValidate: true
                          })
                          if (url.port) {
                            rhfForm.setValue('PROXY_HTTPS_PORT', url.port, { shouldValidate: true })
                          }
                        } catch {
                          // ignore - validation will surface errors
                        }
                      }}
                    />

                    <RHFTextField
                      name="PROXY_HTTPS_PORT"
                      label="HTTPS Proxy Port"
                      required
                      labelProps={{ tooltip: FIELD_TOOLTIPS.PROXY_HTTPS_PORT }}
                      error={Boolean(errors.PROXY_HTTPS_PORT)}
                      helperText={errors.PROXY_HTTPS_PORT}
                    />
                  </FormGrid>
                </Box>

                <FormGrid minWidth={320} gap={2}>
                  <RHFTextField
                    name="NO_PROXY"
                    label="No Proxy Hosts"
                    labelProps={{ tooltip: FIELD_TOOLTIPS.NO_PROXY }}
                    error={Boolean(errors.NO_PROXY)}
                    helperText={errors.NO_PROXY}
                  />
                </FormGrid>
              </>
            )}
          </TabPanel>

          <TabPanel activeTab={activeTab} value="retry">
            <FormGrid minWidth={320} gap={2}>
              {(
                [
                  {
                    name: 'VM_ACTIVE_WAIT_INTERVAL_SECONDS',
                    label: 'VM Active Wait Interval (seconds)',
                    tooltip: FIELD_TOOLTIPS.VM_ACTIVE_WAIT_INTERVAL_SECONDS
                  },
                  {
                    name: 'VM_ACTIVE_WAIT_RETRY_LIMIT',
                    label: 'VM Active Retry Limit',
                    tooltip: FIELD_TOOLTIPS.VM_ACTIVE_WAIT_RETRY_LIMIT
                  },
                  {
                    name: 'VOLUME_AVAILABLE_WAIT_INTERVAL_SECONDS',
                    label: 'Volume Wait Interval (seconds)',
                    tooltip: FIELD_TOOLTIPS.VOLUME_AVAILABLE_WAIT_INTERVAL_SECONDS
                  },
                  {
                    name: 'VOLUME_AVAILABLE_WAIT_RETRY_LIMIT',
                    label: 'Volume Retry Limit',
                    tooltip: FIELD_TOOLTIPS.VOLUME_AVAILABLE_WAIT_RETRY_LIMIT
                  },
                  {
                    name: 'VCENTER_LOGIN_RETRY_LIMIT',
                    label: 'vCenter Login Retry Limit',
                    tooltip: FIELD_TOOLTIPS.VCENTER_LOGIN_RETRY_LIMIT
                  },
                  {
                    name: 'VCENTER_SCAN_CONCURRENCY_LIMIT',
                    label: 'vCenter Concurrency Limit',
                    tooltip: FIELD_TOOLTIPS.VCENTER_SCAN_CONCURRENCY_LIMIT
                  }
                ] as const
              ).map((item) => (
                <RHFTextField
                  key={item.name}
                  name={item.name}
                  label={item.label}
                  type="number"
                  labelProps={{ tooltip: item.tooltip }}
                  error={Boolean(errors[item.name])}
                  helperText={errors[item.name]}
                  onValueChange={(value) => {
                    rhfForm.setValue(item.name, value === '' ? ('' as any) : Number(value), {
                      shouldValidate: true
                    })
                  }}
                />
              ))}
            </FormGrid>
          </TabPanel>

          <TabPanel activeTab={activeTab} value="advanced">
            <FormGrid minWidth={320} gap={2}>
              <RHFTextField
                name="OPENSTACK_CREDS_REQUEUE_AFTER_MINUTES"
                label="PCD Creds Requeue After (minutes)"
                type="number"
                labelProps={{ tooltip: FIELD_TOOLTIPS.OPENSTACK_CREDS_REQUEUE_AFTER_MINUTES }}
                error={Boolean(errors.OPENSTACK_CREDS_REQUEUE_AFTER_MINUTES)}
                helperText={errors.OPENSTACK_CREDS_REQUEUE_AFTER_MINUTES}
                onValueChange={(value) => {
                  rhfForm.setValue(
                    'OPENSTACK_CREDS_REQUEUE_AFTER_MINUTES',
                    value === '' ? ('' as any) : Number(value),
                    { shouldValidate: true }
                  )
                }}
              />

              <RHFTextField
                name="VMWARE_CREDS_REQUEUE_AFTER_MINUTES"
                label="VMware Creds Requeue After (minutes)"
                type="number"
                labelProps={{ tooltip: FIELD_TOOLTIPS.VMWARE_CREDS_REQUEUE_AFTER_MINUTES }}
                error={Boolean(errors.VMWARE_CREDS_REQUEUE_AFTER_MINUTES)}
                helperText={errors.VMWARE_CREDS_REQUEUE_AFTER_MINUTES}
                onValueChange={(value) => {
                  rhfForm.setValue(
                    'VMWARE_CREDS_REQUEUE_AFTER_MINUTES',
                    value === '' ? ('' as any) : Number(value),
                    { shouldValidate: true }
                  )
                }}
              />

              <FormControl
                fullWidth
                size="small"
                error={!!errors.DEFAULT_MIGRATION_METHOD}
                sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}
              >
                <FieldLabel
                  label="Default Migration Method"
                  tooltip={FIELD_TOOLTIPS.DEFAULT_MIGRATION_METHOD}
                />
                <Select
                  name="DEFAULT_MIGRATION_METHOD"
                  value={form.DEFAULT_MIGRATION_METHOD}
                  onChange={onSelect}
                  data-testid="global-settings-field-DEFAULT_MIGRATION_METHOD"
                >
                  <MenuItem value="hot">hot</MenuItem>
                  <MenuItem value="cold">cold</MenuItem>
                </Select>
                {errors.DEFAULT_MIGRATION_METHOD && (
                  <FormHelperText>{errors.DEFAULT_MIGRATION_METHOD}</FormHelperText>
                )}
              </FormControl>
            </FormGrid>

            <Typography variant="subtitle2" sx={{ mt: 3, mb: 1 }}>
              Automation Flags
            </Typography>
            <FormGrid minWidth={260} gap={2}>
              {TOGGLE_FIELDS.map(({ key, label, description }) => (
                <ToggleField
                  key={key}
                  label={label}
                  name={String(key)}
                  checked={form[key] as boolean}
                  onChange={onBool}
                  tooltip={FIELD_TOOLTIPS[key]}
                  description={description}
                  data-testid={`global-settings-toggle-${String(key)}`}
                />
              ))}
            </FormGrid>
          </TabPanel>

          <TabPanel activeTab={activeTab} value="vddk">
            <VDDKUploadTab
              selectedFile={vddkFile}
              status={vddkStatus}
              progress={vddkProgress}
              message={vddkMessage}
              extractedPath={vddkExtractedPath}
              existingVddkPath={existingVddkPath}
              existingVddkVersion={existingVddkVersion}
              onFileSelected={handleVddkFileSelected}
              onClear={handleVddkClear}
            />
          </TabPanel>

          <Box sx={{ flexGrow: 1 }} />

          <Footer sx={{ marginTop: 'auto', marginBottom: theme.spacing(3) }}>
            <Button
              variant="outlined"
              color="inherit"
              onClick={onResetDefaults}
              startIcon={<RefreshIcon />}
              data-testid="global-settings-reset-defaults"
            >
              Reset to Defaults
            </Button>
            <Button variant="outlined" onClick={onCancel} data-testid="global-settings-cancel">
              Cancel
            </Button>
            <Button
              variant="contained"
              type="submit"
              color="primary"
              disabled={
                saving ||
                vddkStatus === 'uploading' ||
                (activeTab === 'vddk' && !vddkFile && !existingVddkPath)
              }
              data-tour="global-settings-save"
              startIcon={
                saving || vddkStatus === 'uploading' ? (
                  <CircularProgress size={20} color="inherit" />
                ) : null
              }
              data-testid="global-settings-save"
            >
              {saving || vddkStatus === 'uploading' ? 'Saving...' : 'Save'}
            </Button>
          </Footer>
        </Box>
      </FormProvider>

      <Snackbar
        open={notification.open}
        autoHideDuration={6000}
        onClose={handleNotificationClose}
        anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
        sx={{
          top: { xs: 80, sm: 90 },
          right: { xs: 16, sm: 24 }
        }}
      >
        <Alert
          onClose={handleNotificationClose}
          severity={notification.severity}
          variant="filled"
          sx={{
            minWidth: '350px',
            fontSize: '1rem',
            fontWeight: 600,
            boxShadow: '0 8px 24px rgba(0, 0, 0, 0.25)',
            '& .MuiAlert-icon': {
              fontSize: '28px'
            },
            '& .MuiAlert-message': {
              fontSize: '1rem',
              fontWeight: 600
            }
          }}
        >
          {notification.message}
        </Alert>
      </Snackbar>
    </StyledPaper>
  )
}
