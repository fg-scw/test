import type { Meta, StoryObj } from '@storybook/react'
import { useForm, FormProvider } from 'react-hook-form'
import { Box } from '@mui/material'
import RHFOpenstackRCFileField from './RHFOpenstackRCFileField'

const meta: Meta<typeof RHFOpenstackRCFileField> = {
  title: 'Forms/RHF/RHFOpenstackRCFileField',
  component: RHFOpenstackRCFileField,
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
      <Box sx={{ width: 500 }}>{children}</Box>
    </FormProvider>
  )
}

export const Default: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFOpenstackRCFileField {...args} />
    </FormWrapper>
  ),
  args: {
    labelHelperText: 'Upload the RC file exported from your OpenStack environment.'
  }
}

export const NotRequired: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFOpenstackRCFileField {...args} />
    </FormWrapper>
  ),
  args: {
    required: false,
    labelHelperText: 'Optional RC file upload'
  }
}

export const SmallSize: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFOpenstackRCFileField {...args} />
    </FormWrapper>
  ),
  args: {
    size: 'small',
    labelHelperText: 'Small size variant'
  }
}

export const MediumSize: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFOpenstackRCFileField {...args} />
    </FormWrapper>
  ),
  args: {
    size: 'medium',
    labelHelperText: 'Medium size variant'
  }
}

export const WithExternalError: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFOpenstackRCFileField {...args} />
    </FormWrapper>
  ),
  args: {
    externalError: 'This is an external error message',
    labelHelperText: 'Example with external error'
  }
}

export const WithCustomName: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFOpenstackRCFileField {...args} />
    </FormWrapper>
  ),
  args: {
    name: 'customRCFile',
    labelHelperText: 'Custom field name example'
  }
}
