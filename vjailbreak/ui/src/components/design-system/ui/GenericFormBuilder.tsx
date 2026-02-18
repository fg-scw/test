import { Box } from '@mui/material'
import React, { useCallback, useMemo } from 'react'
import { Controller, useForm, useWatch, useFieldArray } from 'react-hook-form'
import {
  DesignSystemForm,
  RHFTextField,
  RHFToggleField,
  RHFSelect,
  RHFCheckbox,
  RHFRadioGroup,
  RHFDateField,
  RHFFileField
} from 'src/shared/components/forms'
import ActionButton from './ActionButton'
import FormGrid from './FormGrid'
import Section from './Section'
import SectionHeader from './SectionHeader'
import InlineHelp from './InlineHelp'
import Row from './Row'

// Design-system primitives used in your codebase. If a field type isn't available
// in your design-system, the builder falls back to `RHFTextField` so you can
// gradually adopt it.

/* -------------------------------------------------------------------------- */
/*                                 Types                                      */
/* -------------------------------------------------------------------------- */

export type Option = { label: string; value: any }
export type FieldType =
  | 'text'
  | 'password'
  | 'number'
  | 'textarea'
  | 'select'
  | 'checkbox'
  | 'switch'
  | 'radio'
  | 'date'
  | 'file'
  | 'custom'
  | 'array'

export type VisibilityCondition = { field: string; equals?: any } | ((values: any) => boolean)

export interface BaseFieldConfig {
  name: string
  label?: string
  type?: FieldType
  placeholder?: string
  description?: string
  rules?: Record<string, any>
  defaultValue?: any
  disabled?: boolean
  hidden?: boolean
  visible?: VisibilityCondition
  grid?: number
  render?: (props: {
    value: any
    onChange: (val: any) => void
    onBlur: () => void
    fieldState: { error?: any }
    formState: any
  }) => React.ReactNode
  options?: Option[] | ((values: any) => Option[])
  loadOptions?: (values: any) => Promise<Option[]>
}

export interface ArrayFieldConfig extends BaseFieldConfig {
  type: 'array'
  children: FieldConfig[]
  minItems?: number
  maxItems?: number
  addLabel?: string
}

export type FieldConfig = BaseFieldConfig | ArrayFieldConfig

export interface SectionConfig {
  id?: string
  title?: string
  subtitle?: string
  description?: string
  fields: FieldConfig[]
}

export interface FormBuilderProps {
  id?: string
  sections?: SectionConfig[]
  fields?: FieldConfig[] // legacy single-section support
  onSubmit: (values: any) => Promise<void> | void
  onCancel?: () => void
  submitLabel?: string
  defaultValues?: any
  showReset?: boolean
  verticalSpacing?: number
  columnCount?: number
  submitButtonProps?: Omit<React.ComponentProps<typeof ActionButton>, 'onClick'>
  showErrors?: boolean
}

/* -------------------------------------------------------------------------- */
/*                               Utilities                                    */
/* -------------------------------------------------------------------------- */

const isArrayField = (f: FieldConfig): f is ArrayFieldConfig => (f as any).type === 'array'

function evaluateVisibility(condition: VisibilityCondition | undefined, values: any) {
  if (condition === undefined) return true
  if (typeof condition === 'function') return condition(values)
  return values && (values as any)[condition.field] === (condition as any).equals
}

/* -------------------------------------------------------------------------- */
/*                           GenericFormBuilder                              */
/* -------------------------------------------------------------------------- */

/**
 * GenericFormBuilder adapted to your design-system
 * - Uses DesignSystemForm and RHF* primitives from your codebase
 * - Supports: sections/sub-headings, conditional visibility, arrays, custom renderers,
 *   async options, file & date fields, and fallback behavior so adoption is incremental.
 */
