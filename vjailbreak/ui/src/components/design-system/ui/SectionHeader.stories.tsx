import type { Meta, StoryObj } from '@storybook/react'
import SectionHeader from './SectionHeader'

const meta: Meta<typeof SectionHeader> = {
  title: 'Components/Design System/SectionHeader',
  component: SectionHeader,
  parameters: {
    layout: 'centered'
  },
  tags: ['autodocs']
}

export default meta
type Story = StoryObj<typeof meta>

export const TitleOnly: Story = {
  args: {
    title: 'Section Title'
  }
}

export const WithSubtitle: Story = {
  args: {
    title: 'Section Title',
    subtitle: 'This is a subtitle that provides additional context about the section.'
  }
}

export const LongTitle: Story = {
  args: {
    title:
      'This is a very long section title that demonstrates how the component handles extended text content',
    subtitle:
      'The subtitle can also be quite long to show how both title and subtitle work together in various scenarios.'
  }
}

export const ShortSubtitle: Story = {
  args: {
    title: 'User Settings',
    subtitle: 'Configure your preferences'
  }
}

export const WithoutTitle: Story = {
  args: {
    subtitle: 'This section header only has a subtitle'
  }
}

export const Empty: Story = {
  args: {}
}
