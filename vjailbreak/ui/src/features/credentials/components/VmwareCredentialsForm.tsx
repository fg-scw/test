import { Box, FormControl, FormHelperText } from '@mui/material'
import CredentialSelector from './CredentialSelector'

export interface VmwareCredential {
  metadata: {
    name: string
    namespace?: string
  }
  spec: {
    VCENTER_HOST?: string
    VCENTER_USERNAME?: string
    VCENTER_PASSWORD?: string
    secretRef?: {
      name: string
    }
  }
  status?: {
    vmwareValidationStatus?: string
    vmwareValidationMessage?: string
  }
}

export interface VmwareCredentialsFormProps {
  credentialsList?: VmwareCredential[]
  loadingCredentials?: boolean
  error?: string
  onCredentialSelect?: (credId: string | null) => void
  selectedCredential?: string | null
  showCredentialSelector?: boolean
  showCredentialNameField?: boolean
  fullWidth?: boolean
  size?: 'small' | 'medium'
}

export default function VmwareCredentialsForm({
  credentialsList = [],
  loadingCredentials = false,
  error,
  onCredentialSelect,
  selectedCredential = null,
  showCredentialSelector = true,
  size = 'small'
}: VmwareCredentialsFormProps) {
  const credentialOptions = credentialsList.map((cred) => ({
    label: cred.metadata.name,
    value: cred.metadata.name,
    metadata: cred.metadata,
    status: {
      validationStatus: cred.status?.vmwareValidationStatus,
      validationMessage: cred.status?.vmwareValidationMessage
    }
  }))

  return (
    <div>
      {showCredentialSelector && (
        <FormControl fullWidth error={!!error}>
          <CredentialSelector
            placeholder="Select VMware credentials"
            options={credentialOptions}
            value={selectedCredential}
            onChange={onCredentialSelect || (() => {})}
            loading={loadingCredentials}
            size={size}
            emptyMessage="No VMware credentials found. Please use the credential drawer to create new ones."
            showAddNewButton={false}
          />
          {error && (
            <Box sx={{ mt: 1 }}>
              <FormHelperText error>{error}</FormHelperText>
            </Box>
          )}
        </FormControl>
      )}
    </div>
  )
}
