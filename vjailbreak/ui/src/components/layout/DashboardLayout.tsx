import { Paper, styled } from '@mui/material'
import { Outlet } from 'react-router-dom'
import Sidenav from './Sidenav'
import { navigationItems } from 'src/config/navigation'

const DashboardContainer = styled('div')({
  display: 'flex',
  height: '100vh',
  width: '100%',
  overflow: 'hidden'
})

const ContentContainer = styled('div')(({ theme }) => ({
  flex: 1,
  display: 'flex',
  flexDirection: 'column',
  padding: theme.spacing(2),
  backgroundColor: theme.palette.background.default,
  overflow: 'hidden',
  minHeight: 0
}))

const StyledPaper = styled(Paper)(({ theme }) => ({
  flex: 1,
  display: 'flex',
  flexDirection: 'column',
  overflow: 'auto',
  minHeight: 0,
  border: `1px solid ${theme.palette.divider}`,
  borderRadius: theme.spacing(1.25),
  '& .MuiDataGrid-root': {
    flex: 1,
    border: 'none',
    height: '100%'
  },
  '& .MuiDataGrid-main': {
    overflow: 'hidden'
  },
  '& .MuiDataGrid-virtualScroller': {
    overflow: 'auto !important'
  }
}))

export default function DashboardLayout() {
  return (
    <DashboardContainer>
      <Sidenav items={navigationItems} />
      <ContentContainer>
        <StyledPaper elevation={0}>
          <Outlet />
        </StyledPaper>
      </ContentContainer>
    </DashboardContainer>
  )
}
