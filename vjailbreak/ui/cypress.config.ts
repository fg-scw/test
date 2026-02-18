import { defineConfig } from 'cypress'
import { addMatchImageSnapshotPlugin } from '@simonsmith/cypress-image-snapshot/plugin'

export default defineConfig({
  e2e: {
    baseUrl: 'http://localhost:3000',
    setupNodeEvents(on, config) {
      addMatchImageSnapshotPlugin(on, config)
    },
    viewportWidth: 1280,
    viewportHeight: 720,
    video: false,
    screenshotOnRunFailure: true,
  },
})
