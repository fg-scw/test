import { GridColDef, GridToolbarContainer, GridRowSelectionModel } from '@mui/x-data-grid'
import { Button, Typography, Box, IconButton, Tooltip, Chip, CircularProgress } from '@mui/material'
import DeleteIcon from '@mui/icons-material/DeleteOutlined'
import WarningIcon from '@mui/icons-material/Warning'
import AddIcon from '@mui/icons-material/Add'
import CredentialsIcon from '@mui/icons-material/VpnKey'
import SyncIcon from '@mui/icons-material/Sync'
import { CustomSearchToolbar } from 'src/components/grid'
import { CommonDataGrid } from 'src/components/grid'
import { useState, useCallback, useEffect, useRef } from 'react'
import { useVmwareCredentialsQuery } from 'src/hooks/api/useVmwareCredentialsQuery'
import { useOpenstackCredentialsQuery } from 'src/hooks/api/useOpenstackCredentialsQuery'
import { VmwareCredential } from './VmwareCredentialsForm'
import { OpenstackCredential } from './OpenstackCredentialsForm'
import { ConfirmationDialog } from 'src/components/dialogs'
import { useQueryClient, useMutation } from '@tanstack/react-query'
import { VMWARE_CREDS_QUERY_KEY } from 'src/hooks/api/useVmwareCredentialsQuery'
import { OPENSTACK_CREDS_QUERY_KEY } from 'src/hooks/api/useOpenstackCredentialsQuery'
import {
  deleteVMwareCredsWithSecretFlow,
  deleteOpenStackCredsWithSecretFlow,
  revalidateCredentials
} from 'src/api/helpers'
import VMwareCredentialsDrawer from './VMwareCredentialsDrawer'
import { useErrorHandler } from 'src/hooks/useErrorHandler'
import OpenstackCredentialsDrawer from './OpenstackCredentialsDrawer'
import { VJAILBREAK_DEFAULT_NAMESPACE } from 'src/api/constants'

interface CredentialItem {
  id: string
  name: string
  type: 'VMware' | 'OpenStack'
  status: string
  credObject: VmwareCredential | OpenstackCredential
}

const getStatusColor = (status: string): 'success' | 'error' | 'warning' | 'info' | 'default' => {
  if (!status) return 'default'
  const normalizedStatus = status.toLowerCase()
  if (normalizedStatus === 'succeeded') {
    return 'success'
  }
  if (normalizedStatus === 'failed') {
    return 'error'
  }
  if (normalizedStatus === 'validating') {
    return 'warning'
  }
  return 'default'
}

const getColumns = (
  onDeleteClick: (id: string, type: 'VMware' | 'OpenStack') => void,
  onRevalidateClick: (row: CredentialItem) => void,
  revalidatingId: string | null
): GridColDef[] => [
  {
    field: 'name',
    headerName: 'Name',
    flex: 1
  },
  {
    field: 'type',
    headerName: 'Type',
    flex: 1,
    renderCell: (params) => (
      <Chip
        label={params.value === 'OpenStack' ? 'PCD' : params.value}
        color={params.value === 'VMware' ? 'primary' : 'secondary'}
        variant="outlined"
        size="small"
      />
    )
  },
  {
    field: 'status',
    headerName: 'Status',
    flex: 1,
    renderCell: (params) => (
      <Chip
        label={params.value || 'Unknown'}
        variant="outlined"
        color={getStatusColor(params.value)}
        size="small"
        icon={
          params.value === 'Validating' ? (
            <CircularProgress size={16} sx={{ marginRight: '5px' }} />
          ) : undefined
        }
      />
    )
  },
  {
    field: 'actions',
    headerName: 'Actions',
    flex: 1,
    width: 100,
    sortable: false,
    renderCell: (params) => {
      const isRowLoading = revalidatingId === params.row.id

      return (
        <Box>
          <Tooltip title="Revalidate Credentials">
            <span>
              <IconButton
                onClick={(e) => {
                  e.stopPropagation()
                  onRevalidateClick(params.row)
                }}
                disabled={isRowLoading}
                aria-label="revalidate credential"
              >
                {isRowLoading ? <CircularProgress size={20} /> : <SyncIcon />}
              </IconButton>
            </span>
          </Tooltip>

          <Tooltip title="Delete credential">
            <span>
              <IconButton
                onClick={(e) => {
                  e.stopPropagation()
                  onDeleteClick(params.row.id, params.row.type)
                }}
                aria-label="delete credential"
                disabled={isRowLoading}
              >
                <DeleteIcon />
              </IconButton>
            </span>
          </Tooltip>
        </Box>
      )
    }
  }
]

