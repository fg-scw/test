import { GridColDef, GridToolbarContainer, GridRowSelectionModel } from '@mui/x-data-grid'
import { Button, Typography, Box, IconButton, Tooltip, Chip, Alert, Snackbar } from '@mui/material'
import DeleteIcon from '@mui/icons-material/DeleteOutlined'
import EditIcon from '@mui/icons-material/Edit'
import WarningIcon from '@mui/icons-material/Warning'
import AddIcon from '@mui/icons-material/Add'
import SdStorageIcon from '@mui/icons-material/SdStorage'
import KeyIcon from '@mui/icons-material/Key'
import { CustomSearchToolbar } from 'src/components/grid'
import { CommonDataGrid } from 'src/components/grid'
import { useState, useCallback, useEffect, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import {
  useArrayCredentialsQuery,
  ARRAY_CREDS_QUERY_KEY
} from 'src/hooks/api/useArrayCredentialsQuery'
import { ArrayCreds, ARRAY_VENDOR_TYPES } from 'src/api/array-creds/model'
import { ConfirmationDialog } from 'src/components/dialogs'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { deleteArrayCredsWithSecretFlow } from 'src/api/helpers'
import { useErrorHandler } from 'src/hooks/useErrorHandler'
import { getSecret, upsertSecret } from 'src/api/secrets/secrets'
import { ActionButton, DrawerFooter, DrawerHeader, DrawerShell } from 'src/components/design-system'
import { DesignSystemForm, RHFTextField } from 'src/shared/components/forms'
import AddArrayCredentialsDrawer from './AddArrayCredentialsDrawer'
import EditArrayCredentialsDrawer from './EditArrayCredentialsDrawer'

interface ArrayCredentialRow {
  id: string
  name: string
  vendor: string
  vendorLabel: string
  volumeType: string
  backendName: string
  source: string
  credentialsStatus: string
  credObject: ArrayCreds
}

const getCredentialsStatusColor = (status: string): 'success' | 'warning' | 'default' => {
  if (!status) return 'default'
  const normalizedStatus = status.toLowerCase()
  if (normalizedStatus === 'configured' || normalizedStatus === 'succeeded') {
    return 'success'
  }
  if (normalizedStatus === 'pending' || normalizedStatus === 'validating') {
    return 'warning'
  }
  return 'default'
}

const getSourceColor = (source: string): 'info' | 'default' => {
  if (source === 'Auto-discovered') return 'info'
  return 'default'
}

const getVendorLabel = (vendorType: string): string => {
  const vendor = ARRAY_VENDOR_TYPES.find((v) => v.value === vendorType)
  return vendor?.label || vendorType || 'Unknown'
}

const getColumns = (
  onEditClick: (row: ArrayCredentialRow) => void,
  onDeleteClick: (row: ArrayCredentialRow) => void
): GridColDef[] => [
  {
    field: 'name',
    headerName: 'Name',
    flex: 1.5,
    minWidth: 200
  },
  {
    field: 'vendorLabel',
    headerName: 'Vendor',
    flex: 1,
    minWidth: 120,
    renderCell: (params) => (
      <Chip
        label={params.value || 'unsupported'}
        variant="outlined"
        size="small"
        sx={{ borderRadius: '4px' }}
      />
    )
  },
  {
    field: 'volumeType',
    headerName: 'Volume Type',
    flex: 1,
    minWidth: 120
  },
  {
    field: 'backendName',
    headerName: 'Backend Name',
    flex: 1,
    minWidth: 120
  },
  {
    field: 'source',
    headerName: 'Source',
    flex: 1,
    minWidth: 120,
    renderCell: (params) => (
      <Chip
        label={params.value}
        variant="outlined"
        color={getSourceColor(params.value)}
        size="small"
        sx={{ borderRadius: '4px' }}
      />
    )
  },
  {
    field: 'credentialsStatus',
    headerName: 'Credentials',
    flex: 1,
    minWidth: 120,
    renderCell: (params) => (
      <Chip
        label={params.value || 'Pending'}
        variant="outlined"
        color={getCredentialsStatusColor(params.value)}
        size="small"
        sx={{ borderRadius: '4px' }}
      />
    )
  },
  {
    field: 'actions',
    headerName: 'Actions',
    width: 100,
    sortable: false,
    renderCell: (params) => (
      <Box>
        <Tooltip title="Edit credentials">
          <IconButton
            onClick={(e) => {
              e.stopPropagation()
              onEditClick(params.row)
            }}
            aria-label="edit credential"
            size="small"
          >
            <EditIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        <Tooltip title="Delete">
          <IconButton
            onClick={(e) => {
              e.stopPropagation()
              onDeleteClick(params.row)
            }}
            aria-label="delete credential"
            size="small"
          >
            <DeleteIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      </Box>
    )
  }
]

interface CustomToolbarProps {
  onRefresh: () => void
  onAddCredential: () => void
  selectedCount: number
  onDeleteSelected: () => void
}

const CustomToolbar = ({
  onRefresh,
  onAddCredential,
  selectedCount,
  onDeleteSelected
}: CustomToolbarProps) => {
  return (
    <GridToolbarContainer
      sx={{
        p: 2,
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center'
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <SdStorageIcon />
        <Typography variant="h6" component="h2">
          Storage Management
        </Typography>
      </Box>
      <Box sx={{ display: 'flex', gap: 2 }}>
        <Button
          variant="outlined"
          color="primary"
          startIcon={<AddIcon />}
          onClick={onAddCredential}
          sx={{ height: 40 }}
        >
          ADD ARRAY CREDENTIALS
        </Button>
        {selectedCount > 0 && (
          <Button
            variant="outlined"
            color="error"
            startIcon={<DeleteIcon />}
            onClick={onDeleteSelected}
            sx={{ height: 40 }}
          >
            Delete Selected ({selectedCount})
          </Button>
        )}
        <CustomSearchToolbar placeholder="Search by Name" onRefresh={onRefresh} />
      </Box>
    </GridToolbarContainer>
  )
}

export default function StorageArrayTable() {
  const { reportError } = useErrorHandler({ component: 'StorageArrayTable' })
  const queryClient = useQueryClient()

  const [esxiKeyDrawerOpen, setEsxiKeyDrawerOpen] = useState(false)
  const [esxiKeyError, setEsxiKeyError] = useState<string | null>(null)
  const [esxiKeyToastOpen, setEsxiKeyToastOpen] = useState(false)
  const [esxiKeyToastMessage, setEsxiKeyToastMessage] = useState<string>('')
  const [esxiKeyToastSeverity, setEsxiKeyToastSeverity] = useState<'success' | 'error'>('success')

  type EsxiKeyFormData = {
    sshPrivateKey: string
  }

  const esxiKeyForm = useForm<EsxiKeyFormData>({
    defaultValues: {
      sshPrivateKey: ''
    }
  })

  const {
    reset: resetEsxiKeyForm,
    setValue: setEsxiKeyValue,
    getValues: getEsxiKeyValues
  } = esxiKeyForm

  const {
    data: esxiKeySecret,
    isLoading: isEsxiKeyLoading,
    refetch: refetchEsxiKey
  } = useQuery({
    queryKey: ['secret', 'migration-system', 'esxi-ssh-key'],
    queryFn: async () => {
      try {
        return await getSecret('esxi-ssh-key', 'migration-system')
      } catch (error: any) {
        if (error?.response?.status === 404) {
          return null
        }
        throw error
      }
    },
    retry: false
  })

  const { mutateAsync: saveEsxiKey, isPending: isSavingEsxiKey } = useMutation({
    mutationFn: async (keyContent: string) => {
      return upsertSecret('esxi-ssh-key', { 'ssh-privatekey': keyContent }, 'migration-system')
    },
    onSuccess: () => {
      refetchEsxiKey()
    }
  })

  const isEsxiKeyConfigured = !!esxiKeySecret?.metadata?.name

  const {
    data: arrayCredentials,
    isLoading,
    refetch
  } = useArrayCredentialsQuery(undefined, {
    staleTime: 0,
    refetchOnMount: true
  })

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [selectedForDeletion, setSelectedForDeletion] = useState<ArrayCredentialRow | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [addDrawerOpen, setAddDrawerOpen] = useState(false)
  const [editDrawerOpen, setEditDrawerOpen] = useState(false)
  const [selectedForEdit, setSelectedForEdit] = useState<ArrayCredentialRow | null>(null)
  const [rowSelectionModel, setRowSelectionModel] = useState<GridRowSelectionModel>([])
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = useState(false)

  useEffect(() => {
    refetch()
  }, [refetch])

  const rows: ArrayCredentialRow[] =
    arrayCredentials?.map((cred: ArrayCreds) => {
      const hasSecret = !!cred.spec?.secretRef?.name
      const credentialsStatus = hasSecret ? 'Configured' : 'Pending'
      const source = cred.spec?.autoDiscovered ? 'Auto-discovered' : 'Manual'

      return {
        id: cred.metadata.name,
        name: cred.metadata.name,
        vendor: cred.spec?.vendorType || '',
        vendorLabel: getVendorLabel(cred.spec?.vendorType),
        volumeType: cred.spec?.openstackMapping?.volumeType || '',
        backendName: cred.spec?.openstackMapping?.cinderBackendName || '',
        source,
        credentialsStatus,
        credObject: cred
      }
    }) || []

  const handleRefresh = useCallback(() => {
    refetch()
    refetchEsxiKey()
  }, [refetch, refetchEsxiKey])

  const validateOpenSshPrivateKey = (value: string): string | null => {
    const trimmed = value.trim()
    if (!trimmed) return 'SSH private key is required'
    if (/^ssh-privatekey\s*:/m.test(trimmed)) {
      return 'Paste only the key content (do not include "ssh-privatekey:")'
    }
    const hasBegin = /-----BEGIN OPENSSH PRIVATE KEY-----/.test(trimmed)
    const hasEnd = /-----END OPENSSH PRIVATE KEY-----/.test(trimmed)
    if (!hasBegin || !hasEnd) {
      return 'Invalid key format. Expected OpenSSH private key (-----BEGIN OPENSSH PRIVATE KEY-----)'
    }
    return null
  }

  const handleOpenEsxiKeyDrawer = () => {
    setEsxiKeyError(null)
    const existing = (esxiKeySecret as any)?.data?.['ssh-privatekey']
    resetEsxiKeyForm({ sshPrivateKey: typeof existing === 'string' ? existing : '' })
    setEsxiKeyDrawerOpen(true)
  }

  const handleCloseEsxiKeyDrawer = () => {
    if (isSavingEsxiKey) return
    const formValues = getEsxiKeyValues()
    const hasChanges = Boolean(formValues.sshPrivateKey)

    if (hasChanges) {
      resetEsxiKeyForm({ sshPrivateKey: '' })
    }

    setEsxiKeyDrawerOpen(false)
    setEsxiKeyError(null)
  }

  const handleCloseEsxiKeyToast = () => {
    setEsxiKeyToastOpen(false)
  }

  const handleEsxiKeyFileChange = async (file: File | null) => {
    if (!file) return
    const MAX_KEY_FILE_SIZE = 1024 * 1024 // 1 MB limit for SSH key files
    if (file.size > MAX_KEY_FILE_SIZE) {
      setEsxiKeyError('File is too large. SSH private key files should be less than 1 MB.')
      return
    }
    try {
      const text = await file.text()
      setEsxiKeyValue('sshPrivateKey', text, { shouldDirty: true })
      setEsxiKeyError(null)
    } catch (error) {
      setEsxiKeyError('Failed to read file')
    }
  }

  const onSubmitEsxiKey = async (data: EsxiKeyFormData) => {
    const validationError = validateOpenSshPrivateKey(data.sshPrivateKey)
    if (validationError) {
      setEsxiKeyError(validationError)
      return
    }

    try {
      setEsxiKeyError(null)
      await saveEsxiKey(data.sshPrivateKey.trim())
      setEsxiKeyToastSeverity('success')
      setEsxiKeyToastMessage('ESXi SSH key saved successfully.')
      setEsxiKeyToastOpen(true)
      setEsxiKeyDrawerOpen(false)
    } catch (error: any) {
      const errorMessage =
        error?.response?.data?.message || error?.message || 'Failed to save ESXi SSH key'
      setEsxiKeyError(errorMessage)
      setEsxiKeyToastSeverity('error')
      setEsxiKeyToastMessage(errorMessage)
      setEsxiKeyToastOpen(true)
      reportError(error, { context: 'save-esxi-ssh-key' })
    }
  }

  const handleEditClick = (row: ArrayCredentialRow) => {
    setSelectedForEdit(row)
    setEditDrawerOpen(true)
  }

  const handleDeleteClick = (row: ArrayCredentialRow) => {
    setSelectedForDeletion(row)
    setDeleteDialogOpen(true)
  }

  const handleDeleteClose = () => {
    setDeleteDialogOpen(false)
    setSelectedForDeletion(null)
    setDeleteError(null)
  }

  const handleConfirmDelete = async () => {
    if (!selectedForDeletion) return

    setDeleting(true)
    try {
      await deleteArrayCredsWithSecretFlow(selectedForDeletion.name)
      queryClient.invalidateQueries({ queryKey: ARRAY_CREDS_QUERY_KEY })
      handleDeleteClose()
    } catch (error) {
      console.error('Error deleting array credential:', error)
      reportError(error as Error, {
        context: 'array-credentials-deletion',
        metadata: {
          credentialName: selectedForDeletion.name
        }
      })
      setDeleteError(error instanceof Error ? error.message : 'Unknown error occurred')
    } finally {
      setDeleting(false)
    }
  }

  const handleBulkDeleteClick = () => {
    setBulkDeleteDialogOpen(true)
  }

  const handleBulkDeleteClose = () => {
    setBulkDeleteDialogOpen(false)
    setDeleteError(null)
  }

  const handleConfirmBulkDelete = async () => {
    setDeleting(true)
    try {
      await Promise.all(rowSelectionModel.map((id) => deleteArrayCredsWithSecretFlow(id as string)))
      queryClient.invalidateQueries({ queryKey: ARRAY_CREDS_QUERY_KEY })
      setRowSelectionModel([])
      handleBulkDeleteClose()
    } catch (error) {
      console.error('Error deleting array credentials:', error)
      reportError(error as Error, {
        context: 'array-credentials-bulk-deletion'
      })
      setDeleteError(error instanceof Error ? error.message : 'Unknown error occurred')
    } finally {
      setDeleting(false)
    }
  }

  const selectedItems = useMemo(() => {
    return rows.filter((row) => rowSelectionModel.includes(row.id))
  }, [rows, rowSelectionModel])

  const handleOpenAddDrawer = () => {
    setAddDrawerOpen(true)
  }

  const handleCloseAddDrawer = () => {
    setAddDrawerOpen(false)
    refetch()
  }

  const handleCloseEditDrawer = () => {
    setEditDrawerOpen(false)
    setSelectedForEdit(null)
    refetch()
  }

  const getCustomErrorMessage = useCallback((error: Error | string) => {
    const baseMessage = 'Failed to delete array credential'
    if (error instanceof Error) {
      return `${baseMessage}: ${error.message}`
    }
    return `${baseMessage}: ${error}`
  }, [])

  const tableColumns = getColumns(handleEditClick, handleDeleteClick)

  return (
    <div style={{ height: '100%', width: '100%', overflow: 'hidden' }}>
      <Box
        sx={{
          mx: 2,
          mt: 2,
          mb: 1.5,
          px: 2,
          py: 1.5,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 2,
          border: '1px solid',
          borderColor: 'divider',
          borderRadius: 1,
          backgroundColor: 'background.paper'
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1.25 }}>
          <Box
            sx={{
              mt: '2px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: 32,
              height: 32,
              borderRadius: 1,
              backgroundColor: 'action.hover'
            }}
          >
            <KeyIcon fontSize="small" />
          </Box>
          <Box sx={{ display: 'flex', flexDirection: 'column' }}>
            <Typography variant="subtitle1" fontWeight={600} sx={{ lineHeight: 1.2 }}>
              ESXi SSH Key
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Configure the ESXi SSH private key secret (migration-system/esxi-ssh-key)
            </Typography>
          </Box>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Chip
            label={
              isEsxiKeyLoading ? 'Loading' : isEsxiKeyConfigured ? 'Configured' : 'Not configured'
            }
            color={isEsxiKeyConfigured ? 'success' : 'default'}
            variant="outlined"
            size="small"
            sx={{ borderRadius: '4px' }}
          />
          <Button
            variant="outlined"
            color="primary"
            onClick={handleOpenEsxiKeyDrawer}
            disabled={isEsxiKeyLoading}
            size="small"
            sx={{ height: 30, minWidth: 72 }}
          >
            {isEsxiKeyConfigured ? 'Edit' : 'Configure'}
          </Button>
        </Box>
      </Box>

      <CommonDataGrid
        rows={rows}
        columns={tableColumns}
        checkboxSelection
        disableRowSelectionOnClick
        rowSelectionModel={rowSelectionModel}
        onRowSelectionModelChange={setRowSelectionModel}
        initialState={{
          sorting: {
            sortModel: [{ field: 'name', sort: 'asc' }]
          },
          pagination: {
            paginationModel: {
              pageSize: 25
            }
          }
        }}
        slots={{
          toolbar: () => (
            <CustomToolbar
              onRefresh={handleRefresh}
              onAddCredential={handleOpenAddDrawer}
              selectedCount={rowSelectionModel.length}
              onDeleteSelected={handleBulkDeleteClick}
            />
          )
        }}
        pageSizeOptions={[10, 25, 50, 100]}
        loading={isLoading || deleting}
        emptyMessage="No storage array credentials available"
        sx={{
          '& .MuiDataGrid-main': {
            overflow: 'auto'
          },
          '& .MuiDataGrid-cell:focus': {
            outline: 'none'
          }
        }}
      />

      <ConfirmationDialog
        open={deleteDialogOpen}
        onClose={handleDeleteClose}
        title="Confirm Delete"
        icon={<WarningIcon color="warning" />}
        message={`Are you sure you want to delete storage array credential "${selectedForDeletion?.name}"?`}
        actionLabel="Delete"
        actionColor="error"
        actionVariant="outlined"
        onConfirm={handleConfirmDelete}
        customErrorMessage={getCustomErrorMessage}
        errorMessage={deleteError}
        onErrorChange={setDeleteError}
      />

      <ConfirmationDialog
        open={bulkDeleteDialogOpen}
        onClose={handleBulkDeleteClose}
        title="Confirm Delete"
        icon={<WarningIcon color="warning" />}
        message={`Are you sure you want to delete ${rowSelectionModel.length} storage array credential${rowSelectionModel.length > 1 ? 's' : ''}?`}
        items={selectedItems.map((item) => ({ id: item.id, name: item.name }))}
        actionLabel="Delete"
        actionColor="error"
        actionVariant="outlined"
        onConfirm={handleConfirmBulkDelete}
        customErrorMessage={getCustomErrorMessage}
        errorMessage={deleteError}
        onErrorChange={setDeleteError}
      />

      {addDrawerOpen && (
        <AddArrayCredentialsDrawer open={addDrawerOpen} onClose={handleCloseAddDrawer} />
      )}

      {editDrawerOpen && selectedForEdit && (
        <EditArrayCredentialsDrawer
          open={editDrawerOpen}
          onClose={handleCloseEditDrawer}
          credential={selectedForEdit.credObject}
        />
      )}

      <DrawerShell
        open={esxiKeyDrawerOpen}
        onClose={handleCloseEsxiKeyDrawer}
        requireCloseConfirmation={false}
        width={820}
        header={
          <DrawerHeader
            title={
              isEsxiKeyConfigured ? 'Edit ESXi SSH Private Key' : 'Configure ESXi SSH Private Key'
            }
            subtitle="Paste or upload an OpenSSH private key."
            onClose={handleCloseEsxiKeyDrawer}
          />
        }
        footer={
          <DrawerFooter>
            <ActionButton
              tone="secondary"
              onClick={handleCloseEsxiKeyDrawer}
              disabled={isSavingEsxiKey}
            >
              Cancel
            </ActionButton>
            <ActionButton
              tone="primary"
              type="submit"
              form="esxi-ssh-key-form"
              loading={isSavingEsxiKey}
            >
              Save
            </ActionButton>
          </DrawerFooter>
        }
        data-testid="esxi-ssh-key-drawer"
      >
        <DesignSystemForm
          id="esxi-ssh-key-form"
          form={esxiKeyForm}
          onSubmit={onSubmitEsxiKey}
          keyboardSubmitProps={{
            open: esxiKeyDrawerOpen,
            onClose: handleCloseEsxiKeyDrawer,
            isSubmitDisabled: isSavingEsxiKey
          }}
          sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}
        >
          {esxiKeyError && (
            <Alert severity="error" sx={{ mb: 1 }} onClose={() => setEsxiKeyError(null)}>
              {esxiKeyError}
            </Alert>
          )}

          <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
            <ActionButton tone="secondary" component="label" disabled={isSavingEsxiKey}>
              Upload key file
              <input
                type="file"
                hidden
                onChange={(e) => handleEsxiKeyFileChange(e.target.files?.[0] ?? null)}
              />
            </ActionButton>
            <Typography variant="body2" color="text.secondary">
              Only the key content will be stored (do not include a field name).
            </Typography>
          </Box>

          <RHFTextField
            name="sshPrivateKey"
            label="SSH Private Key"
            placeholder="-----BEGIN OPENSSH PRIVATE KEY-----\n...\n-----END OPENSSH PRIVATE KEY-----"
            multiline
            minRows={12}
            disabled={isSavingEsxiKey}
            rules={{
              validate: (val: string) => validateOpenSshPrivateKey(val) || true
            }}
            onValueChange={() => setEsxiKeyError(null)}
          />
        </DesignSystemForm>
      </DrawerShell>

      <Snackbar
        open={esxiKeyToastOpen}
        autoHideDuration={6000}
        onClose={handleCloseEsxiKeyToast}
        anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
      >
        <Alert
          onClose={handleCloseEsxiKeyToast}
          severity={esxiKeyToastSeverity}
          variant="filled"
          sx={{ width: '100%' }}
        >
          {esxiKeyToastMessage}
        </Alert>
      </Snackbar>
    </div>
  )
}
