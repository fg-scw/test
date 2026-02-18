import { Box, BoxProps } from '@mui/material'

export interface FormGridProps extends BoxProps {
  minWidth?: number
}

export function FormGrid({ minWidth = 320, gap = 2, ...rest }: FormGridProps) {
  return (
    <Box
      display="grid"
      sx={{
        gridTemplateColumns: `repeat(auto-fit, minmax(${minWidth}px, 1fr))`,
        gap
      }}
      {...rest}
    />
  )
}

export default FormGrid
