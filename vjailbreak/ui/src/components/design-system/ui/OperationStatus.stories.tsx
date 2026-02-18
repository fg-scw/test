import type { Meta, StoryObj } from '@storybook/react'
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline'

import OperationStatus from './OperationStatus'

const meta: Meta<typeof OperationStatus> = {
  title: 'Components/Design System/OperationStatus',
  component: OperationStatus,
  parameters: {
    layout: 'centered'
  },
  args: {
    sx: { width: 420 }
  },
  tags: ['autodocs']
}

export default meta

type Story = StoryObj<typeof OperationStatus>

export const LoadingRow: Story = {
  args: {
    loading: true,
    loadingMessage: 'Validating credentials…'
  }
}

export const LoadingText: Story = {
  args: {
    loading: true,
    loadingLayout: 'text',
    loadingMessage: 'Syncing inventory in the background.'
  }
}

export const SuccessText: Story = {
  args: {
    success: true,
    successMessage: 'Saved successfully.'
  }
}

export const SuccessRowWithIcon: Story = {
  args: {
    success: true,
    successIcon: <CheckCircleOutlineIcon fontSize="small" color="success" />,
    successMessage: 'Connection verified.'
  }
}

export const Error: Story = {
  args: {
    error: 'Unable to reach the endpoint. Check connectivity and try again.'
  }
}
