import type { Meta, StoryObj } from '@storybook/react'
import { useForm, FormProvider } from 'react-hook-form'
import { Box } from '@mui/material'
import RHFDateField from './RHFDateField'

const meta: Meta<typeof RHFDateField> = {
  title: 'Forms/RHF/RHFDateField',
  component: RHFDateField,
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
    <FormWrapper>
      <RHFDateField {...args} name="birthDate" />
    </FormWrapper>
  ),
  args: {
    label: 'Date of Birth',
    placeholder: 'Select a date'
  }
}

export const WithHelperText: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFDateField {...args} name="startDate" />
    </FormWrapper>
  ),
  args: {
    label: 'Start Date',
    helperText: 'Select the start date for your project',
    labelHelperText: 'This date will be used to schedule your project'
  }
}

export const Required: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFDateField {...args} name="requiredDate" />
    </FormWrapper>
  ),
  args: {
    label: 'Required Date',
    required: true,
    rules: { required: 'Date is required' }
  }
}

export const Disabled: Story = {
  render: (args) => (
    <FormWrapper defaultValues={{ disabledDate: '2024-01-01' }}>
      <RHFDateField {...args} name="disabledDate" />
    </FormWrapper>
  ),
  args: {
    label: 'Disabled Date',
    disabled: true
  }
}

export const WithValidation: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFDateField {...args} name="futureDate" />
    </FormWrapper>
  ),
  args: {
    label: 'Future Date',
    rules: {
      required: 'Date is required',
      validate: (value) => {
        if (!value) return 'Date is required'
        const selectedDate = new Date(value)
        const today = new Date()
        today.setHours(0, 0, 0, 0)
        return selectedDate > today || 'Date must be in the future'
      }
    }
  }
}

export const WithDefaultValue: Story = {
  render: (args) => (
    <FormWrapper defaultValues={{ defaultDate: '2024-06-15' }}>
      <RHFDateField {...args} name="defaultDate" />
    </FormWrapper>
  ),
  args: {
    label: 'Date with Default Value'
  }
}
