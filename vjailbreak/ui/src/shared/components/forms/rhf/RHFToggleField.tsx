import { Controller, ControllerProps, FieldValues, useFormContext } from 'react-hook-form'
import { ToggleField, ToggleFieldProps } from 'src/components'

type ToggleFieldControlledProps = Omit<ToggleFieldProps, 'onChange' | 'checked' | 'name'>

export type RHFToggleFieldProps = ToggleFieldControlledProps & {
  name: ControllerProps<FieldValues>['name']
}

export default function RHFToggleField({ name, ...rest }: RHFToggleFieldProps) {
  const { control } = useFormContext()

  return (
    <Controller
      name={name}
      control={control}
      render={({ field }) => (
        <ToggleField
          {...rest}
          name={field.name}
          checked={!!field.value}
          onChange={(_, checked) => {
            field.onChange(checked)
          }}
        />
      )}
    />
  )
}
