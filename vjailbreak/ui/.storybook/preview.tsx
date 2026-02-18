import type { Preview, StoryFn } from '@storybook/react'
import { CssBaseline, Container } from '@mui/material'
import { ThemeProvider } from '../src/theme/ThemeContext.tsx'

const preview: Preview = {
  parameters: {
    actions: { argTypesRegex: '^on[A-Z].*' },
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/
      }
    },
    options: {
      storySort: {
        order: ['Components', ['Design System', 'Grid', 'Dialogs']]
      }
    }
  },
  decorators: [
    (Story: StoryFn) => (
      <ThemeProvider>
        <CssBaseline />
        <Container maxWidth="md" sx={{ py: 4 }}>
          <Story />
        </Container>
      </ThemeProvider>
    )
  ]
}

export default preview
