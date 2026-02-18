import { Box } from '@mui/material'
import { useState } from 'react'
import { Step } from 'src/shared/components/forms'
import { Row, SectionHeader } from 'src/components'
import { useVmwareCredentialsQuery } from 'src/hooks/api/useVmwareCredentialsQuery'
import { useOpenstackCredentialsQuery } from 'src/hooks/api/useOpenstackCredentialsQuery'
import {
  VmwareCredentialsForm,
  OpenstackCredentialsForm
} from 'src/features/credentials/components'

interface SourceAndDestinationEnvStepProps {
  onChange: (id: string) => (value: unknown) => void
  errors: { [fieldId: string]: string }
}

export default function SourceAndDestinationEnvStep({
  onChange,
  errors
}: SourceAndDestinationEnvStepProps) {
  const [selectedVmwareCred, setSelectedVmwareCred] = useState<string | null>(null)
  const [selectedOpenstackCred, setSelectedOpenstackCred] = useState<string | null>(null)

  const { data: vmwareCredsList = [], isLoading: loadingVmwareCreds } = useVmwareCredentialsQuery()
  const { data: openstackCredsList = [], isLoading: loadingOpenstackCreds } =
    useOpenstackCredentialsQuery()

  const handleVmwareCredSelect = async (credId: string | null) => {
    setSelectedVmwareCred(credId)
    if (credId) {
      const selectedCred = vmwareCredsList.find((cred) => cred.metadata.name === credId)
      if (selectedCred) {
        try {
          const mappedCreds = {
            existingCredName: selectedCred.metadata.name,
            secretRef: selectedCred.spec.secretRef,
            datacenter: selectedCred.spec.datacenter,
            hostName: selectedCred.spec.hostName
          }

          onChange('vmwareCreds')(mappedCreds)
        } catch (error) {
          console.error('Error processing VMware credential:', error)
        }
      }
    } else {
      // Clear the selection
      onChange('vmwareCreds')({})
    }
  }

  const handleOpenstackCredSelect = async (credId: string | null) => {
    setSelectedOpenstackCred(credId)
    if (credId) {
      const selectedCred = openstackCredsList.find((cred) => cred.metadata.name === credId)
      if (selectedCred) {
        try {
          // Use data directly from credential spec instead of fetching from secret
          const mappedCreds = {
            existingCredName: selectedCred.metadata.name,
            projectName: selectedCred.spec.projectName
          }

          onChange('openstackCreds')(mappedCreds)
        } catch (error) {
          console.error('Error processing OpenStack credential:', error)
        }
      }
    } else {
      // Clear the selection
      onChange('openstackCreds')({})
    }
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
      <Step stepNumber="1" label="Source and Destination Environments" />
      <Row gap={3} flexWrap="wrap" sx={{ ml: 6 }}>
        <Box sx={{ flex: 1, minWidth: 300 }}>
          <SectionHeader title="Source VMware" sx={{ mb: 1 }} />
          <VmwareCredentialsForm
            credentialsList={vmwareCredsList}
            loadingCredentials={loadingVmwareCreds}
            error={errors['vmwareCreds']}
            onCredentialSelect={handleVmwareCredSelect}
            selectedCredential={selectedVmwareCred}
            showCredentialSelector={true}
            showCredentialNameField={false}
          />
        </Box>

        <Box sx={{ flex: 1, minWidth: 300 }}>
          <SectionHeader title="Destination Platform" sx={{ mb: 1 }} />
          <OpenstackCredentialsForm
            credentialsList={openstackCredsList}
            loadingCredentials={loadingOpenstackCreds}
            error={errors['openstackCreds']}
            onCredentialSelect={handleOpenstackCredSelect}
            selectedCredential={selectedOpenstackCred}
            showCredentialSelector={true}
            showCredentialNameField={false}
          />
        </Box>
      </Row>
    </Box>
  )
}
