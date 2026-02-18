import { v4 as uuidv4 } from 'uuid'

interface VmwareCredsParams {
  name?: string
  vcenterHost?: string
  username?: string
  password?: string
  namespace?: string
  existingCredName?: string
}

export const createVmwareCredsJson = (params: VmwareCredsParams | null | undefined) => {
  const {
    name,
    vcenterHost,
    username,
    password,
    namespace = 'migration-system',
    existingCredName
  } = params || {}

  // If existingCredName is provided, we're using an existing credential
  // and don't need to create a new one
  if (existingCredName) {
    return null
  }

  return {
    apiVersion: 'vjailbreak.k8s.pf9.io/v1alpha1',
    kind: 'VMwareCreds',
    metadata: {
      name: name || uuidv4(),
      namespace
    },
    spec: {
      VCENTER_HOST: vcenterHost,
      VCENTER_INSECURE: true,
      VCENTER_PASSWORD: password,
      VCENTER_USERNAME: username
    }
  }
}
