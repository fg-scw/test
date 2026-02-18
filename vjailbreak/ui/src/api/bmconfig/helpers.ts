import { BMConfig } from './model'

export const isBMConfigValid = (bmconfig: BMConfig): boolean => {
  return (
    bmconfig?.status?.validationStatus === 'validated' ||
    bmconfig?.status?.validationStatus === 'valid'
  )
}

export const getBMConfigValidationMessage = (bmconfig: BMConfig): string | null => {
  if (!bmconfig?.status?.validationMessage) {
    return null
  }

  return bmconfig.status.validationMessage
}

export const formatBMConfigApiUrl = (url: string): string => {
  if (!url) return ''

  if (!url.startsWith('http://') && !url.startsWith('https://')) {
    return `http://${url}`
  }

  return url
}

export const getBMConfigDisplayName = (bmconfig: BMConfig): string => {
  if (!bmconfig) return ''

  const { name } = bmconfig.metadata
  return name || 'Unnamed BMConfig'
}