interface CustomToolbarProps {
  numSelected: number
  onDeleteSelected: () => void
  loading: boolean
  onRefresh: () => void
  onAddVMwareCredential: () => void
  onAddOpenstackCredential: () => void
}

const CustomToolbar = ({
  numSelected,
  onDeleteSelected,
  loading,
  onRefresh,
  onAddVMwareCredential,
  onAddOpenstackCredential
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
        <CredentialsIcon />
        <Typography variant="h6" component="h2">
          Credentials
        </Typography>
      </Box>
      <Box sx={{ display: 'flex', gap: 2 }}>
        <Button
          variant="outlined"
          color="primary"
          startIcon={<AddIcon />}
          onClick={onAddVMwareCredential}
          sx={{ height: 40 }}
          data-tour="add-vmware-creds"
        >
          Add VMware Credentials
        </Button>
        <Button
          variant="outlined"
          color="primary"
          startIcon={<AddIcon />}
          onClick={onAddOpenstackCredential}
          sx={{ height: 40 }}
          data-tour="add-pcd-creds"
        >
          Add PCD Credentials
        </Button>
        {numSelected > 0 && (
          <Button
            variant="outlined"
            color="error"
            startIcon={<DeleteIcon />}
            onClick={onDeleteSelected}
            disabled={loading}
            sx={{ height: 40 }}
          >
            Delete Selected ({numSelected})
          </Button>
        )}
        <CustomSearchToolbar placeholder="Search by Name or Type" onRefresh={onRefresh} />
      </Box>
    </GridToolbarContainer>
  )
}

