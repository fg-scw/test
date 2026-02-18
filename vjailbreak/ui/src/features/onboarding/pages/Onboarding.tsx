import RocketIcon from '@mui/icons-material/Rocket'
import TextSnippetIcon from '@mui/icons-material/TextSnippet'
import { Divider, Typography } from '@mui/material'
import Button from '@mui/material/Button'
import { styled } from '@mui/system'
import { useState } from 'react'
import MigrationFormDrawer from '../../migration/MigrationForm'
import GuidesList from './GuidesList'

const OnboardingContainer = styled('div')({
  display: 'flex',
  justifyContent: 'center',
  alignItems: 'center',
  height: '100%'
})

const Container = styled('div')(({ theme }) => ({
  display: 'grid',
  alignItems: 'center',
  justifyItems: 'center',
  gap: theme.spacing(4),
  textAlign: 'center'
}))

const CircularLogoContainer = styled('div')(() => ({
  width: '150px',
  height: '150px',
  borderRadius: '50%',
  overflow: 'hidden',
  display: 'flex',
  justifyContent: 'center',
  alignItems: 'center',
  '& img': {
    width: '100%',
    height: '100%',
    objectFit: 'cover'
  }
}))

const InfoSection = styled('div')(({ theme }) => ({
  display: 'grid',
  gridTemplateColumns: '1fr 1fr',
  gap: theme.spacing(6),
  borderTop: `1px solid ${theme.palette.grey[500]}`,
  padding: theme.spacing(4),
  justifyItems: 'center'
}))

const gettingStartedLinks = [
  { text: 'Quick Start guide to vJailbreak', url: '#' },
  { text: 'Supported VMware and Destination Platforms', url: '#' },
  { text: 'How is Data Copied?', url: '#' },
  { text: 'How is the VM converted?', url: '#' }
]

const advancedTopicsLinks = [
  { text: 'Migrating Workflow: by VM, by Host, by Cluster', url: '#' },
  { text: 'Pre and Post Migration Callouts', url: '#' },
  { text: 'Freeing up ESXi host hardware for KVM', url: '#' }
]

export default function Onboarding() {
  const [openMigrationForm, setOpenMigrationForm] = useState(false)

  return (
    <OnboardingContainer>
      {openMigrationForm && (
        <MigrationFormDrawer open onClose={() => setOpenMigrationForm(false)} />
      )}
      <Container>
        <CircularLogoContainer>
          <img src="/logo.png" alt="vJailbreak Logo" />
        </CircularLogoContainer>
        <Typography variant="h2">vJailbreak</Typography>
        <Typography variant="body1">
          Ready to migrate from VMware to PCD? <br />
          Click below to start moving your VMware workloads to PCD with ease.
        </Typography>
        <Button variant="contained" size="large" onClick={() => setOpenMigrationForm(true)}>
          Start Migration
        </Button>
        <Divider />
        <InfoSection>
          <GuidesList
            listHeader="Getting Started"
            listIcon={<RocketIcon />}
            links={gettingStartedLinks}
          />
          <GuidesList
            listHeader="Advanced Topics"
            listIcon={<TextSnippetIcon />}
            links={advancedTopicsLinks}
          />
        </InfoSection>
      </Container>
    </OnboardingContainer>
  )
}
