import type { Meta, StoryObj } from '@storybook/react'
import { TextField, Button } from '@mui/material'
import Section from './Section'
import SectionHeader from './SectionHeader'
import FieldLabel from './FieldLabel'

const meta: Meta<typeof Section> = {
  title: 'Components/Design System/Section',
  component: Section,
  parameters: {
    layout: 'centered'
  },
  tags: ['autodocs']
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  render: () => (
    <Section sx={{ width: 400 }}>
      <div>First item</div>
      <div>Second item</div>
      <div>Third item</div>
    </Section>
  )
}

export const WithFormFields: Story = {
  render: () => (
    <Section sx={{ width: 400 }}>
      <FieldLabel label="First Name" required />
      <TextField size="small" placeholder="Enter first name" fullWidth />
      <FieldLabel label="Last Name" required />
      <TextField size="small" placeholder="Enter last name" fullWidth />
      <FieldLabel label="Email" helperText="We'll never share your email" />
      <TextField size="small" type="email" placeholder="Enter email" fullWidth />
    </Section>
  )
}

export const WithSectionHeader: Story = {
  render: () => (
    <Section sx={{ width: 400 }}>
      <SectionHeader title="User Information" subtitle="Enter your personal details below" />
      <FieldLabel label="First Name" required />
      <TextField size="small" placeholder="Enter first name" fullWidth />
      <FieldLabel label="Last Name" required />
      <TextField size="small" placeholder="Enter last name" fullWidth />
    </Section>
  )
}

export const MultipleSections: Story = {
  render: () => (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 24, width: 400 }}>
      <Section>
        <SectionHeader title="Personal Information" />
        <FieldLabel label="Name" required />
        <TextField size="small" placeholder="Enter name" fullWidth />
        <FieldLabel label="Email" required />
        <TextField size="small" type="email" placeholder="Enter email" fullWidth />
      </Section>
      <Section>
        <SectionHeader title="Preferences" />
        <FieldLabel label="Theme" />
        <TextField size="small" placeholder="Select theme" fullWidth />
        <FieldLabel label="Language" />
        <TextField size="small" placeholder="Select language" fullWidth />
      </Section>
    </div>
  )
}

export const WithActions: Story = {
  render: () => (
    <Section sx={{ width: 400 }}>
      <SectionHeader title="Settings" subtitle="Configure your preferences" />
      <FieldLabel label="Setting 1" />
      <TextField size="small" placeholder="Value 1" fullWidth />
      <FieldLabel label="Setting 2" />
      <TextField size="small" placeholder="Value 2" fullWidth />
      <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 8 }}>
        <Button variant="outlined">Cancel</Button>
        <Button variant="contained">Save</Button>
      </div>
    </Section>
  )
}
