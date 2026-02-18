import { Drawer, styled } from '@mui/material'

export const StyledDrawer = styled(Drawer)(({ theme }) => ({
  '& .MuiDrawer-paper': {
    display: 'grid',
    gridTemplateRows: 'max-content 1fr max-content',
    width: '800px',
    backgroundColor: theme.palette.background.paper,
    borderLeft: `1px solid ${theme.palette.divider}`
  }
}))

export const DrawerContent = styled('div')(({ theme }) => ({
  overflow: 'auto',
  padding: theme.spacing(4, 6, 4, 4)
}))