export default function CredentialsTable() {
  const { reportError } = useErrorHandler({ component: 'CredentialsTable' })
  const queryClient = useQueryClient()
  const [revalidatingId, setRevalidatingId] = useState<string | null>(null)

  const {
    data: vmwareCredentials,
    isLoading: loadingVmware,
    refetch: refetchVmware
  } = useVmwareCredentialsQuery(undefined, {
    staleTime: 0,
    refetchOnMount: true,
    refetchInterval: revalidatingId ? 5000 : false
  })

  const {
    data: openstackCredentials,
    isLoading: loadingOpenstack,
    refetch: refetchOpenstack
  } = useOpenstackCredentialsQuery(undefined, {
    staleTime: 0,
    refetchOnMount: true,
    refetchInterval: revalidatingId ? 5000 : false
  })

  const [selectedIds, setSelectedIds] = useState<string[]>([])
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [selectedForDeletion, setSelectedForDeletion] = useState<CredentialItem[]>([])
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [vmwareCredDrawerOpen, setVmwareCredDrawerOpen] = useState(false)
  const [openstackCredDrawerOpen, setOpenstackCredDrawerOpen] = useState(false)

  const revalidationTimeoutRef = useRef<NodeJS.Timeout | null>(null)

  const { mutate: revalidate } = useMutation({
    mutationFn: revalidateCredentials,
    onSuccess: () => {
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: VMWARE_CREDS_QUERY_KEY })
        queryClient.invalidateQueries({ queryKey: OPENSTACK_CREDS_QUERY_KEY })
      }, 500)
    },
    onError: (error: any, variables) => {
      reportError(error, {
        context: 'credentials-revalidation',
        metadata: {
          credentialName: variables.name,
          credentialKind: variables.kind
        }
      })
      if (revalidationTimeoutRef.current) {
        clearTimeout(revalidationTimeoutRef.current)
        revalidationTimeoutRef.current = null
      }
      setRevalidatingId(null)
    }
  })

  const vmwareItems: CredentialItem[] =
    vmwareCredentials?.map((cred: VmwareCredential) => ({
      id: `vmware-${cred.metadata.name}`,
      name: cred.metadata.name,
      type: 'VMware' as const,
      status: cred.status?.vmwareValidationStatus || 'Unknown',
      credObject: cred
    })) || []

  const openstackItems: CredentialItem[] =
    openstackCredentials?.map((cred: OpenstackCredential) => ({
      id: `openstack-${cred.metadata.name}`,
      name: cred.metadata.name,
      type: 'OpenStack' as const,
      status: cred.status?.openstackValidationStatus || 'Unknown',
      credObject: cred
    })) || []

  const allCredentials = [...vmwareItems, ...openstackItems]

  useEffect(() => {
    if (revalidatingId) {
      const revalidatingItem = allCredentials.find((cred) => cred.id === revalidatingId)

      if (revalidatingItem) {
        const status = revalidatingItem.status.toLowerCase()
        if (status !== 'validating') {
          if (revalidationTimeoutRef.current) {
            clearTimeout(revalidationTimeoutRef.current)
            revalidationTimeoutRef.current = null
          }
          setRevalidatingId(null)
        }
      } else {
        if (revalidationTimeoutRef.current) {
          clearTimeout(revalidationTimeoutRef.current)
          revalidationTimeoutRef.current = null
        }
        setRevalidatingId(null)
      }
    }

    return () => {
      if (revalidationTimeoutRef.current) {
        clearTimeout(revalidationTimeoutRef.current)
        revalidationTimeoutRef.current = null
      }
    }
  }, [allCredentials, revalidatingId])

  useEffect(() => {
    refetchVmware()
    refetchOpenstack()
  }, [refetchVmware, refetchOpenstack])

  const handleRefresh = useCallback(() => {
    refetchVmware()
    refetchOpenstack()
  }, [refetchVmware, refetchOpenstack])

  const handleDeleteCredential = (id: string, type: 'VMware' | 'OpenStack') => {
    const credentialName = id.startsWith('vmware-')
      ? id.replace('vmware-', '')
      : id.replace('openstack-', '')

    const credential =
      type === 'VMware'
        ? vmwareCredentials?.find((cred) => cred.metadata.name === credentialName)
        : openstackCredentials?.find((cred) => cred.metadata.name === credentialName)

    if (credential) {
      const credItem: CredentialItem = {
        id,
        name: credential.metadata.name,
        type,
        status:
          type === 'VMware'
            ? (credential as VmwareCredential).status?.vmwareValidationStatus || 'Unknown'
            : (credential as OpenstackCredential).status?.openstackValidationStatus || 'Unknown',
        credObject: credential
      }
      setSelectedForDeletion([credItem])
      setDeleteDialogOpen(true)
    }
  }

  const handleRevalidateClick = (row: CredentialItem) => {
    const { credObject } = row
    if (!credObject) {
      reportError(new Error('Cannot revalidate: Missing credential object data.'), {
        context: 'credentials-revalidation',
        metadata: { credentialName: row.name }
      })
      return
    }

    if (revalidationTimeoutRef.current) {
      clearTimeout(revalidationTimeoutRef.current)
    }

    setRevalidatingId(row.id)

    revalidationTimeoutRef.current = setTimeout(() => {
      reportError(new Error(`Revalidation for ${row.name} timed out. Please check logs.`), {
        context: 'credentials-revalidation-timeout',
        metadata: { credentialName: row.name }
      })
      setRevalidatingId(null)
      revalidationTimeoutRef.current = null
    }, 120000)

    revalidate({
      name: credObject.metadata.name,
      namespace: credObject.metadata.namespace || VJAILBREAK_DEFAULT_NAMESPACE,
      kind: row.type === 'VMware' ? 'VmwareCreds' : 'OpenstackCreds'
    })
  }

  const handleSelectionChange = (rowSelectionModel: GridRowSelectionModel) => {
    setSelectedIds(rowSelectionModel as string[])
  }

  const handleDeleteSelected = () => {
    const selectedCreds = allCredentials.filter((cred) => selectedIds.includes(cred.id))
    setSelectedForDeletion(selectedCreds)
    setDeleteDialogOpen(true)
  }

  const handleDeleteClose = () => {
    setDeleteDialogOpen(false)
    setSelectedForDeletion([])
    setDeleteError(null)
  }

  const handleConfirmDelete = async () => {
    setDeleting(true)
    try {
      const vmwareCreds = selectedForDeletion.filter((cred) => cred.type === 'VMware')
      const openstackCreds = selectedForDeletion.filter((cred) => cred.type === 'OpenStack')

      await Promise.all(
        vmwareCreds.map((cred) => {
          const credName = cred.id.replace('vmware-', '')
          return deleteVMwareCredsWithSecretFlow(credName)
        })
      )

      await Promise.all(
        openstackCreds.map((cred) => {
          const credName = cred.id.replace('openstack-', '')
          return deleteOpenStackCredsWithSecretFlow(credName)
        })
      )

      queryClient.invalidateQueries({ queryKey: VMWARE_CREDS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: OPENSTACK_CREDS_QUERY_KEY })

      setSelectedIds([])
      handleDeleteClose()
    } catch (error) {
      console.error('Error deleting credentials:', error)
      reportError(error as Error, {
        context: 'credentials-deletion',
        metadata: {
          selectedIds: selectedIds,
          credentialsCount: selectedIds.length,
          action: 'delete-credentials'
        }
      })
      setDeleteError(error instanceof Error ? error.message : 'Unknown error occurred')
    } finally {
      setDeleting(false)
    }
  }

  const getCustomErrorMessage = useCallback((error: Error | string) => {
    const baseMessage = 'Failed to delete credentials'
    if (error instanceof Error) {
      return `${baseMessage}: ${error.message}`
    }
    return `${baseMessage}: ${error}`
  }, [])

  const tableColumns = getColumns(handleDeleteCredential, handleRevalidateClick, revalidatingId)

  const isLoading = loadingVmware || loadingOpenstack || deleting

  const handleOpenVMwareCredDrawer = () => {
    setVmwareCredDrawerOpen(true)
  }

  const handleCloseVMwareCredDrawer = () => {
    setVmwareCredDrawerOpen(false)
    refetchVmware()
  }

  const handleOpenOpenstackCredDrawer = () => {
    setOpenstackCredDrawerOpen(true)
  }

  const handleCloseOpenstackCredDrawer = () => {
    setOpenstackCredDrawerOpen(false)
    refetchOpenstack()
  }

  return (
    <div style={{ height: '100%', width: '100%', overflow: 'hidden' }}>
      <CommonDataGrid
        rows={allCredentials}
        columns={tableColumns}
        disableRowSelectionOnClick
        checkboxSelection
        rowSelectionModel={selectedIds}
        onRowSelectionModelChange={handleSelectionChange}
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
              numSelected={selectedIds.length}
              onDeleteSelected={handleDeleteSelected}
              loading={isLoading}
              onRefresh={handleRefresh}
              onAddVMwareCredential={handleOpenVMwareCredDrawer}
              onAddOpenstackCredential={handleOpenOpenstackCredDrawer}
            />
          )
        }}
        pageSizeOptions={[10, 25, 50, 100]}
        loading={isLoading}
        emptyMessage="No credentials available"
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
        message={
          selectedForDeletion.length > 1
            ? 'Are you sure you want to delete these credentials?'
            : `Are you sure you want to delete ${selectedForDeletion[0]?.type} credential "${selectedForDeletion[0]?.name}"?`
        }
        items={selectedForDeletion.map((cred) => ({
          id: cred.id,
          name: `${cred.name} (${cred.type})`
        }))}
        actionLabel="Delete"
        actionColor="error"
        actionVariant="outlined"
        onConfirm={handleConfirmDelete}
        customErrorMessage={getCustomErrorMessage}
        errorMessage={deleteError}
        onErrorChange={setDeleteError}
      />

      {vmwareCredDrawerOpen && (
        <VMwareCredentialsDrawer
          open={vmwareCredDrawerOpen}
          onClose={handleCloseVMwareCredDrawer}
        />
      )}

      {openstackCredDrawerOpen && (
        <OpenstackCredentialsDrawer
          open={openstackCredDrawerOpen}
          onClose={handleCloseOpenstackCredDrawer}
        />
      )}
    </div>
  )
}
