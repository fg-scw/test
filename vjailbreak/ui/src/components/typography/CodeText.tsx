import { styled } from '@mui/material/styles'
import { Typography, TypographyProps } from '@mui/material'
import customTypography from '../../theme/typography'

export const CodeText = styled(Typography)<TypographyProps>(({ theme }) => ({
  ...customTypography.code,
  backgroundColor: theme.palette.action.hover,
  padding: '2px 6px',
  borderRadius: '4px',
  display: 'inline-block'
}))

export default CodeText
