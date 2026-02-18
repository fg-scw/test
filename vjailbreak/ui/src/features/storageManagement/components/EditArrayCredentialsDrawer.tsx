import { useState, useEffect } from 'react'
import { Alert, CircularProgress } from '@mui/material'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import ErrorIcon from '@mui/icons-material/Error'
import { useForm } from 'react-hook-form'
import { useQueryClient } from '@tanstack/react-query'
import { ARRAY_CREDS_QUERY_KEY } from 'src/hooks/api/useArrayCredentialsQuery'
import { ArrayCreds } from 'src/api/array-creds/model'
import {
  updateArrayCredsWithSecret,
  getArrayCredentials,
  patchArrayCredentials
} from 'src/api/array-creds/arrayCreds'
import { createArrayCredsSecret, deleteSecret } from 'src/api/secrets/secrets'
import { useErrorHandler } from 'src/hooks/useErrorHandler'
import { ConfirmationDialog } from 'src/components/dialogs'
import { VJAILBREAK_DEFAULT_NAMESPACE } from 'src/api/constants'
import { DesignSystemForm } from 'src/shared/components/forms'
import { ActionButton, DrawerFooter, DrawerHeader, DrawerShell } from 'src/components/design-system'
import ArrayCredentialsFormFields from './ArrayCredentialsFormFields'

interface EditArrayCredentialsDrawerProps {
  open: boolean
  onClose: () => void
  credential: ArrayCreds
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

export default function EditArrayCredentialsDrawer({
  open,
  onClose,
  credential
}: EditArrayCredentialsDrawerProps) {
  const { reportError } = useErrorHandler({ component: 'EditArrayCredentialsDrawer' })
  const queryClient = useQueryClient()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isValidating, setIsValidating] = useState(false)
  const [validationStatus, setValidationStatus] = useState<ValidationStatus>('idle')
  const [validationMessage, setValidationMessage] = useState<string>('')
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [showDiscardDialog, setShowDiscardDialog] = useState(false)

  const isAutoDiscovered = credential.spec?.autoDiscovered

  const form = useForm<FormData>({
    defaultValues: {
      name: credential.metadata.name,
      vendorType: credential.spec?.vendorType || 'pure',
      volumeType: credential.spec?.openstackMapping?.volumeType || '',
      backendName: credential.spec?.openstackMapping?.cinderBackendName || '',
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

  useEffect(() => {
    reset({
      name: credential.metadata.name,
      vendorType: credential.spec?.vendorType || 'pure',
      volumeType: credential.spec?.openstackMapping?.volumeType || '',
      backendName: credential.spec?.openstackMapping?.cinderBackendName || '',
      managementEndpoint: '',
      username: '',
      password: '',
      skipSslVerification: false
    })
  }, [credential, reset])

  const handleClose = () => {
    if (isValidating) return // Don't allow closing while validating

    // Check if form has been touched (has any values)
    const formValues = getValues()
    const hasChanges = formValues.managementEndpoint || formValues.username || formValues.password

    if (hasChanges && !isSubmitting) {
      setShowDiscardDialog(true)
      return
    }

    setSubmitError(null)
    setValidationStatus('idle')
    setValidationMessage('')
    onClose()
  }

  const handleDiscardChanges = async () => {
    setSubmitError(null)
    setValidationStatus('idle')
    setValidationMessage('')
    onClose()
  }

  const handleCancelDiscard = () => {
    setShowDiscardDialog(false)
  }

  // Poll for validation status after updating credentials
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

  const onSubmit = async (data: FormData) => {
    setIsSubmitting(true)
    setSubmitError(null)
    setValidationStatus('idle')
    setValidationMessage('')

    try {
      const namespace = credential.metadata.namespace || VJAILBREAK_DEFAULT_NAMESPACE
      const secretName = `${data.name}-array-secret`

      // Create or update the secret if credentials are provided
      if (data.managementEndpoint && data.username && data.password) {
        await createArrayCredsSecret(
          secretName,
          {
            ARRAY_HOSTNAME: data.managementEndpoint,
            ARRAY_USERNAME: data.username,
            ARRAY_PASSWORD: data.password,
            ARRAY_SKIP_SSL_VERIFICATION: data.skipSslVerification
          },
          namespace
        )
      }

      // Update the ArrayCreds resource
      await updateArrayCredsWithSecret(
        data.name,
        secretName,
        data.vendorType,
        {
          volumeType: data.volumeType,
          cinderBackendName: data.backendName
        },
        namespace
      )

      // Start validation polling if credentials were provided
      if (data.managementEndpoint && data.username && data.password) {
        setIsSubmitting(false)
        setIsValidating(true)
        setValidationStatus('validating')
        setValidationMessage('Validating credentials with storage array...')

        const validationResult = await pollForValidation(data.name)
        setIsValidating(false)

        if (!validationResult.success) {
          // Validation failed - delete the secret and clear secretRef to allow retry with same name
          setValidationStatus('failed')
          setValidationMessage(validationResult.message || 'Validation failed')

          try {
            const secretName = `${data.name}-array-secret`
            await deleteSecret(secretName, namespace)
            console.log(`Deleted invalid secret: ${secretName}`)

            // Clear the secretRef from the ArrayCreds spec
            await patchArrayCredentials(
              data.name,
              {
                spec: {
                  secretRef: {
                    name: ''
                  }
                }
              } as any,
              namespace
            )
            console.log(`Cleared secretRef for: ${data.name}`)
          } catch (deleteErr) {
            console.error('Failed to delete invalid secret or clear secretRef:', deleteErr)
            // Continue anyway - user can still see the validation error
          }

          queryClient.invalidateQueries({ queryKey: ARRAY_CREDS_QUERY_KEY })
          return // Stay in the form
        }

        // Validation succeeded
        setValidationStatus('success')
        setValidationMessage('Credentials validated successfully!')
        queryClient.invalidateQueries({ queryKey: ARRAY_CREDS_QUERY_KEY })

        // Wait a moment to show success message before closing
        await new Promise((resolve) => setTimeout(resolve, 1500))
      } else {
        queryClient.invalidateQueries({ queryKey: ARRAY_CREDS_QUERY_KEY })
      }

      setValidationStatus('idle')
      setValidationMessage('')

      onClose()
    } catch (error: any) {
      console.error('Error updating array credentials:', error)
      // Reset validating states first to ensure form doesn't remain in validating state
      setIsValidating(false)
      setValidationStatus('idle')
      setValidationMessage('')
      // Then set the error message
      const errorMessage =
        error?.response?.data?.message || error?.message || 'Failed to update array credentials'
      setSubmitError(errorMessage)
      reportError(error, {
        context: 'edit-array-credentials',
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
          title="Edit Array Credentials"
          onClose={handleClose}
          data-testid="edit-array-credentials-drawer-header"
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
            form="edit-array-credentials-form"
            loading={isSubmitting || isValidating}
            disabled={validationStatus === 'success'}
          >
            {isSubmitting
              ? 'Updating...'
              : isValidating
                ? 'Validating...'
                : validationStatus === 'success'
                  ? 'Success'
                  : 'Update'}
          </ActionButton>
        </DrawerFooter>
      }
      data-testid="edit-array-credentials-drawer"
    >
      <DesignSystemForm
        id="edit-array-credentials-form"
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

        <ArrayCredentialsFormFields
          mode="edit"
          errors={errors}
          isAutoDiscovered={isAutoDiscovered}
        />
      </DesignSystemForm>

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
