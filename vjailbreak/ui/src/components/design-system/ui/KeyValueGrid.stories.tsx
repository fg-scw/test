import type { Meta, StoryObj } from '@storybook/react'
import { KeyValueItem } from './KeyValueGrid'
import KeyValueGrid from './KeyValueGrid'

const meta: Meta<typeof KeyValueGrid> = {
  title: 'Components/Design System/KeyValueGrid',
  component: KeyValueGrid,
  args: {
    items: []
  },
  parameters: {
    layout: 'padded'
  }
}

export default meta

type Story = StoryObj<typeof KeyValueGrid>

const sampleItems: KeyValueItem[] = [
  { label: 'Datacenter', value: 'DC-1' },
  { label: 'Source Cluster', value: 'Cluster-A' },
  { label: 'Destination Cluster', value: 'Cluster-B' },
  { label: 'Notes', value: 'Longer text that wraps when the container is narrow.' }
]

export const Default: Story = {
  args: {
    items: sampleItems
  }
}
