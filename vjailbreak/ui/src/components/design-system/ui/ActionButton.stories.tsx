import type { Meta, StoryObj } from '@storybook/react'
import { Stack } from '@mui/material'
import ActionButton from './ActionButton'

const meta: Meta<typeof ActionButton> = {
  title: 'Components/Design System/ActionButton',
  component: ActionButton,
  args: {
    children: 'Primary CTA'
  },
  parameters: {
    layout: 'centered'
  }
}

export default meta

type Story = StoryObj<typeof ActionButton>

export const Primary: Story = {
  args: {
    tone: 'primary'
  }
}

export const Secondary: Story = {
  render: (args) => (
    <Stack direction="row" spacing={2}>
      <ActionButton {...args} tone="secondary">
        Cancel
      </ActionButton>
      <ActionButton {...args} tone="primary">
        Save Changes
      </ActionButton>
    </Stack>
  )
}

export const DangerLoading: Story = {
  args: {
    tone: 'danger',
    loading: true,
    children: 'Deleting...'
  }
}
