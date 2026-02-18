import type { Meta, StoryObj } from '@storybook/react'
import { useState } from 'react'
import ToggleField from './ToggleField'

const meta: Meta<typeof ToggleField> = {
  title: 'Components/Design System/ToggleField',
  component: ToggleField,
  args: {
    label: 'Enable data copy throttling',
    description: 'Toggles whether data copy jobs run with throttled throughput limits.',
    helperText: 'Keep toggles descriptive and pair them with helper copy when needed.',
    tooltip: 'This feature can reduce impact on busy clusters.'
  },
  parameters: {
    layout: 'centered'
  }
}

export default meta

type Story = StoryObj<typeof ToggleField>

export const Controlled: Story = {
  render: (args) => {
    const [checked, setChecked] = useState(false)
    return (
      <ToggleField
        {...args}
        checked={checked}
        onChange={(_, value) => setChecked(value)}
        name="throttle-toggle"
      />
    )
  }
}

export const WithCustomContainer: Story = {
  args: {
    containerProps: {
      elevation: 1,
      sx: { borderStyle: 'dashed' }
    }
  },
  render: (args) => <ToggleField {...args} checked onChange={() => {}} name="custom-container" />
}

export const Disabled: Story = {
  args: {
    description: 'Disabled toggles communicate scheduled maintenance windows.',
    helperText: 'Use helper text to explain why the toggle is locked.',
    disabled: true
  },
  render: (args) => (
    <ToggleField {...args} checked={false} onChange={() => {}} name="disabled-toggle" />
  )
}
