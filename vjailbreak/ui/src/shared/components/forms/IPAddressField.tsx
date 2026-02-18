import { styled } from '@mui/material/styles'
import { TextField, TextFieldProps } from '@mui/material'
import customTypography from 'src/theme/typography'

export const IPAddressField = styled(TextField)<TextFieldProps>(() => ({
  '& .MuiInputBase-input': {
    ...customTypography.monospace
  }
}))

export default IPAddressField
