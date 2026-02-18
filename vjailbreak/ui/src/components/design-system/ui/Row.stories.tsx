import type { Meta, StoryObj } from '@storybook/react'
import { Box, Button } from '@mui/material'
import Row from './Row'

const meta: Meta<typeof Row> = {
  title: 'Components/Design System/Row',
  component: Row,
  parameters: {
    layout: 'centered'
  },
  tags: ['autodocs']
}

export default meta
type Story = StoryObj<typeof meta>

// Helper component for visual demonstration
const DemoBox = ({
  children,
  color = '#1976d2',
  sx
}: {
  children: React.ReactNode
  color?: string
  sx?: any
}) => (
  <Box
    sx={{
      p: 2,
      backgroundColor: color,
      color: 'white',
      borderRadius: 1,
      minWidth: 100,
      textAlign: 'center',
      ...(sx ?? {})
    }}
  >
    {children}
  </Box>
)

export const Default: Story = {
  render: () => (
    <Row>
      <DemoBox>Item 1</DemoBox>
      <DemoBox color="#2e7d32">Item 2</DemoBox>
      <DemoBox color="#ed6c02">Item 3</DemoBox>
    </Row>
  )
}

export const JustifyContentStart: Story = {
  render: () => (
    <Row justifyContent="flex-start" sx={{ width: 500, border: '1px dashed #ccc', p: 2 }}>
      <DemoBox>Start</DemoBox>
      <DemoBox color="#2e7d32">Item</DemoBox>
    </Row>
  )
}

export const JustifyContentCenter: Story = {
  render: () => (
    <Row justifyContent="center" sx={{ width: 500, border: '1px dashed #ccc', p: 2 }}>
      <DemoBox>Center</DemoBox>
      <DemoBox color="#2e7d32">Item</DemoBox>
    </Row>
  )
}

export const JustifyContentEnd: Story = {
  render: () => (
    <Row justifyContent="flex-end" sx={{ width: 500, border: '1px dashed #ccc', p: 2 }}>
      <DemoBox>End</DemoBox>
      <DemoBox color="#2e7d32">Item</DemoBox>
    </Row>
  )
}

export const JustifyContentSpaceBetween: Story = {
  render: () => (
    <Row justifyContent="space-between" sx={{ width: 500, border: '1px dashed #ccc', p: 2 }}>
      <DemoBox>Left</DemoBox>
      <DemoBox color="#2e7d32">Right</DemoBox>
    </Row>
  )
}

export const AlignItemsStart: Story = {
  render: () => (
    <Row alignItems="flex-start" sx={{ height: 150, border: '1px dashed #ccc', p: 2 }}>
      <DemoBox>Top</DemoBox>
      <DemoBox color="#2e7d32" sx={{ height: 80 }}>
        Taller
      </DemoBox>
      <DemoBox color="#ed6c02">Bottom</DemoBox>
    </Row>
  )
}

export const AlignItemsCenter: Story = {
  render: () => (
    <Row alignItems="center" sx={{ height: 150, border: '1px dashed #ccc', p: 2 }}>
      <DemoBox>Center</DemoBox>
      <DemoBox color="#2e7d32" sx={{ height: 80 }}>
        Taller
      </DemoBox>
      <DemoBox color="#ed6c02">Center</DemoBox>
    </Row>
  )
}

export const AlignItemsEnd: Story = {
  render: () => (
    <Row alignItems="flex-end" sx={{ height: 150, border: '1px dashed #ccc', p: 2 }}>
      <DemoBox>Bottom</DemoBox>
      <DemoBox color="#2e7d32" sx={{ height: 80 }}>
        Taller
      </DemoBox>
      <DemoBox color="#ed6c02">Bottom</DemoBox>
    </Row>
  )
}

export const CustomGap: Story = {
  render: () => (
    <Row gap={4}>
      <DemoBox>Gap 4</DemoBox>
      <DemoBox color="#2e7d32">Gap 4</DemoBox>
      <DemoBox color="#ed6c02">Gap 4</DemoBox>
    </Row>
  )
}

export const WithButtons: Story = {
  render: () => (
    <Row justifyContent="flex-end" gap={2}>
      <Button variant="outlined">Cancel</Button>
      <Button variant="contained">Save</Button>
    </Row>
  )
}
