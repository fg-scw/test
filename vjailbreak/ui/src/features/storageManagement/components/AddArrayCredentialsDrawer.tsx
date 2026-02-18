import { useState } from 'react'
import { Alert, CircularProgress } from '@mui/material'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import ErrorIcon from '@mui/icons-material/Error'
import { useForm } from 'react-hook-form'
import { useQueryClient } from '@tanstack/react-query'
import { ARRAY_CREDS_QUERY_KEY } from 'src/hooks/api/useArrayCredentialsQuery'
import { createArrayCredsWithSecretFlow, deleteArrayCredsWithSecretFlow } from 'src/api/helpers'
import { getArrayCredentials } from 'src/api/array-creds/arrayCreds'
import { useErrorHandler } from 'src/hooks/useErrorHandler'
import { ConfirmationDialog } from 'src/components/dialogs'
import { DesignSystemForm } from 'src/shared/components/forms'
import { ActionButton, DrawerFooter, DrawerHeader, DrawerShell } from 'src/components/design-system'
import ArrayCredentialsFormFields from './ArrayCredentialsFormFields'

interface AddArrayCredentialsDrawerProps {
  open: boolean
  onClose: () => void
}

interface FormData {
  name: string
  vendorType: string
  volumeType: string
  backendName: string
  managementEndpoint: string
  username: string
  password: string
  skipSslVerification: boolean
}

type ValidationStatus = 'idle' | 'validating' | 'success' | 'failed'

