import { GetMigrationTemplatesList, MigrationTemplate } from 'src/api/migration-templates/model'
import axios from '../axios'
import { VJAILBREAK_API_BASE_PATH, VJAILBREAK_DEFAULT_NAMESPACE } from '../constants'

export const getMigrationTemplatesList = async (namespace = VJAILBREAK_DEFAULT_NAMESPACE) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/migrationtemplates`
  const response = await axios.get<GetMigrationTemplatesList>({
    endpoint
  })
  return response?.items
}

export const getMigrationTemplate = async (
  templateName,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/migrationtemplates/${templateName}`
  const response = await axios.get<MigrationTemplate>({
    endpoint
  })
  return response
}

export const postMigrationTemplate = async (body, namespace = VJAILBREAK_DEFAULT_NAMESPACE) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/migrationtemplates`
  const response = await axios.post<MigrationTemplate>({
    endpoint,
    data: body
  })
  return response
}

export const putMigrationTemplate = async (
  templateName,
  body,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/migrationtemplates/${templateName}`
  const response = await axios.put<MigrationTemplate>({
    endpoint,
    data: body
  })
  return response
}

export const patchMigrationTemplate = async (
  templateName,
  body,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/migrationtemplates/${templateName}`
  const response = await axios.patch<MigrationTemplate>({
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

export const deleteMigrationTemplate = async (
  templateName,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/migrationtemplates/${templateName}`
  const response = await axios.del<MigrationTemplate>({
    endpoint
  })
  return response
}
