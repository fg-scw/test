export type SettingsForm = {
  CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD: number
  PERIODIC_SYNC_INTERVAL: string
  VM_ACTIVE_WAIT_INTERVAL_SECONDS: number
  VM_ACTIVE_WAIT_RETRY_LIMIT: number
  DEFAULT_MIGRATION_METHOD: 'hot' | 'cold'
  VCENTER_SCAN_CONCURRENCY_LIMIT: number
  CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE: boolean
  CLEANUP_PORTS_AFTER_MIGRATION_FAILURE: boolean
  POPULATE_VMWARE_MACHINE_FLAVORS: boolean
  VOLUME_AVAILABLE_WAIT_INTERVAL_SECONDS: number
  VOLUME_AVAILABLE_WAIT_RETRY_LIMIT: number
  VCENTER_LOGIN_RETRY_LIMIT: number
  OPENSTACK_CREDS_REQUEUE_AFTER_MINUTES: number
  VMWARE_CREDS_REQUEUE_AFTER_MINUTES: number
  VALIDATE_RDM_OWNER_VMS: boolean
  AUTO_FSTAB_UPDATE: boolean
  DEPLOYMENT_NAME: string
  // Proxy-related fields are UI-only and handled via injectEnvVariables
  PROXY_ENABLED: boolean
  PROXY_HTTP_SCHEME: 'http' | 'https'
  PROXY_HTTP_HOST: string
  PROXY_HTTP_PORT: string
  PROXY_HTTPS_SCHEME: 'http' | 'https'
  PROXY_HTTPS_HOST: string
  PROXY_HTTPS_PORT: string
  NO_PROXY: string
}

type ProxyParts = { scheme: 'http' | 'https'; host: string; port: string }

type ProxyFormState = Pick<
  SettingsForm,
  | 'PROXY_ENABLED'
  | 'PROXY_HTTP_SCHEME'
  | 'PROXY_HTTP_HOST'
  | 'PROXY_HTTP_PORT'
  | 'PROXY_HTTPS_SCHEME'
  | 'PROXY_HTTPS_HOST'
  | 'PROXY_HTTPS_PORT'
  | 'NO_PROXY'
>

