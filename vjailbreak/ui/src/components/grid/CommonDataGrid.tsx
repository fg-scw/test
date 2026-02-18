import { Box, CircularProgress, Typography, styled } from '@mui/material'
import { DataGrid, DataGridProps, GridOverlay, GridValidRowModel } from '@mui/x-data-grid'
import { alpha } from '@mui/material/styles'
import type { SxProps, Theme } from '@mui/material/styles'

const StyledGridOverlay = styled(GridOverlay)(({ theme }) => ({
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  justifyContent: 'center',
  height: '100%',
  backgroundColor: alpha(
    theme.palette.background.paper,
    theme.palette.mode === 'dark' ? 0.72 : 0.92
  ),
  color: theme.palette.text.primary,
  padding: theme.spacing(3)
}))

function DefaultLoadingOverlay({ loadingMessage }: { loadingMessage?: string }) {
  return (
    <StyledGridOverlay>
      <CircularProgress size={28} />
      {loadingMessage ? (
        <Typography variant="body2" sx={{ mt: 2, color: 'text.secondary' }}>
          {loadingMessage}
        </Typography>
      ) : null}
    </StyledGridOverlay>
  )
}

function DefaultNoRowsOverlay({ emptyMessage }: { emptyMessage?: string }) {
  return (
    <StyledGridOverlay>
      <Box sx={{ maxWidth: 520, textAlign: 'center' }}>
        <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
          {emptyMessage || 'Nothing to show'}
        </Typography>
        <Typography variant="body2" sx={{ mt: 0.5, color: 'text.secondary' }}>
          Once data is available, it will appear here.
        </Typography>
      </Box>
    </StyledGridOverlay>
  )
}

type CommonDataGridProps<R extends GridValidRowModel> = DataGridProps<R> & {
  loadingMessage?: string
  emptyMessage?: string
}

export default function CommonDataGrid<R extends GridValidRowModel>(props: CommonDataGridProps<R>) {
  const { loadingMessage, emptyMessage, slots, slotProps, sx, ...rest } = props

  const mergedSlots = {
    loadingOverlay: () => <DefaultLoadingOverlay loadingMessage={loadingMessage} />,
    noRowsOverlay: () => <DefaultNoRowsOverlay emptyMessage={emptyMessage} />,
    ...slots
  }

  const baseSx = (theme) => ({
    border: `1px solid ${theme.palette.divider}`,
    borderRadius: 2,
    backgroundColor: theme.palette.background.paper,
    boxShadow: theme.palette.mode === 'dark' ? 'none' : theme.shadows[1],

    '& .MuiDataGrid-columnHeaders': {
      borderBottom: `1px solid ${theme.palette.divider}`
    },
    '& .MuiDataGrid-columnHeaders, & .MuiDataGrid-columnHeadersInner, & .MuiDataGrid-container--top [role="row"], & .MuiDataGrid-columnHeader':
      {
        backgroundColor: `${theme.palette.background.paper} !important`
      },
    '& .MuiDataGrid-columnHeaderTitle': {
      fontWeight: 600,
      fontSize: '0.8125rem'
    },
    '& .MuiDataGrid-columnSeparator': {
      color: alpha(theme.palette.text.primary, 0.12)
    },
    '& .MuiDataGrid-cell': {
      py: 1.25,
      lineHeight: 1.4,
      display: 'flex',
      alignItems: 'center'
    },
    '& .MuiDataGrid-row': {
      maxHeight: 'none !important'
    },
    '& .MuiDataGrid-row:hover': {
      backgroundColor: theme.palette.action.hover
    },
    '& .MuiDataGrid-row.Mui-selected': {
      backgroundColor: alpha(
        theme.palette.primary.main,
        theme.palette.mode === 'dark' ? 0.22 : 0.08
      )
    },
    '& .MuiDataGrid-row.Mui-selected:hover': {
      backgroundColor: alpha(
        theme.palette.primary.main,
        theme.palette.mode === 'dark' ? 0.28 : 0.12
      )
    },
    '& .MuiDataGrid-cell:focus, & .MuiDataGrid-columnHeader:focus, & .MuiDataGrid-cell:focus-within, & .MuiDataGrid-columnHeader:focus-within':
      {
        outline: 'none'
      },
    '& .MuiDataGrid-footerContainer': {
      borderTop: `1px solid ${theme.palette.divider}`
    },
    '& .MuiDataGrid-virtualScroller': {
      backgroundColor: theme.palette.background.paper
    },
    '& .MuiDataGrid-main': {
      overflow: 'auto'
    }
  })

  const mergedSx: SxProps<Theme> = (() => {
    if (!sx) return baseSx
    if (Array.isArray(sx)) return [baseSx, ...sx]
    return [baseSx, sx]
  })()

  return <DataGrid {...rest} slots={mergedSlots} slotProps={slotProps} sx={mergedSx} />
}
