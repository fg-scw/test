import type { Meta, StoryObj } from '@storybook/react'
import { Box } from '@mui/material'
import StatusChip from './StatusChip'

const meta: Meta<typeof StatusChip> = {
  title: 'Components/Design System/StatusChip',
  component: StatusChip,
  args: {
    label: 'Running',
    size: 'small',
    variant: 'filled'
  },
  parameters: {
    layout: 'padded'
  }
}

export default meta

type Story = StoryObj<typeof StatusChip>

export const Default: Story = {}

export const Variants: Story = {
  render: () => (
    <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
      <StatusChip label="Succeeded" tone="success" size="small" variant="filled" />
      <StatusChip label="Failed" tone="error" size="small" variant="filled" />
      <StatusChip label="Running" tone="info" size="small" variant="filled" />
      <StatusChip label="Pending" tone="default" size="small" variant="filled" />
    </Box>
  )
}