export const getGlobalSettingsHelpers = (defaults: SettingsForm) => {
  const parseBool = (v: unknown, fallback: boolean) =>
    typeof v === 'string' ? v.toLowerCase() === 'true' : typeof v === 'boolean' ? v : fallback

  const parseNum = (v: unknown, fallback: number) => {
    const n = typeof v === 'string' ? Number(v) : typeof v === 'number' ? v : NaN
    return Number.isFinite(n) ? n : fallback
  }

  const parseInterval = (val: string): string | undefined => {
    const trimmedVal = val?.trim()
    if (!trimmedVal) return 'Periodic Sync is required'

    const regex = /^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?$/
    const match = trimmedVal.match(regex)

    if (!match || match[0] === '') {
      return 'Use duration format like 5m, 1h30m, 5m30s (units: h,m,s).'
    }

    const hours = match[1] ? Number(match[1]) : 0
    const minutes = match[2] ? Number(match[2]) : 0
    const seconds = match[3] ? Number(match[3]) : 0

    const totalMinutes = hours * 60 + minutes + seconds / 60

    if (isNaN(totalMinutes) || totalMinutes < 5) {
      return 'Interval must be at least 5 minutes'
    }

    return undefined
  }

  const validateProxyUrl = (val: string): string | undefined => {
    const trimmed = (val ?? '').trim()
    if (!trimmed) return undefined
    try {
      const u = new URL(trimmed)
      if (u.protocol !== 'http:' && u.protocol !== 'https:') {
        return 'Proxy must start with http:// or https://'
      }
    } catch {
      return 'Enter a valid URL (e.g. http://proxy.local:3128)'
    }
    return undefined
  }

  const parseProxyParts = (value?: string): ProxyParts => {
    if (!value) return { scheme: 'http', host: '', port: '' }
    try {
      const url = new URL(value)
      const scheme = url.protocol === 'https:' ? 'https' : 'http'
      const defaultPort = scheme === 'https' ? '443' : '80'
      return {
        scheme,
        host: url.hostname,
        port: url.port || defaultPort
      }
    } catch {
      return { scheme: 'http', host: '', port: '' }
    }
  }

  const deriveProxyState = (
    base: SettingsForm,
    envData?: Record<string, string | undefined>
  ): ProxyFormState => {
    const httpProxy = envData?.http_proxy
    const httpsProxy = envData?.https_proxy

    const httpParts = parseProxyParts(httpProxy)
    const httpsParts = parseProxyParts(httpsProxy)

    const proxyEnabled = Boolean(httpProxy || httpsProxy)
    const noProxy = envData?.no_proxy ?? base.NO_PROXY

    return {
      PROXY_ENABLED: proxyEnabled,
      PROXY_HTTP_SCHEME: httpParts.scheme,
      PROXY_HTTP_HOST: httpParts.host,
      PROXY_HTTP_PORT: httpParts.port,
      PROXY_HTTPS_SCHEME: httpsParts.scheme,
      PROXY_HTTPS_HOST: httpsParts.host,
      PROXY_HTTPS_PORT: httpsParts.port,
      NO_PROXY: noProxy
    }
  }

  const applyProxyState = (form: SettingsForm, proxyState: ProxyFormState): SettingsForm => ({
    ...form,
    ...proxyState
  })

  const normalizeNoProxy = (value: string): string =>
    value
      .split(',')
      .map((entry) => entry.trim())
      .filter((entry) => entry.length > 0)
      .join(',')

  const buildEnvPayload = (form: SettingsForm) => {
    const httpScheme = form.PROXY_HTTP_SCHEME ?? 'http'
    const httpHost = (form.PROXY_HTTP_HOST ?? '').trim()
    const httpPort = (form.PROXY_HTTP_PORT ?? '').trim()
    const httpsScheme = form.PROXY_HTTPS_SCHEME ?? 'http'
    const httpsHost = (form.PROXY_HTTPS_HOST ?? '').trim()
    const httpsPort = (form.PROXY_HTTPS_PORT ?? '').trim()
    const proxyEnabled = form.PROXY_ENABLED

    return {
      http_proxy:
        proxyEnabled && httpHost && httpPort ? `${httpScheme}://${httpHost}:${httpPort}` : '',
      https_proxy:
        proxyEnabled && httpsHost && httpsPort ? `${httpsScheme}://${httpsHost}:${httpsPort}` : '',
      no_proxy: normalizeNoProxy((form.NO_PROXY ?? '').trim())
    }
  }

  const toConfigMapData = (f: SettingsForm): Record<string, string> => ({
    CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD: String(f.CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD),
    PERIODIC_SYNC_INTERVAL: f.PERIODIC_SYNC_INTERVAL,
    VM_ACTIVE_WAIT_INTERVAL_SECONDS: String(f.VM_ACTIVE_WAIT_INTERVAL_SECONDS),
    VM_ACTIVE_WAIT_RETRY_LIMIT: String(f.VM_ACTIVE_WAIT_RETRY_LIMIT),
    DEFAULT_MIGRATION_METHOD: f.DEFAULT_MIGRATION_METHOD,
    VCENTER_SCAN_CONCURRENCY_LIMIT: String(f.VCENTER_SCAN_CONCURRENCY_LIMIT),
    CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE: String(f.CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE),
    CLEANUP_PORTS_AFTER_MIGRATION_FAILURE: String(f.CLEANUP_PORTS_AFTER_MIGRATION_FAILURE),
    POPULATE_VMWARE_MACHINE_FLAVORS: String(f.POPULATE_VMWARE_MACHINE_FLAVORS),
    VOLUME_AVAILABLE_WAIT_INTERVAL_SECONDS: String(f.VOLUME_AVAILABLE_WAIT_INTERVAL_SECONDS),
    VOLUME_AVAILABLE_WAIT_RETRY_LIMIT: String(f.VOLUME_AVAILABLE_WAIT_RETRY_LIMIT),
    VCENTER_LOGIN_RETRY_LIMIT: String(f.VCENTER_LOGIN_RETRY_LIMIT),
    OPENSTACK_CREDS_REQUEUE_AFTER_MINUTES: String(f.OPENSTACK_CREDS_REQUEUE_AFTER_MINUTES),
    VMWARE_CREDS_REQUEUE_AFTER_MINUTES: String(f.VMWARE_CREDS_REQUEUE_AFTER_MINUTES),
    VALIDATE_RDM_OWNER_VMS: String(f.VALIDATE_RDM_OWNER_VMS),
    AUTO_FSTAB_UPDATE: String(f.AUTO_FSTAB_UPDATE),
    DEPLOYMENT_NAME: f.DEPLOYMENT_NAME
  })

  const fromConfigMapData = (
    data: Record<string, string | number | undefined> | undefined
  ): SettingsForm => ({
    CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD: parseNum(
      data?.CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD,
      defaults.CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD
    ),
    PERIODIC_SYNC_INTERVAL:
      typeof data?.PERIODIC_SYNC_INTERVAL === 'string'
        ? data.PERIODIC_SYNC_INTERVAL
        : defaults.PERIODIC_SYNC_INTERVAL,
    VM_ACTIVE_WAIT_INTERVAL_SECONDS: parseNum(
      data?.VM_ACTIVE_WAIT_INTERVAL_SECONDS,
      defaults.VM_ACTIVE_WAIT_INTERVAL_SECONDS
    ),
    VM_ACTIVE_WAIT_RETRY_LIMIT: parseNum(
      data?.VM_ACTIVE_WAIT_RETRY_LIMIT,
      defaults.VM_ACTIVE_WAIT_RETRY_LIMIT
    ),
    DEFAULT_MIGRATION_METHOD: (data?.DEFAULT_MIGRATION_METHOD === 'cold' ? 'cold' : 'hot') as
      | 'hot'
      | 'cold',
    VCENTER_SCAN_CONCURRENCY_LIMIT: parseNum(
      data?.VCENTER_SCAN_CONCURRENCY_LIMIT,
      defaults.VCENTER_SCAN_CONCURRENCY_LIMIT
    ),
    CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE: parseBool(
      data?.CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE,
      defaults.CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE
    ),
    CLEANUP_PORTS_AFTER_MIGRATION_FAILURE: parseBool(
      data?.CLEANUP_PORTS_AFTER_MIGRATION_FAILURE,
      defaults.CLEANUP_PORTS_AFTER_MIGRATION_FAILURE
    ),
    POPULATE_VMWARE_MACHINE_FLAVORS: parseBool(
      data?.POPULATE_VMWARE_MACHINE_FLAVORS,
      defaults.POPULATE_VMWARE_MACHINE_FLAVORS
    ),
    VOLUME_AVAILABLE_WAIT_INTERVAL_SECONDS: parseNum(
      data?.VOLUME_AVAILABLE_WAIT_INTERVAL_SECONDS,
      defaults.VOLUME_AVAILABLE_WAIT_INTERVAL_SECONDS
    ),
    VOLUME_AVAILABLE_WAIT_RETRY_LIMIT: parseNum(
      data?.VOLUME_AVAILABLE_WAIT_RETRY_LIMIT,
      defaults.VOLUME_AVAILABLE_WAIT_RETRY_LIMIT
    ),
    VCENTER_LOGIN_RETRY_LIMIT: parseNum(
      data?.VCENTER_LOGIN_RETRY_LIMIT,
      defaults.VCENTER_LOGIN_RETRY_LIMIT
    ),
    OPENSTACK_CREDS_REQUEUE_AFTER_MINUTES: parseNum(
      data?.OPENSTACK_CREDS_REQUEUE_AFTER_MINUTES,
      defaults.OPENSTACK_CREDS_REQUEUE_AFTER_MINUTES
    ),
    VMWARE_CREDS_REQUEUE_AFTER_MINUTES: parseNum(
      data?.VMWARE_CREDS_REQUEUE_AFTER_MINUTES,
      defaults.VMWARE_CREDS_REQUEUE_AFTER_MINUTES
    ),
    VALIDATE_RDM_OWNER_VMS: parseBool(
      data?.VALIDATE_RDM_OWNER_VMS,
      defaults.VALIDATE_RDM_OWNER_VMS
    ),
    AUTO_FSTAB_UPDATE: parseBool(data?.AUTO_FSTAB_UPDATE, defaults.AUTO_FSTAB_UPDATE),
    DEPLOYMENT_NAME:
      typeof data?.DEPLOYMENT_NAME === 'string' ? data.DEPLOYMENT_NAME : defaults.DEPLOYMENT_NAME,
    PROXY_ENABLED: defaults.PROXY_ENABLED,
    PROXY_HTTP_SCHEME: defaults.PROXY_HTTP_SCHEME,
    PROXY_HTTP_HOST: defaults.PROXY_HTTP_HOST,
    PROXY_HTTP_PORT: defaults.PROXY_HTTP_PORT,
    PROXY_HTTPS_SCHEME: defaults.PROXY_HTTPS_SCHEME,
    PROXY_HTTPS_HOST: defaults.PROXY_HTTPS_HOST,
    PROXY_HTTPS_PORT: defaults.PROXY_HTTPS_PORT,
    NO_PROXY: defaults.NO_PROXY
  })

  return {
    parseBool,
    parseNum,
    parseInterval,
    validateProxyUrl,
    parseProxyParts,
    deriveProxyState,
    applyProxyState,
    normalizeNoProxy,
    buildEnvPayload,
    toConfigMapData,
    fromConfigMapData
  }
}
