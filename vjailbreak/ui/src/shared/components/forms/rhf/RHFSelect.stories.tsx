import type { Meta, StoryObj } from '@storybook/react'
import { useForm, FormProvider } from 'react-hook-form'
import { Box } from '@mui/material'
import RHFSelect from './RHFSelect'

const baseOptions = [
  { label: 'Option 1', value: 'option1' },
  { label: 'Option 2', value: 'option2' },
  { label: 'Option 3', value: 'option3' }
]

const meta: Meta<typeof RHFSelect> = {
  title: 'Forms/RHF/RHFSelect',
  component: RHFSelect,
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
      <RHFSelect {...args} name="option" options={args.options ?? baseOptions} />
    </FormWrapper>
  ),
  args: {
    label: 'Select an option',
    placeholder: 'Choose an option',
    options: baseOptions
  }
}

export const WithHelperText: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFSelect {...args} name="country" options={args.options ?? baseOptions} />
    </FormWrapper>
  ),
  args: {
    label: 'Country',
    placeholder: 'Select a country',
    options: [
      { label: 'United States', value: 'us' },
      { label: 'Canada', value: 'ca' },
      { label: 'United Kingdom', value: 'uk' }
    ],
    helperText: 'Select your country of residence',
    labelHelperText: 'This will be used for shipping calculations'
  }
}

export const Required: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFSelect {...args} name="requiredOption" options={args.options ?? baseOptions} />
    </FormWrapper>
  ),
  args: {
    label: 'Required Selection',
    placeholder: 'Please select',
    required: true,
    options: [
      { label: 'Option 1', value: 'option1' },
      { label: 'Option 2', value: 'option2' }
    ],
    rules: { required: 'This field is required' }
  }
}

export const Disabled: Story = {
  render: (args) => (
    <FormWrapper defaultValues={{ disabled: 'option1' }}>
      <RHFSelect {...args} name="disabled" options={args.options ?? baseOptions} />
    </FormWrapper>
  ),
  args: {
    label: 'Disabled Select',
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
      <RHFSelect {...args} name="selected" options={args.options ?? baseOptions} />
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
      <RHFSelect {...args} name="state" options={args.options ?? baseOptions} />
    </FormWrapper>
  ),
  args: {
    label: 'State',
    placeholder: 'Select a state',
    options: [
      { label: 'California', value: 'ca' },
      { label: 'New York', value: 'ny' },
      { label: 'Texas', value: 'tx' },
      { label: 'Florida', value: 'fl' },
      { label: 'Illinois', value: 'il' },
      { label: 'Pennsylvania', value: 'pa' }
    ]
  }
}

export const Searchable: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFSelect {...args} name="searchableOption" options={args.options ?? baseOptions} />
    </FormWrapper>
  ),
  args: {
    label: 'Searchable Select',
    placeholder: 'Select a city',
    searchable: true,
    searchPlaceholder: 'Search cities…',
    options: [
      { label: 'San Francisco', value: 'sf' },
      { label: 'San Jose', value: 'sj' },
      { label: 'Los Angeles', value: 'la' },
      { label: 'San Diego', value: 'sd' },
      { label: 'New York City', value: 'nyc' },
      { label: 'Buffalo', value: 'buf' },
      { label: 'Chicago', value: 'chi' },
      { label: 'Houston', value: 'hou' },
      { label: 'Austin', value: 'aus' },
      { label: 'Dallas', value: 'dal' }
    ]
  }
}

export const WithValidation: Story = {
  render: (args) => (
    <FormWrapper>
      <RHFSelect {...args} name="validated" options={args.options ?? baseOptions} />
    </FormWrapper>
  ),
  args: {
    label: 'Validated Select',
    placeholder: 'Select an option',
    options: [
      { label: 'Option 1', value: 'option1' },
      { label: 'Option 2', value: 'option2' },
      { label: 'Option 3', value: 'option3' }
    ],
    rules: {
      required: 'Please select an option',
      validate: (value) => value !== 'option1' || 'Option 1 is not allowed'
    }
  }
}
