import type { Meta, StoryObj } from '@storybook/react'
import { useState } from 'react'

import SectionNav from './SectionNav'

const meta: Meta<typeof SectionNav> = {
  title: 'Components/Design System/SectionNav',
  component: SectionNav,
  parameters: {
    layout: 'padded'
  },
  tags: ['autodocs']
}

export default meta

type Story = StoryObj<typeof SectionNav>

const demoItems = [
  {
    id: 'basic',
    title: 'Basic Info',
    description: 'Name and connectivity details',
    status: 'complete' as const
  },
  {
    id: 'network',
    title: 'Network',
    description: 'VLANs, MTU, and routing',
    status: 'attention' as const
  },
  {
    id: 'storage',
    title: 'Storage',
    description: 'Disks and cache configuration',
    status: 'incomplete' as const
  },
  {
    id: 'advanced',
    title: 'Advanced',
    description: 'Optional tuning',
    status: 'optional' as const
  }
]

export const Interactive: Story = {
  render: (args) => {
    const [activeId, setActiveId] = useState(demoItems[0]?.id)

    return (
      <SectionNav
        {...args}
        items={demoItems}
        activeId={activeId}
        onSelect={(id) => setActiveId(id)}
      />
    )
  }
}

export const Dense: Story = {
  args: {
    dense: true
  },
  render: (args) => {
    const [activeId, setActiveId] = useState(demoItems[1]?.id)

    return (
      <SectionNav
        {...args}
        items={demoItems}
        activeId={activeId}
        onSelect={(id) => setActiveId(id)}
      />
    )
  }
}

export const WithoutDescriptions: Story = {
  args: {
    showDescriptions: false
  },
  render: (args) => {
    const [activeId, setActiveId] = useState(demoItems[2]?.id)

    return (
      <SectionNav
        {...args}
        items={demoItems}
        activeId={activeId}
        onSelect={(id) => setActiveId(id)}
      />
    )
  }
}
