import { v4 as uuidv4 } from 'uuid'

// Helper function to parse OS_INSECURE from string to boolean
const getBooleanValue = (value: string | undefined): boolean | undefined => {
  if (value === undefined) return undefined
  return value.toLowerCase() === 'true'
}

interface OpenstackCredsParams {
  name?: string
  namespace?: string
  OS_AUTH_URL?: string
  OS_DOMAIN_NAME?: string
  OS_USERNAME?: string
  OS_PASSWORD?: string
  OS_AUTH_TOKEN?: string
  OS_REGION_NAME?: string
  OS_TENANT_NAME?: string
  OS_INSECURE?: string
  existingCredName?: string
}

export const createOpenstackCredsJson = (params: OpenstackCredsParams) => {
  const {
    name,
    namespace = 'migration-system',
    OS_AUTH_URL,
    OS_DOMAIN_NAME,
    OS_USERNAME,
    OS_PASSWORD,
    OS_AUTH_TOKEN,
    OS_REGION_NAME,
    OS_TENANT_NAME,
    OS_INSECURE,
    existingCredName
  } = params || {}

  // If existingCredName is provided, we're using an existing credential
  // and don't need to create a new one
  if (existingCredName) {
    return null
  }

  return {
    apiVersion: 'vjailbreak.k8s.pf9.io/v1alpha1',
    kind: 'OpenstackCreds',
    metadata: {
      name: name || uuidv4(),
      namespace
    },
    spec: {
      osAuthUrl: OS_AUTH_URL,
      osAuthToken: OS_AUTH_TOKEN,
      osUsername: OS_USERNAME,
      osPassword: OS_PASSWORD,
      osDomainName: OS_DOMAIN_NAME,
      osRegionName: OS_REGION_NAME,
      osTenantName: OS_TENANT_NAME,
      osInsecure: getBooleanValue(OS_INSECURE)
    }
  }
}

interface OpenstackCreds {
  OS_USERNAME: string
  OS_USER_DOMAIN_NAME?: string
  OS_PASSWORD: string
  OS_PROJECT_NAME?: string
  OS_PROJECT_DOMAIN_NAME?: string
}

export const createOpenstackTokenRequestBody = (creds: OpenstackCreds) => {
  return {
    auth: {
      identity: {
        methods: ['password'],
        password: {
          user: {
            name: creds.OS_USERNAME,
            domain: { name: creds.OS_USER_DOMAIN_NAME || 'default' },
            password: creds.OS_PASSWORD
          }
        }
      },
      scope: {
        project: {
          name: creds.OS_PROJECT_NAME || 'service',
          domain: { name: creds.OS_PROJECT_DOMAIN_NAME || 'default' }
        }
      }
    }
  }
}
