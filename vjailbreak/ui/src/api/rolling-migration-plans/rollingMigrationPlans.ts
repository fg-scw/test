import axios from '../axios'
import { VJAILBREAK_API_BASE_PATH, VJAILBREAK_DEFAULT_NAMESPACE } from '../constants'
import { GetRollingMigrationPlansList, RollingMigrationPlan } from './model'

export const getRollingMigrationPlansList = async (namespace = VJAILBREAK_DEFAULT_NAMESPACE) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/rollingmigrationplans`
  const response = await axios.get<GetRollingMigrationPlansList>({
    endpoint
  })
  return response?.items
}

export const getRollingMigrationPlan = async (
  planName: string,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/rollingmigrationplans/${planName}`
  const response = await axios.get<RollingMigrationPlan>({
    endpoint
  })
  return response
}

export const postRollingMigrationPlan = async (
  body: unknown,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/rollingmigrationplans`
  const response = await axios.post<RollingMigrationPlan>({
    endpoint,
    data: body
  })
  return response
}

export const deleteRollingMigrationPlan = async (
  planName: string,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/rollingmigrationplans/${planName}`
  const response = await axios.del<RollingMigrationPlan>({
    endpoint
  })
  return response
}

export const patchRollingMigrationPlan = async (
  planName: string,
  body: unknown,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/rollingmigrationplans/${planName}`
  const response = await axios.patch<RollingMigrationPlan>({
    endpoint,
    data: body,
    config: {
      headers: {
        'Content-Type': 'application/merge-patch+json'
      }
    }
  })
  return response
}

export const getRollingMigrationPlans = async (
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
): Promise<RollingMigrationPlan[]> => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/rollingmigrationplans`
  const response = await axios.get<{ items: RollingMigrationPlan[] }>({
    endpoint
  })
  return response?.items || []
}
