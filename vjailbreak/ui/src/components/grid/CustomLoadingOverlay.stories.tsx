import type { Meta, StoryObj } from '@storybook/react'
import { Box } from '@mui/material'

import CustomLoadingOverlay from './CustomLoadingOverlay'

const meta: Meta<typeof CustomLoadingOverlay> = {
  title: 'Components/Grid/CustomLoadingOverlay',
  component: CustomLoadingOverlay,
  parameters: {
    layout: 'centered'
  },
  tags: ['autodocs']
}

export default meta

type Story = StoryObj<typeof CustomLoadingOverlay>

export const Default: Story = {
  args: {
    loadingMessage: 'Loading…'
  },
  render: (args) => (
    <Box sx={{ width: 520, height: 240, border: '1px dashed', borderColor: 'divider' }}>
      <CustomLoadingOverlay {...args} />
    </Box>
  )
}
