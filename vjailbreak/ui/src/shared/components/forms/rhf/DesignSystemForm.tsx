import { Box } from '@mui/material'
import { BoxProps } from '@mui/material/Box'
import { ReactNode, useMemo } from 'react'
import { FieldValues, FormProvider, SubmitHandler, UseFormReturn } from 'react-hook-form'
import { useKeyboardSubmit } from 'src/hooks/ui/useKeyboardSubmit'

interface KeyboardSubmitProps {
  open: boolean
  onClose: () => void
  isSubmitDisabled?: boolean
}

export interface DesignSystemFormProps<TFieldValues extends FieldValues>
  extends Omit<BoxProps<'form'>, 'component' | 'onSubmit'> {
  form: UseFormReturn<TFieldValues>
  onSubmit: SubmitHandler<TFieldValues>
  children: ReactNode
  keyboardSubmitProps?: KeyboardSubmitProps
}

export default function DesignSystemForm<TFieldValues extends FieldValues>({
  form,
  onSubmit,
  children,
  keyboardSubmitProps,
  ...rest
}: DesignSystemFormProps<TFieldValues>) {
  const submitHandler = useMemo(() => form.handleSubmit(onSubmit), [form, onSubmit])

  if (keyboardSubmitProps) {
    useKeyboardSubmit({
      open: keyboardSubmitProps.open,
      isSubmitDisabled: keyboardSubmitProps.isSubmitDisabled ?? false,
      onSubmit: submitHandler,
      onClose: keyboardSubmitProps.onClose
    })
  }

  return (
    <FormProvider {...form}>
      <Box component="form" onSubmit={submitHandler} noValidate {...rest}>
        {children}
      </Box>
    </FormProvider>
  )
}