export default function GenericFormBuilder(props: FormBuilderProps) {
  const {
    id,
    sections,
    fields,
    onSubmit,
    onCancel,
    submitLabel = 'Save',
    defaultValues = {},
    showReset = false,
    verticalSpacing = 3,
    columnCount = 2,
    submitButtonProps = {},
    showErrors = true
  } = props

  const form = useForm({ defaultValues: defaultValues })
  const { control, handleSubmit, reset, formState } = form
  const { isSubmitting, isSubmitSuccessful, errors } = formState

  const watched = useWatch({ control })

  const onFormSubmit = useCallback(
    async (values: any) => {
      await onSubmit(values)
    },
    [onSubmit]
  )

  // Flatten either sections or single fields into a canonical structure
  const normalizedSections: SectionConfig[] = useMemo(() => {
    if (sections && sections.length > 0) return sections
    return [
      {
        id: 'main',
        title: undefined,
        subtitle: undefined,
        description: undefined,
        fields: fields ?? []
      }
    ]
  }, [sections, fields])

  const renderField = (field: FieldConfig, parentName?: string) => {
    const name = parentName ? `${parentName}.${field.name}` : field.name

    if (field.hidden) return null

    const visible = evaluateVisibility(field.visible, watched)
    if (!visible) return null

    if (isArrayField(field)) {
      return (
        <Box key={name} mb={2}>
          <ArrayFieldRenderer
            field={field}
            name={name}
            control={control}
            columnCount={columnCount}
          />
        </Box>
      )
    }

    return (
      <Box key={name} mb={2}>
        <FieldRenderer control={control} field={field} name={name} watched={watched} />
      </Box>
    )
  }

  return (
    <DesignSystemForm form={form} id={id} onSubmit={handleSubmit(onFormSubmit as any)}>
      <Box display="flex" flexDirection="column" gap={verticalSpacing}>
        {normalizedSections.map((section) => (
          <Section key={section.id ?? section.title}>
            {(section.title || section.subtitle) && (
              <SectionHeader title={section.title} subtitle={section.subtitle} />
            )}

            {section.description && <InlineHelp>{section.description}</InlineHelp>}

            <FormGrid minWidth={columnCount === 1 ? 400 : 320} gap={2}>
              {section.fields.map((f) => renderField(f))}
            </FormGrid>
          </Section>
        ))}

        {showErrors && Object.keys(errors).length > 0 && (
          <Box>
            <InlineHelp tone="critical">
              Please fix the highlighted errors before submitting.
            </InlineHelp>
          </Box>
        )}

        <Row justifyContent="flex-end">
          {showReset && (
            <ActionButton
              tone="secondary"
              onClick={() => reset(defaultValues)}
              disabled={isSubmitting}
            >
              Reset
            </ActionButton>
          )}

          {onCancel && (
            <ActionButton tone="secondary" onClick={onCancel} disabled={isSubmitting}>
              Cancel
            </ActionButton>
          )}

          <ActionButton type="submit" tone="primary" loading={isSubmitting} {...submitButtonProps}>
            {submitLabel}
          </ActionButton>
        </Row>

        {isSubmitSuccessful && <InlineHelp tone="positive">Submitted successfully.</InlineHelp>}
      </Box>
    </DesignSystemForm>
  )
}

/* -------------------------------------------------------------------------- */
/*                              FieldRenderer                                 */
/* -------------------------------------------------------------------------- */

function FieldRenderer(props: { control: any; field: FieldConfig; name: string; watched: any }) {
  const { control, field, name, watched } = props

  const options: Option[] | undefined = useMemo(() => {
    if (!field.options) return undefined
    if (typeof field.options === 'function') return (field.options as any)(watched)
    return field.options
  }, [field.options, watched])

  const [asyncOptions, setAsyncOptions] = React.useState<Option[] | null>(
    field.loadOptions ? null : (options ?? [])
  )
  React.useEffect(() => {
    let mounted = true
    if (field.loadOptions) {
      field
        .loadOptions(watched)
        .then((res) => mounted && setAsyncOptions(res))
        .catch(() => mounted && setAsyncOptions([]))
    }
    return () => {
      mounted = false
    }
  }, [field, watched])

  const finalOptions = asyncOptions ?? options

  const renderInner = () => {
    const t = field.type ?? 'text'

    switch (t) {
      case 'text':
      case 'password':
      case 'number':
      case 'textarea':
        return (
          <Controller
            control={control}
            name={name as any}
            defaultValue={(field as any).defaultValue ?? ''}
            rules={field.rules}
            render={({ field: ctrlField }) => (
              <Box>
                <RHFTextField {...ctrlField} />
              </Box>
            )}
          />
        )

      case 'select':
        return (
          <Controller
            control={control}
            name={name as any}
            defaultValue={(field as any).defaultValue ?? ''}
            rules={field.rules}
            render={({ field: ctrlField }) => (
              <Box>
                <RHFSelect {...ctrlField} options={finalOptions ?? []} />
              </Box>
            )}
          />
        )

      case 'checkbox':
        return (
          <Controller
            control={control}
            name={name as any}
            defaultValue={(field as any).defaultValue ?? false}
            rules={field.rules}
            render={({ field: ctrlField }) => (
              <Box>
                <RHFCheckbox {...ctrlField} />
              </Box>
            )}
          />
        )

      case 'switch':
        return (
          <Controller
            control={control}
            name={name as any}
            defaultValue={(field as any).defaultValue ?? false}
            rules={field.rules}
            render={({ field: ctrlField }) => (
              <Box>
                <RHFToggleField {...ctrlField} label={field.label} />
              </Box>
            )}
          />
        )

      case 'radio':
        return (
          <Controller
            control={control}
            name={name as any}
            defaultValue={(field as any).defaultValue ?? ''}
            rules={field.rules}
            render={({ field: ctrlField }) => (
              <Box>
                <RHFRadioGroup {...ctrlField} options={finalOptions ?? []} />
              </Box>
            )}
          />
        )

      case 'date':
        return (
          <Controller
            control={control}
            name={name as any}
            defaultValue={(field as any).defaultValue ?? ''}
            rules={field.rules}
            render={({ field: ctrlField }) => (
              <Box>
                <RHFDateField {...ctrlField} />
              </Box>
            )}
          />
        )

      case 'file':
        return (
          <Controller
            control={control}
            name={name as any}
            defaultValue={(field as any).defaultValue ?? null}
            rules={field.rules}
            render={({ field: ctrlField }) => (
              <Box>
                <RHFFileField {...ctrlField} accept={(field as any).accept} />
              </Box>
            )}
          />
        )

      case 'custom':
        if (!field.render) return null
        return (
          <Controller
            control={control}
            name={name as any}
            defaultValue={(field as any).defaultValue}
            rules={field.rules}
            render={({ field: ctrlField, fieldState }) => (
              <Box>
                {field.render &&
                  field.render({
                    value: ctrlField.value,
                    onChange: ctrlField.onChange,
                    onBlur: ctrlField.onBlur,
                    fieldState,
                    formState: {}
                  })}
              </Box>
            )}
          />
        )

      default:
        return null
    }
  }

  return renderInner()
}

