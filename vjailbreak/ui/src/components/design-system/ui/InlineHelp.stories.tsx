import type { Meta, StoryObj } from '@storybook/react'
import { Stack } from '@mui/material'
import InlineHelp from './InlineHelp'

const meta: Meta<typeof InlineHelp> = {
  title: 'Components/Design System/InlineHelp',
  component: InlineHelp,
  parameters: {
    layout: 'centered'
  },
  tags: ['autodocs']
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    children: 'This is a default inline help message with neutral styling.'
  }
}

export const Positive: Story = {
  args: {
    tone: 'positive',
    children: 'This is a positive message indicating success or confirmation.'
  }
}

export const Critical: Story = {
  args: {
    tone: 'critical',
    children: 'This is a critical message indicating an error or important warning.'
  }
}

export const Warning: Story = {
  args: {
    tone: 'warning',
    children: 'This is a warning message that requires attention.'
  }
}

export const AllTones: Story = {
  render: () => (
    <Stack spacing={2} sx={{ width: 400 }}>
      <InlineHelp tone="default">Default tone for general information.</InlineHelp>
      <InlineHelp tone="positive">Positive tone for success messages.</InlineHelp>
      <InlineHelp tone="warning">Warning tone for cautionary messages.</InlineHelp>
      <InlineHelp tone="critical">Critical tone for error messages.</InlineHelp>
    </Stack>
  )
}

export const Variants: Story = {
  render: () => (
    <Stack spacing={2} sx={{ width: 400 }}>
      <InlineHelp tone="default" variant="contained">
        Contained (default)
      </InlineHelp>
      <InlineHelp tone="default" variant="outline">
        Outline
      </InlineHelp>
    </Stack>
  )
}

export const Icons: Story = {
  render: () => (
    <Stack spacing={2} sx={{ width: 400 }}>
      <InlineHelp tone="default" icon="auto">
        Auto icon (defaults to info)
      </InlineHelp>
      <InlineHelp tone="positive" icon="auto">
        Auto icon with positive tone
      </InlineHelp>
      <InlineHelp tone="warning" icon="warning">
        Explicit warning icon
      </InlineHelp>
      <InlineHelp tone="critical" icon="danger">
        Explicit danger icon
      </InlineHelp>
    </Stack>
  )
}

export const LongContent: Story = {
  args: {
    tone: 'default',
    children:
      'This is a longer inline help message that demonstrates how the component handles extended content. It should wrap properly and maintain readability across different screen sizes.'
  }
}
