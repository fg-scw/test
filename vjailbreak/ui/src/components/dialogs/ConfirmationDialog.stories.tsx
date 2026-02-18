import type { Meta, StoryObj } from '@storybook/react'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import { useState } from 'react'

import ConfirmationDialog from './ConfirmationDialog'

const meta: Meta<typeof ConfirmationDialog> = {
  title: 'Components/Dialogs/ConfirmationDialog',
  component: ConfirmationDialog,
  parameters: {
    layout: 'centered'
  },
  tags: ['autodocs']
}

export default meta

type Story = StoryObj<typeof ConfirmationDialog>

export const Default: Story = {
  args: {
    open: true,
    title: 'Delete credential?',
    message: 'This action cannot be undone.',
    icon: <DeleteOutlineIcon color="error" />,
    actionLabel: 'Delete',
    actionColor: 'error',
    onClose: () => {},
    onConfirm: async () => {}
  }
}

export const WithItems: Story = {
  args: {
    open: true,
    title: 'Remove agents?',
    message: 'The following agents will be removed:',
    actionLabel: 'Remove',
    actionColor: 'warning',
    items: [
      { id: 'a1', name: 'agent-01' },
      { id: 'a2', name: 'agent-02' },
      { id: 'a3', name: 'agent-03' },
      { id: 'a4', name: 'agent-04' },
      { id: 'a5', name: 'agent-05' }
    ],
    onClose: () => {},
    onConfirm: async () => {}
  }
}

export const Interactive: Story = {
  render: (args) => {
    const [open, setOpen] = useState(true)

    return (
      <ConfirmationDialog
        {...args}
        open={open}
        onClose={() => setOpen(false)}
        title="Rotate API token?"
        message="This will invalidate the existing token and generate a new one."
        actionLabel="Rotate"
        onConfirm={async () => {
          await new Promise((r) => setTimeout(r, 800))
        }}
      />
    )
  }
}
