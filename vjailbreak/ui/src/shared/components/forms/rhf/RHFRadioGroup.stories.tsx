import type { Meta, StoryObj } from '@storybook/react'
import { useForm, FormProvider } from 'react-hook-form'
import { Box } from '@mui/material'
import RHFRadioGroup from './RHFRadioGroup'

const baseOptions = [
  { label: 'Option 1', value: 'option1' },
  { label: 'Option 2', value: 'option2' },
  { label: 'Option 3', value: 'option3' }
]

const meta: Meta<typeof RHFRadioGroup> = {
  title: 'Forms/RHF/RHFRadioGroup',
  component: RHFRadioGroup,
  parameters: {
    layout: 'centered'
  },
  tags: ['autodocs'],
  args: {
    options: baseOptions
  }
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
      <RHFRadioGroup {...args} name="option" options={args.options ?? baseOptions} />
    </FormWrapper>
  ),
  args: {
    label: 'Select an option',
    options: baseOptions
  }
}

export const WithHelperText: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFRadioGroup {...args} name="theme" options={args.options ?? baseOptions} />
    </FormWrapper>
  ),
  args: {
    label: 'Theme Preference',
    options: [
      { label: 'Light', value: 'light' },
      { label: 'Dark', value: 'dark' },
      { label: 'Auto', value: 'auto' }
    ],
    helperText: 'Choose your preferred theme'
  }
}

export const RowLayout: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFRadioGroup {...args} name="priority" options={args.options ?? baseOptions} />
    </FormWrapper>
  ),
  args: {
    label: 'Priority',
    row: true,
    options: [
      { label: 'Low', value: 'low' },
      { label: 'Medium', value: 'medium' },
      { label: 'High', value: 'high' }
    ]
  }
}

export const Required: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFRadioGroup {...args} name="requiredOption" options={args.options ?? baseOptions} />
    </FormWrapper>
  ),
  args: {
    label: 'Required Selection',
    options: [
      { label: 'Yes', value: 'yes' },
      { label: 'No', value: 'no' }
    ],
    rules: { required: 'Please select an option' }
  }
}

export const Disabled: Story = {
  render: (args) => (
    <FormWrapper defaultValues={{ disabled: 'option1' }}>
      <RHFRadioGroup {...args} name="disabled" options={args.options ?? baseOptions} />
    </FormWrapper>
  ),
  args: {
    label: 'Disabled Radio Group',
    disabled: true,
    options: [
      { label: 'Option 1', value: 'option1' },
      { label: 'Option 2', value: 'option2' }
    ]
  }
}

export const WithDefaultValue: Story = {
  render: (args) => (
    <FormWrapper defaultValues={{ selected: 'option2' }}>
      <RHFRadioGroup {...args} name="selected" options={args.options ?? baseOptions} />
    </FormWrapper>
  ),
  args: {
    label: 'Pre-selected Option',
    options: [
      { label: 'Option 1', value: 'option1' },
      { label: 'Option 2', value: 'option2' },
      { label: 'Option 3', value: 'option3' }
    ]
  }
}

export const ManyOptions: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFRadioGroup {...args} name="country" options={args.options ?? baseOptions} />
    </FormWrapper>
  ),
  args: {
    label: 'Country',
    options: [
      { label: 'United States', value: 'us' },
      { label: 'Canada', value: 'ca' },
      { label: 'United Kingdom', value: 'uk' },
      { label: 'Germany', value: 'de' },
      { label: 'France', value: 'fr' },
      { label: 'Japan', value: 'jp' }
    ]
  }
}
