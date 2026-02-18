import type { Meta, StoryObj } from '@storybook/react'
import { Box, Stack, Typography } from '@mui/material'
import { useState } from 'react'
import FileDropzone from './FileDropzone'

const meta: Meta<typeof FileDropzone> = {
  title: 'Components/Design System/FileDropzone',
  component: FileDropzone,
  args: {
    placeholder: 'Drag and drop a file here',
    helperText: 'or click to browse',
    caption: 'Supported formats: .tar, .tar.gz (max 500MB)'
  },
  parameters: {
    layout: 'centered'
  }
}

export default meta

type Story = StoryObj<typeof FileDropzone>

function StatefulWrapper(args: React.ComponentProps<typeof FileDropzone>) {
  const [file, setFile] = useState<File | null>(null)

  return (
    <Stack spacing={2} sx={{ width: 640 }}>
      <FileDropzone
        {...args}
        file={file}
        onFileSelected={(next) => {
          setFile(next)
          args.onFileSelected(next)
        }}
      />

      <Box>
        <Typography variant="caption" color="text.secondary">
          Selected: {file ? `${file.name} (${file.size} bytes)` : 'None'}
        </Typography>
      </Box>
    </Stack>
  )
}

export const Default: Story = {
  render: (args) => (
    <StatefulWrapper
      {...args}
      onFileSelected={() => {
        // noop for story, wrapper owns state
      }}
    />
  )
}

export const Disabled: Story = {
  args: {
    disabled: true
  },
  render: (args) => (
    <StatefulWrapper
      {...args}
      onFileSelected={() => {
        // noop for story, wrapper owns state
      }}
    />
  )
}

export const CustomCopy: Story = {
  args: {
    placeholder: 'Drop your config bundle here',
    helperText: 'or browse from your computer',
    caption: 'Example: config.tar.gz'
  },
  render: (args) => (
    <StatefulWrapper
      {...args}
      onFileSelected={() => {
        // noop for story, wrapper owns state
      }}
    />
  )
}
