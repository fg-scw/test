import { either, isEmpty, isNil } from 'ramda'

export const isNilOrEmpty = either(isNil, isEmpty)

const duplicatedSlashesRegexp = new RegExp('(^\\/|[^:\\/]+\\/)\\/+', 'g')

// Given some path segments returns a properly formatted path similarly to Nodejs path.join()
// Remove duplicated slashes
// Does not remove leading/trailing slashes and adds a slash between segments
export const pathJoin = (...pathParts) =>
  []
    .concat(...pathParts) // Flatten
    .filter((segment) => !!segment) // Remove empty parts
    .join('/')
    .replace(duplicatedSlashesRegexp, '$1')

/**
 * Checks if the given string is a valid Resource credential name.
 *
 * A valid Resource credential name is a string of up to 253 characters which
 * conforms to the following rules:
 *
 * 1. The first character must be a letter (a-z) or a number (0-9).
 * 2. The remaining characters must be letters, numbers, or hyphens.
 * 3. The string must end with a letter or number.
 *
 * @param {string} name The string to check.
 * @returns {boolean} Whether the string is a valid Resource credential name.
 */
export const isValidName = (name: string) =>
  (/^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/.test(name) && name.length <= 253 && name.length > 0) || !name

export const debounce = (func, delay) => {
  let timeout

  const debouncedFunction = (...args) => {
    clearTimeout(timeout)
    timeout = setTimeout(() => func(...args), delay)
  }

  // Add a cancel method to clear the timeout
  debouncedFunction.cancel = () => {
    clearTimeout(timeout)
  }

  return debouncedFunction
}

/**
 * Calculates the time elapsed since a given timestamp and returns a human-readable string.
 * For completed migrations, shows duration from creation to completion.
 * For running migrations, shows time since creation.
 * @param creationTimestamp - ISO 8601 timestamp string from metadata.creationTimestamp
 * @param status - Optional migration status object with phase and conditions
 * @returns Human-readable time elapsed string (e.g., "5m 30s", "2h 15m", "3d 4h")
 */
export const calculateTimeElapsed = (creationTimestamp: string, status?: any): string => {
  if (!creationTimestamp) {
    return 'N/A'
  }

  try {
    const createdAt = new Date(creationTimestamp)
    let endTime = new Date() // Default to current time for running migrations

    // For completed migrations, use the completion time
    if (status?.phase === 'Succeeded' || status?.phase === 'Failed' || status?.phase === 'ValidationFailed') {
      // For succeeded migrations, look for the Migrated condition type
      // For failed migrations, look for the Failed condition type
      // For validation failed migrations, look for the Validated condition type with VMValidationFailed reason
      let targetConditionType: string
      let targetReason: string

      if (status.phase === 'Succeeded') {
        targetConditionType = 'Migrated'
        targetReason = 'Migration'
      } else if (status.phase === 'Failed') {
        targetConditionType = 'Failed'
        targetReason = 'Migration'
      } else {
        // ValidationFailed
        targetConditionType = 'Validated'
        targetReason = 'VMValidationFailed'
      }

      const completionCondition = status.conditions?.find(
        (condition) => condition.type === targetConditionType && condition.reason === targetReason
      )

      if (completionCondition?.lastTransitionTime) {
        endTime = new Date(completionCondition.lastTransitionTime)
      }
    }

    const diffMs = endTime.getTime() - createdAt.getTime()

    if (diffMs < 0) {
      return 'N/A'
    }

    const diffSeconds = Math.floor(diffMs / 1000)
    const diffMinutes = Math.floor(diffSeconds / 60)
    const diffHours = Math.floor(diffMinutes / 60)
    const diffDays = Math.floor(diffHours / 24)

    if (diffDays > 0) {
      const remainingHours = diffHours % 24
      return remainingHours > 0 ? `${diffDays}d ${remainingHours}h` : `${diffDays}d`
    } else if (diffHours > 0) {
      const remainingMinutes = diffMinutes % 60
      return remainingMinutes > 0 ? `${diffHours}h ${remainingMinutes}m` : `${diffHours}h`
    } else if (diffMinutes > 0) {
      const remainingSeconds = diffSeconds % 60
      return remainingSeconds > 0 ? `${diffMinutes}m ${remainingSeconds}s` : `${diffMinutes}m`
    } else {
      return `${diffSeconds}s`
    }
  } catch (error) {
    console.error('Error calculating time elapsed:', error)
    return 'N/A'
  }
}

/**
 * Formats a byte value into a human-readable string with appropriate units.
 * Automatically selects the most appropriate unit (B, KB, MB, GB, TB) based on size.
 * @param bytes - The size in bytes to format
 * @returns Human-readable size string with one decimal place (e.g., "1.5 GB", "256.0 MB", "0 B")
 */

export const formatDiskSize = (bytes: number): string => {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)

  const size = bytes / Math.pow(1024, i)
  return `${parseFloat(size.toFixed(1))} ${units[i]}`
}

export const formatDateTime = (value: unknown) => {
  if (!value) return '-'
  const d = value instanceof Date ? value : new Date(String(value))
  if (Number.isNaN(d.getTime())) return '-'
  return d.toLocaleString('en-GB', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  })
}
