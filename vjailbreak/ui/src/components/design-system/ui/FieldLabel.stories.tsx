import type { Meta, StoryObj } from '@storybook/react'
import FieldLabel from './FieldLabel'

const meta: Meta<typeof FieldLabel> = {
  title: 'Components/Design System/FieldLabel',
  component: FieldLabel,
  args: {
    label: 'Cluster Name',
    helperText: 'Helper copy keeps inputs consistent across forms.'
  },
  parameters: {
    layout: 'centered'
  }
}

export default meta

type Story = StoryObj<typeof FieldLabel>

export const Default: Story = {}

export const Required: Story = {
  args: {
    required: true,
    helperText: 'Required labels display an accent *.'
  }
}

export const WithTooltip: Story = {
  args: {
    tooltip: 'Tooltips surface extra context when hovering the info icon.'
  }
}
