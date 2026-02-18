import type { Meta, StoryObj } from '@storybook/react'
import { useForm, FormProvider } from 'react-hook-form'
import { Box } from '@mui/material'
import RHFFileField from './RHFFileField'

const meta: Meta<typeof RHFFileField> = {
  title: 'Forms/RHF/RHFFileField',
  component: RHFFileField,
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
      <Box sx={{ width: 400 }}>{children}</Box>
    </FormProvider>
  )
}

export const Default: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFFileField {...args} name="file" />
    </FormWrapper>
  ),
  args: {
    label: 'Upload File',
    placeholder: 'No file selected'
  }
}

export const WithHelperText: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFFileField {...args} name="document" />
    </FormWrapper>
  ),
  args: {
    label: 'Document Upload',
    helperText: 'Upload a PDF or Word document',
    labelHelperText: 'Accepted formats: PDF, DOC, DOCX'
  }
}

export const WithAcceptFilter: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFFileField {...args} name="image" />
    </FormWrapper>
  ),
  args: {
    label: 'Image Upload',
    accept: 'image/*',
    helperText: 'Upload an image file (JPG, PNG, GIF)'
  }
}

export const Required: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFFileField {...args} name="requiredFile" />
    </FormWrapper>
  ),
  args: {
    label: 'Required File',
    required: true,
    rules: { required: 'File is required' }
  }
}

export const Disabled: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFFileField {...args} name="disabledFile" />
    </FormWrapper>
  ),
  args: {
    label: 'Disabled File Upload',
    disabled: true
  }
}

export const WithValidation: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFFileField {...args} name="validatedFile" />
    </FormWrapper>
  ),
  args: {
    label: 'File with Size Validation',
    accept: 'image/*',
    rules: {
      required: 'File is required',
      validate: (value: File | undefined) => {
        if (!value) return 'File is required'
        const maxSize = 5 * 1024 * 1024 // 5MB
        return value.size <= maxSize || 'File size must be less than 5MB'
      }
    },
    helperText: 'Maximum file size: 5MB'
  }
}
