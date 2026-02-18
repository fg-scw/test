import type { Meta, StoryObj } from '@storybook/react'
import { useState } from 'react'

import CustomSearchToolbar from './CustomSearchToolbar'

const meta: Meta<typeof CustomSearchToolbar> = {
  title: 'Components/Grid/CustomSearchToolbar',
  component: CustomSearchToolbar,
  parameters: {
    layout: 'padded'
  },
  tags: ['autodocs']
}

export default meta

type Story = StoryObj<typeof CustomSearchToolbar>

export const Interactive: Story = {
  render: () => {
    const [status, setStatus] = useState('All')
    const [date, setDate] = useState('All Time')

    return (
      <CustomSearchToolbar
        title="Runs"
        placeholder="Search by name"
        onRefresh={() => {}}
        onStatusFilterChange={(s) => setStatus(s)}
        currentStatusFilter={status}
        onDateFilterChange={(d) => setDate(d)}
        currentDateFilter={date}
      />
    )
  }
}

export const Minimal: Story = {
  args: {
    title: 'Credentials'
  }
}