export default function AddArrayCredentialsDrawer({
  open,
  onClose
}: AddArrayCredentialsDrawerProps) {
  const { reportError } = useErrorHandler({ component: 'AddArrayCredentialsDrawer' })
  const queryClient = useQueryClient()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isValidating, setIsValidating] = useState(false)
  const [validationStatus, setValidationStatus] = useState<ValidationStatus>('idle')
  const [validationMessage, setValidationMessage] = useState<string>('')
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [showDiscardDialog, setShowDiscardDialog] = useState(false)
  const [createdCredentialName, setCreatedCredentialName] = useState<string | null>(null)

  const form = useForm<FormData>({
    defaultValues: {
      name: '',
      vendorType: 'pure',
      volumeType: '',
      backendName: '',
      managementEndpoint: '',
      username: '',
      password: '',
      skipSslVerification: false
    }
  })

  const {
    formState: { errors },
    reset,
    getValues
  } = form

  const handleClose = () => {
    if (isValidating) return // Don't allow closing while validating

    // Check if form has been touched (has any values)
    const formValues = getValues()
    const hasChanges =
      formValues.name || formValues.managementEndpoint || formValues.username || formValues.password

    if (hasChanges && !isSubmitting) {
      setShowDiscardDialog(true)
      return
    }

    // Cleanup any created credential if user is closing before validation completes
    if (createdCredentialName && validationStatus !== 'success') {
      deleteArrayCredsWithSecretFlow(createdCredentialName).catch((err) => {
        console.error(`Error deleting cancelled credential: ${createdCredentialName}`, err)
      })
    }

    setSubmitError(null)
    setValidationStatus('idle')
    setValidationMessage('')
    setCreatedCredentialName(null)
    onClose()
  }

  const handleDiscardChanges = async () => {
    // Cleanup any created credential when user discards changes
    if (createdCredentialName && validationStatus !== 'success') {
      deleteArrayCredsWithSecretFlow(createdCredentialName).catch((err) => {
        console.error(`Error deleting discarded credential: ${createdCredentialName}`, err)
      })
    }

    setSubmitError(null)
    setValidationStatus('idle')
    setValidationMessage('')
    setCreatedCredentialName(null)
    onClose()
  }

  const handleCancelDiscard = () => {
    setShowDiscardDialog(false)
  }

  // Poll for validation status after creating credentials
  const pollForValidation = async (
    name: string,
    maxAttempts = 30,
    intervalMs = 2000
  ): Promise<{ success: boolean; message?: string }> => {
    for (let attempt = 0; attempt < maxAttempts; attempt++) {
      await new Promise((resolve) => setTimeout(resolve, intervalMs))

      try {
        const updated = await getArrayCredentials(name)
        const status = updated.status?.arrayValidationStatus

        // Check if validation succeeded
        if (status === 'Succeeded') {
          return { success: true }
        }

        // Check if validation failed
        if (status && status !== 'AwaitingCredentials' && status !== 'Validating') {
          return {
            success: false,
            message:
              updated.status?.arrayValidationMessage || `Validation failed with status: ${status}`
          }
        }
      } catch (error) {
        console.error('Error polling for validation:', error)
      }
    }

    // Timeout - validation took too long
    return { success: false, message: 'Validation timed out after 60 seconds' }
  }

  // Cleanup failed creation by deleting the ArrayCreds resource
  const cleanupFailedCreation = async (name: string) => {
    try {
      console.log(`Starting cleanup for failed creation: ${name}`)
      await deleteArrayCredsWithSecretFlow(name)
      console.log(`Cleanup completed for: ${name}`)
      queryClient.invalidateQueries({ queryKey: ARRAY_CREDS_QUERY_KEY })
    } catch (error) {
      console.error('Error cleaning up failed creation:', error)
      throw error
    }
  }

  const onSubmit = async (data: FormData) => {
    setIsSubmitting(true)
    setSubmitError(null)
    setValidationStatus('idle')
    setValidationMessage('')

    try {
      await createArrayCredsWithSecretFlow(data.name, {
        ARRAY_HOSTNAME: data.managementEndpoint,
        ARRAY_USERNAME: data.username,
        ARRAY_PASSWORD: data.password,
        ARRAY_SKIP_SSL_VERIFICATION: data.skipSslVerification,
        VENDOR_TYPE: data.vendorType,
        OPENSTACK_MAPPING: {
          volumeType: data.volumeType,
          cinderBackendName: data.backendName
        }
      })

      // Track the created credential name for cleanup if needed
      setCreatedCredentialName(data.name)

      // Start validation polling
      setIsSubmitting(false)
      setIsValidating(true)
      setValidationStatus('validating')
      setValidationMessage('Validating credentials with storage array...')

      const validationResult = await pollForValidation(data.name)
      setIsValidating(false)

      if (!validationResult.success) {
        // Validation failed - show error and cleanup
        setValidationStatus('failed')
        setValidationMessage(validationResult.message || 'Validation failed')

        try {
          await cleanupFailedCreation(data.name)
          setCreatedCredentialName(null) // Clear after cleanup
        } catch (cleanupError) {
          console.error('Cleanup failed:', cleanupError)
          setValidationMessage(
            `${validationResult.message}. Warning: Failed to clean up - please delete manually.`
          )
        }
        return // Stay in the form
      }

      // Validation succeeded
      setValidationStatus('success')
      setValidationMessage('Credentials validated successfully!')
      queryClient.invalidateQueries({ queryKey: ARRAY_CREDS_QUERY_KEY })

      // Wait a moment to show success message before closing
      await new Promise((resolve) => setTimeout(resolve, 1500))

      reset()
      setValidationStatus('idle')
      setValidationMessage('')
      setCreatedCredentialName(null)
      onClose()
    } catch (error: any) {
      console.error('Error creating array credentials:', error)
      setIsValidating(false)
      setValidationStatus('idle')
      const errorMessage =
        error?.response?.data?.message || error?.message || 'Failed to create array credentials'
      setSubmitError(errorMessage)
      reportError(error, {
        context: 'add-array-credentials',
        metadata: { credentialName: data.name }
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <DrawerShell
      open={open}
      onClose={handleClose}
      requireCloseConfirmation={false}
      header={
        <DrawerHeader
          title="Add Array Credentials"
          onClose={handleClose}
          data-testid="add-array-credentials-drawer-header"
        />
      }
      footer={
        <DrawerFooter>
          <ActionButton
            tone="secondary"
            onClick={handleClose}
            disabled={isSubmitting || isValidating}
          >
            Cancel
          </ActionButton>
          <ActionButton
            type="submit"
            form="add-array-credentials-form"
            loading={isSubmitting || isValidating}
            disabled={validationStatus === 'success'}
          >
            {isSubmitting
              ? 'Creating...'
              : isValidating
                ? 'Validating...'
                : validationStatus === 'success'
                  ? 'Success'
                  : 'Save Credentials'}
          </ActionButton>
        </DrawerFooter>
      }
      data-testid="add-array-credentials-drawer"
    >
      <DesignSystemForm
        id="add-array-credentials-form"
        form={form}
        onSubmit={onSubmit}
        keyboardSubmitProps={{
          open,
          onClose: handleClose,
          isSubmitDisabled: isSubmitting || isValidating || validationStatus === 'success'
        }}
        sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}
      >
        {submitError && (
          <Alert severity="error" sx={{ mb: 2 }} onClose={() => setSubmitError(null)}>
            {submitError}
          </Alert>
        )}

        {validationStatus === 'validating' && (
          <Alert severity="info" icon={<CircularProgress size={20} />} sx={{ mb: 2 }}>
            {validationMessage}
          </Alert>
        )}

        {validationStatus === 'success' && (
          <Alert severity="success" icon={<CheckCircleIcon />} sx={{ mb: 2 }}>
            {validationMessage}
          </Alert>
        )}

        {validationStatus === 'failed' && (
          <Alert severity="error" icon={<ErrorIcon />} sx={{ mb: 2 }}>
            {validationMessage}
          </Alert>
        )}

        <ArrayCredentialsFormFields mode="add" errors={errors} />
      </DesignSystemForm>

      {/* Discard Changes Dialog */}
      <ConfirmationDialog
        open={showDiscardDialog}
        onClose={handleCancelDiscard}
        title="Discard Changes?"
        message="Are you sure you want to leave? Any unsaved changes will be lost."
        actionLabel="Leave"
        actionColor="warning"
        actionVariant="outlined"
        onConfirm={handleDiscardChanges}
      />
    </DrawerShell>
  )
}
