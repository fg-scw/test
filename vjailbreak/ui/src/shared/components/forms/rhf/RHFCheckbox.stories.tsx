import type { Meta, StoryObj } from '@storybook/react'
import { useForm, FormProvider } from 'react-hook-form'
import { Box } from '@mui/material'
import RHFCheckbox from './RHFCheckbox'

const meta: Meta<typeof RHFCheckbox> = {
  title: 'Forms/RHF/RHFCheckbox',
  component: RHFCheckbox,
  parameters: {
    layout: 'centered'
  },
  tags: ['autodocs']
}

export default meta
type Story = StoryObj<typeof meta>

// Wrapper component to provide form context
const FormWrapper = ({
  children,
  defaultValues = {}
}: {
  children: React.ReactNode
  defaultValues?: any
}) => {
  const form = useForm({ defaultValues })
  return (
    <FormProvider {...form}>
      <Box sx={{ width: 300 }}>{children}</Box>
    </FormProvider>
  )
}

export const Default: Story = {
  render: (args) => (
    <FormWrapper defaultValues={{ acceptTerms: false }}>
      <RHFCheckbox {...args} name="acceptTerms" />
    </FormWrapper>
  ),
  args: {
    label: 'I accept the terms and conditions'
  }
}

export const WithHelperText: Story = {
  render: (args) => (
    <FormWrapper defaultValues={{ newsletter: false }}>
      <RHFCheckbox {...args} name="newsletter" />
    </FormWrapper>
  ),
  args: {
    label: 'Subscribe to newsletter',
    helperText: 'Receive updates about new features and products'
  }
}

export const Required: Story = {
  render: (args) => (
    <FormWrapper defaultValues={{ required: false }}>
      <RHFCheckbox {...args} name="required" />
    </FormWrapper>
  ),
  args: {
    label: 'Required checkbox',
    rules: { required: 'This field is required' }
  }
}

export const Disabled: Story = {
  render: (args) => (
    <FormWrapper defaultValues={{ disabled: true }}>
      <RHFCheckbox {...args} name="disabled" />
    </FormWrapper>
  ),
  args: {
    label: 'Disabled checkbox',
    disabled: true
  }
}

export const WithValidation: Story = {
  render: (args) => (
    <FormWrapper defaultValues={{ mustAccept: false }}>
      <RHFCheckbox {...args} name="mustAccept" />
    </FormWrapper>
  ),
  args: {
    label: 'You must accept to continue',
    rules: {
      required: 'You must accept this to continue',
      validate: (value) => value === true || 'This must be checked'
    }
  }
}
