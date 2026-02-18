import type { Meta, StoryObj } from '@storybook/react'
import { Stack, Typography } from '@mui/material'
import SurfaceCard from './SurfaceCard'
import ActionButton from './ActionButton'

const meta: Meta<typeof SurfaceCard> = {
  title: 'Components/Design System/SurfaceCard',
  component: SurfaceCard,
  args: {
    title: 'Cluster Overview',
    subtitle: 'Summary of migration readiness signals',
    children: (
      <Typography variant="body2" color="text.secondary">
        Use cards to group related information blocks and optional actions.
      </Typography>
    )
  },
  parameters: {
    layout: 'centered'
  }
}

export default meta

type Story = StoryObj<typeof SurfaceCard>

export const Default: Story = {}

export const WithActions: Story = {
  render: (args) => (
    <SurfaceCard
      {...args}
      actions={<ActionButton tone="secondary">Manage</ActionButton>}
      footer={
        <Stack direction="row" justifyContent="flex-end" spacing={1}>
          <ActionButton tone="secondary">Dismiss</ActionButton>
          <ActionButton tone="primary">Approve</ActionButton>
        </Stack>
      }
    >
      <Stack spacing={1}>
        <Typography variant="body2">• 12 VMs pending validation</Typography>
        <Typography variant="body2">• 3 VMware clusters ready</Typography>
      </Stack>
    </SurfaceCard>
  )
}
