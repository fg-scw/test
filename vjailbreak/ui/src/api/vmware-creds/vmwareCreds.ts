import axios from '../axios'
import { VJAILBREAK_API_BASE_PATH, VJAILBREAK_DEFAULT_NAMESPACE } from '../constants'
import { GetVMWareCredsList, VMwareCreds } from './model'

export const getVmwareCredentialsList = async (namespace = VJAILBREAK_DEFAULT_NAMESPACE) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/vmwarecreds`
  const response = await axios.get<GetVMWareCredsList>({
    endpoint
  })
  return response?.items
}

export const getVmwareCredentials = async (
  vmwareCredsName,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/vmwarecreds/${vmwareCredsName}`
  const response = await axios.get<VMwareCreds>({
    endpoint
  })
  return response
}

export const postVmwareCredentials = async (body, namespace = VJAILBREAK_DEFAULT_NAMESPACE) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/vmwarecreds`
  const response = await axios.post<VMwareCreds>({
    endpoint,
    data: body
  })
  return response
}

export const deleteVmwareCredentials = async (
  vmwareCredsName,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/vmwarecreds/${vmwareCredsName}`
  const response = await axios.del<VMwareCreds>({
    endpoint
  })
  return response
}

// Create VMware credentials with secret reference
export const createVMwareCredsWithSecret = async (
  name: string,
  secretName: string,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE,
  datacenter?: string
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/vmwarecreds`

  const credBody: any = {
    apiVersion: 'vjailbreak.k8s.pf9.io/v1alpha1',
    kind: 'VMwareCreds',
    metadata: {
      name,
      namespace
    },
    spec: {
      secretRef: {
        name: secretName,
        namespace
      }
    }
  }

  // Use empty string when datacenter is not provided
  credBody.spec.datacenter = datacenter?.trim() || ''

  const response = await axios.post<VMwareCreds>({
    endpoint,
    data: credBody
  })

  return response
}
