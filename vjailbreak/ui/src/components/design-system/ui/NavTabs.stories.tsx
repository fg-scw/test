import type { Meta, StoryObj } from '@storybook/react'
import { useState } from 'react'
import { Box } from '@mui/material'
import { NavTabs, NavTab } from './NavTabs'

const meta: Meta<typeof NavTabs> = {
  title: 'Components/Design System/NavTabs',
  component: NavTabs,
  args: {
    value: 0
  },
  parameters: {
    layout: 'padded'
  }
}

export default meta

type Story = StoryObj<typeof NavTabs>

export const Default: Story = {
  render: (args) => (
    <NavTabs {...args}>
      <NavTab label="Migrations" description="Current runs" value={0} />
      <NavTab label="Agents" description="Connected" value={1} />
      <NavTab label="Credentials" count={12} value={2} />
      <NavTab label="Cluster Conversions" value={3} />
      <NavTab label="Bare Metal Config" value={4} />
    </NavTabs>
  )
}

export const Interactive: Story = {
  render: () => {
    const [value, setValue] = useState(0)
    return (
      <Box sx={{ width: '100%' }}>
        <NavTabs value={value} onChange={(_, newValue) => setValue(newValue)}>
          <NavTab label="Overview" description="High-level" value={0} />
          <NavTab label="Activity" description="Audit trail" value={1} />
          <NavTab label="Settings" value={2} />
        </NavTabs>
      </Box>
    )
  }
}
