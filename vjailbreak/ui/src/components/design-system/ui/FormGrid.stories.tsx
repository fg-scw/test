import type { Meta, StoryObj } from '@storybook/react'
import { Box, TextField } from '@mui/material'
import FieldLabel from './FieldLabel'
import FormGrid from './FormGrid'

const meta: Meta<typeof FormGrid> = {
  title: 'Components/Design System/FormGrid',
  component: FormGrid,
  args: {
    minWidth: 320,
    gap: 2
  },
  parameters: {
    layout: 'centered'
  }
}

export default meta

type Story = StoryObj<typeof FormGrid>

const SampleFields = () => (
  <>
    {[
      { label: 'Cluster Name', placeholder: 'pf9-cluster-01' },
      { label: 'Credential Name', placeholder: 'ops-team-admin' },
      { label: 'Region', placeholder: 'us-west-1' },
      { label: 'Project', placeholder: 'prod-shared' }
    ].map(({ label, placeholder }) => (
      <Box key={label} display="flex" flexDirection="column" gap={0.5}>
        <FieldLabel label={label} helperText="Consistent helper text keeps UX predictable." />
        <TextField size="small" placeholder={placeholder} fullWidth />
      </Box>
    ))}
  </>
)

export const Default: Story = {
  render: (args) => (
    <FormGrid {...args}>
      <SampleFields />
    </FormGrid>
  )
}

export const CompactGap: Story = {
  args: {
    gap: 1.5
  },
  render: (args) => (
    <FormGrid {...args}>
      <SampleFields />
    </FormGrid>
  )
}

export const WideColumns: Story = {
  args: {
    minWidth: 420
  },
  render: (args) => (
    <FormGrid {...args}>
      <SampleFields />
    </FormGrid>
  )
}
