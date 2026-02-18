import axios from 'src/api/axios'
import { VJAILBREAK_API_BASE_PATH, VJAILBREAK_DEFAULT_NAMESPACE } from 'src/api/constants'
import { GetMigrationPlansList, MigrationPlan } from './model'

export const getMigrationPlansList = async (namespace = VJAILBREAK_DEFAULT_NAMESPACE) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/migrationplans`
  const response = await axios.get<GetMigrationPlansList>({
    endpoint
  })
  return response?.items
}

export const getMigrationPlan = async (planName, namespace = VJAILBREAK_DEFAULT_NAMESPACE) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/migrationplans/${planName}`
  const response = await axios.get<MigrationPlan>({
    endpoint
  })
  return response
}

export const postMigrationPlan = async (body, namespace = VJAILBREAK_DEFAULT_NAMESPACE) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/migrationplans`
  const response = await axios.post<MigrationPlan>({
    endpoint,
    data: body
  })
  return response
}

export const deleteMigrationPlan = async (planName, namespace = VJAILBREAK_DEFAULT_NAMESPACE) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/migrationplans/${planName}`
  const response = await axios.del<MigrationPlan>({
    endpoint
  })
  return response
}

export const patchMigrationPlan = async (
  planName: string,
  body: unknown,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/migrationplans/${planName}`
  const response = await axios.patch<MigrationPlan>({
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

export const getMigrationPlans = async (
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
): Promise<MigrationPlan[]> => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/migrationplans`
  const response = await axios.get<{ items: MigrationPlan[] }>({
    endpoint
  })
  return response?.items || []
}