/* -------------------------------------------------------------------------- */
/*                              ArrayFieldRenderer                            */
/* -------------------------------------------------------------------------- */

function ArrayFieldRenderer(props: {
  field: ArrayFieldConfig
  name: string
  control: any
  columnCount: number
}) {
  const { field, name, control, columnCount } = props
  const { fields: arrayFields, append, remove } = useFieldArray({ control, name })

  React.useEffect(() => {
    if (arrayFields.length === 0 && field.minItems && field.minItems > 0) {
      for (let i = 0; i < field.minItems; i++) append({})
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={1}>
        <SectionHeader title={field.label} />
        <ActionButton
          size="small"
          onClick={() => append({})}
          disabled={field.maxItems ? arrayFields.length >= field.maxItems : false}
        >
          {field.addLabel ?? 'Add'}
        </ActionButton>
      </Box>

      <Box display="flex" flexDirection="column" gap={2}>
        {arrayFields.map((item, idx) => (
          <Box key={item.id} p={2} borderRadius={1} border={1} borderColor="divider">
            <Box display="flex" justifyContent="flex-end" mb={1}>
              <ActionButton
                size="small"
                onClick={() => remove(idx)}
                disabled={arrayFields.length <= (field.minItems ?? 0)}
              >
                Remove
              </ActionButton>
            </Box>

            <FormGrid minWidth={columnCount === 1 ? 400 : 320} gap={2}>
              {field.children.map((child) => (
                <Box key={`${name}.${idx}.${child.name}`}>
                  {/* nested dotted path */}
                  <FieldRenderer
                    control={control}
                    field={child}
                    name={`${name}.${idx}.${child.name}`}
                    watched={{} as any}
                  />
                </Box>
              ))}
            </FormGrid>
          </Box>
        ))}
      </Box>
    </Box>
  )
}

/* -------------------------------------------------------------------------- */
/*                             End of File                                    */
/* -------------------------------------------------------------------------- */

/*
  Notes for integration / adoption:

  1. This builder expects the following design-system primitives to exist:
     - DesignSystemForm (wraps react-hook-form provider)
     - RHFTextField, RHFToggleField, RHFSelect, RHFCheckbox, RHFRadioGroup, RHFFileField, RHFDateField
     - FormGrid, Section, SectionHeader, InlineHelp, Row, Col, ActionButton

  2. Fallbacks: Where a specific RHF primitive isn't available, the code uses
     RHFTextField fallbacks so you can progressively implement design-system fields.

  3. Sections: Pass `sections` to the builder for multi-section forms with titles,
     subtitles and descriptions. For single-section forms you can still pass `fields`.

  4. Extensibility: To add a new field type, implement the renderer case in
     FieldRenderer and add a design-system primitive if needed.

  5. Types: The builder is generic over `TValues`. For stricter typing, pass a
     concrete interface for your form values when using the component.

  */
