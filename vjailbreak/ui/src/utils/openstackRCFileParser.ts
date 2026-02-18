/**
 * Utility functions for parsing and validating OpenStack RC files
 */

// Common required fields for both authentication methods
const COMMON_REQUIRED_FIELDS = [
  'OS_AUTH_URL',
  'OS_REGION_NAME',
  'OS_TENANT_NAME'
] as const

// Required fields for password-based authentication
const PASSWORD_AUTH_REQUIRED_FIELDS = [
  'OS_USERNAME',
  'OS_PASSWORD',
  'OS_DOMAIN_NAME'
] as const

// Required fields for token-based authentication
const TOKEN_AUTH_REQUIRED_FIELD = 'OS_AUTH_TOKEN'

export const REQUIRED_OPENSTACK_FIELDS = [
  ...COMMON_REQUIRED_FIELDS
] as const

export interface ParseRCFileResult {
  success: boolean
  fields?: Record<string, string>
  error?: string
}

/**
 * Parses the content of an OpenStack RC file and extracts environment variables
 */
export function parseRCFileContent(content: string): Record<string, string> {
  // Remove 'export' keywords from each line
  const cleanedContent = content.replace(/^export\s+/gm, '')
  const parsedFields: Record<string, string> = {}
  const lines = cleanedContent.split('\n')

  // Parse each line as key=value, handling special cases
  for (const line of lines) {
    const trimmed = line.trim()
    if (!trimmed || trimmed.startsWith('#')) continue // Skip empty lines and comments
    const eqIdx = trimmed.indexOf('=')
    if (eqIdx === -1) continue

    const key = trimmed.slice(0, eqIdx).trim()
    let value = trimmed.slice(eqIdx + 1).trim()

    // Remove surrounding quotes from values, if present
    if (
      (value.startsWith('"') && value.endsWith('"')) ||
      (value.startsWith("'") && value.endsWith("'"))
    ) {
      value = value.slice(1, -1)
    }
    parsedFields[key] = value
  }

  // Handle aliases: OS_PROJECT_DOMAIN_NAME -> OS_DOMAIN_NAME
  if (parsedFields.OS_PROJECT_DOMAIN_NAME && !parsedFields.OS_DOMAIN_NAME) {
    parsedFields.OS_DOMAIN_NAME = parsedFields.OS_PROJECT_DOMAIN_NAME
  }
  // Handle aliases: OS_PROJECT_NAME -> OS_TENANT_NAME
  if (parsedFields.OS_PROJECT_NAME && !parsedFields.OS_TENANT_NAME) {
    parsedFields.OS_TENANT_NAME = parsedFields.OS_PROJECT_NAME
  }

  return parsedFields
}

/**
 * Validates that all required fields are present based on authentication method
 * Supports both token-based (OS_AUTH_TOKEN) and password-based (OS_USERNAME + OS_PASSWORD) auth
 */
export function validateRCFileFields(fields: Record<string, string>): {
  valid: boolean
  missingFields: string[]
} {
  const missingFields: string[] = []

  // Check common required fields
  for (const field of COMMON_REQUIRED_FIELDS) {
    if (!fields[field] || fields[field].trim() === '') {
      missingFields.push(field)
    }
  }

  // Check authentication method
  const hasToken = fields[TOKEN_AUTH_REQUIRED_FIELD] && fields[TOKEN_AUTH_REQUIRED_FIELD].trim() !== ''
  const hasUsername = fields['OS_USERNAME'] && fields['OS_USERNAME'].trim() !== ''
  const hasPassword = fields['OS_PASSWORD'] && fields['OS_PASSWORD'].trim() !== ''

  if (hasToken) {
    // Token-based authentication - OS_DOMAIN_NAME is optional
    // No additional required fields
  } else if (hasUsername && hasPassword) {
    // Password-based authentication - OS_DOMAIN_NAME is required
    for (const field of PASSWORD_AUTH_REQUIRED_FIELDS) {
      if (!fields[field] || fields[field].trim() === '') {
        missingFields.push(field)
      }
    }
  } else {
    // Neither authentication method has complete credentials
    if (!hasToken) {
      if (hasUsername || hasPassword) {
        missingFields.push(hasUsername ? 'OS_PASSWORD' : 'OS_USERNAME')
      } else {
        missingFields.push('OS_AUTH_TOKEN or (OS_USERNAME and OS_PASSWORD)')
      }
    }
  }

  return {
    valid: missingFields.length === 0,
    missingFields
  }
}

/**
 * Reads a File and parses it as an OpenStack RC file
 */
export function parseRCFile(file: File): Promise<ParseRCFileResult> {
  return new Promise((resolve) => {
    const reader = new FileReader()

    reader.onload = (e) => {
      try {
        const content = e.target?.result as string
        const parsedFields = parseRCFileContent(content)
        const validation = validateRCFileFields(parsedFields)

        if (validation.valid) {
          resolve({
            success: true,
            fields: parsedFields
          })
        } else {
          resolve({
            success: false,
            error: `Missing required fields: ${validation.missingFields.join(', ')}`
          })
        }
      } catch (error) {
        resolve({
          success: false,
          error: 'Failed to parse the file. Please check the file format.'
        })
      }
    }

    reader.onerror = () => {
      resolve({
        success: false,
        error: 'Failed to read the file. Please try again.'
      })
    }

    reader.readAsText(file)
  })
}
