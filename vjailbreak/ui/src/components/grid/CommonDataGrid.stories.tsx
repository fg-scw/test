import type { Meta, StoryObj } from '@storybook/react'
import { Box } from '@mui/material'

import CommonDataGrid from './CommonDataGrid'

const meta: Meta<typeof CommonDataGrid> = {
  title: 'Components/Grid/CommonDataGrid',
  component: CommonDataGrid,
  parameters: {
    layout: 'padded'
  },
  tags: ['autodocs']
}

export default meta

type Story = StoryObj<typeof CommonDataGrid>

type Row = {
  id: string
  name: string
  status: string
}

const rows: Row[] = [
  { id: '1', name: 'vmware-cred-01', status: 'Ready' },
  { id: '2', name: 'vmware-cred-02', status: 'Pending' },
  { id: '3', name: 'vmware-cred-03', status: 'Failed' }
]

export const Default: Story = {
  render: () => (
    <Box sx={{ height: 360, width: 720 }}>
      <CommonDataGrid<Row>
        rows={rows}
        columns={[
          { field: 'name', headerName: 'Name', flex: 1, minWidth: 180 },
          { field: 'status', headerName: 'Status', width: 140 }
        ]}
        disableRowSelectionOnClick
        pageSizeOptions={[5, 10, 25]}
      />
    </Box>
  )
}

export const Loading: Story = {
  render: () => (
    <Box sx={{ height: 360, width: 720 }}>
      <CommonDataGrid<Row>
        rows={[]}
        columns={[
          { field: 'name', headerName: 'Name', flex: 1, minWidth: 180 },
          { field: 'status', headerName: 'Status', width: 140 }
        ]}
        loading
        loadingMessage="Fetching latest status…"
        disableRowSelectionOnClick
        pageSizeOptions={[5, 10, 25]}
      />
    </Box>
  )
}

export const Empty: Story = {
  render: () => (
    <Box sx={{ height: 360, width: 720 }}>
      <CommonDataGrid<Row>
        rows={[]}
        columns={[
          { field: 'name', headerName: 'Name', flex: 1, minWidth: 180 },
          { field: 'status', headerName: 'Status', width: 140 }
        ]}
        emptyMessage="No items found"
        disableRowSelectionOnClick
        pageSizeOptions={[5, 10, 25]}
      />
    </Box>
  )
}
