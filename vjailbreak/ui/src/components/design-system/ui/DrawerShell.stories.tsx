import type { Meta, StoryObj } from '@storybook/react'
import { useState } from 'react'
import { Box, Typography } from '@mui/material'
import DrawerShell, { DrawerHeader, DrawerBody, DrawerFooter } from './DrawerShell'
import ActionButton from './ActionButton'

const meta: Meta<typeof DrawerShell> = {
  title: 'Components/Design System/DrawerShell',
  component: DrawerShell,
  parameters: {
    layout: 'fullscreen'
  },
  args: {
    open: true
  }
}

export default meta

type Story = StoryObj<typeof DrawerShell>

export const Playground: Story = {
  render: (args) => (
    <DrawerShell
      {...args}
      header={<DrawerHeader title="Add VMware Credentials" subtitle="Securely stored in-cluster" />}
      footer={
        <DrawerFooter>
          <ActionButton tone="secondary">Cancel</ActionButton>
          <ActionButton tone="primary">Create Credential</ActionButton>
        </DrawerFooter>
      }
    >
      <DrawerBody>
        <Typography variant="body2" color="text.secondary">
          Place interactive form content here. DrawerBody already handles padding and scrolling.
        </Typography>
      </DrawerBody>
    </DrawerShell>
  )
}

export const Controlled: Story = {
  render: () => {
    const [open, setOpen] = useState(true)

    return (
      <Box>
        <ActionButton tone="primary" onClick={() => setOpen(true)}>
          Launch Drawer
        </ActionButton>
        <DrawerShell
          open={open}
          onClose={() => setOpen(false)}
          header={<DrawerHeader title="Cluster Conversion" onClose={() => setOpen(false)} />}
          footer={
            <DrawerFooter>
              <ActionButton tone="secondary" onClick={() => setOpen(false)}>
                Cancel
              </ActionButton>
              <ActionButton tone="primary">Save</ActionButton>
            </DrawerFooter>
          }
        >
          <DrawerBody>
            <Typography variant="body2">
              Controlled example showing how to wire onClose + footer actions.
            </Typography>
          </DrawerBody>
        </DrawerShell>
      </Box>
    )
  }
}
