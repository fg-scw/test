import { Alert, Checkbox, FormControlLabel, MenuItem, Select, styled } from '@mui/material'
import customTypography from '../../theme/typography'
import { AdapterDayjs } from '@mui/x-date-pickers/AdapterDayjs'
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker'
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider'
import dayjs from 'dayjs'
import { useCallback, useEffect } from 'react'
import { Step, TextField } from 'src/shared/components/forms'
import { FieldErrors, FormValues, SelectedMigrationOptionsType } from './MigrationForm'

// Accordian Imports
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import Accordion from '@mui/material/Accordion'
import AccordionDetails from '@mui/material/AccordionDetails'
import AccordionSummary from '@mui/material/AccordionSummary'
import { CUTOVER_TYPES, DATA_COPY_OPTIONS, VM_CUTOVER_OPTIONS } from './constants'

// Styles
const FieldsContainer = styled('div')({})

const Fields = styled('div')(({ theme }) => ({
  display: 'grid',
  gridTemplateColumns: '1fr 2fr 1fr',
  gridGap: '16px', // Adds spacing between the columns
  marginTop: theme.spacing(2)
}))

const CustomTextField = styled(TextField)(() => ({
  '& .MuiOutlinedInput-root': {
    // Use monospace variant for input fields (larger, more readable)
    ...customTypography.monospace
  }
}))

// Intefaces
interface MigrationOptionsPropsInterface {
  params: FormValues
  onChange: (key: string) => (value: unknown) => void
  selectedMigrationOptions: SelectedMigrationOptionsType
  updateSelectedMigrationOptions: (
    key: keyof SelectedMigrationOptionsType
  ) => (value: unknown) => void
  errors: FieldErrors
  getErrorsUpdater: (key: string | number) => (value: string) => void
  showHeader?: boolean
}

// TODO - Commented out the non-required field from the options for now
// const PrePostWebHooksList = [
//   { label: "Pre data-copy web hook", identifier: "preDataCopyWebHook" },
//   { label: "Post data-copy web hook", identifier: "postDataCopyWebHook" },
//   { label: "Pre cutover web hook", identifier: "preCutoverWebHook" },
//   { label: "Post cutover web hook", identifier: "postCutoverWebHook" },
// ]

export default function MigrationOptions({
  params,
  onChange,
  selectedMigrationOptions,
  updateSelectedMigrationOptions,
  errors,
  getErrorsUpdater,
  showHeader = true
}: MigrationOptionsPropsInterface) {
  // Iniitialize fields
  useEffect(() => {
    onChange('dataCopyMethod')('cold')
    onChange('cutoverOption')(CUTOVER_TYPES.IMMEDIATE)
  }, [])

  const isPowerOffThenCopy = (params?.dataCopyMethod || 'cold') === 'cold'

  useEffect(() => {
    if (!isPowerOffThenCopy) return

    if (selectedMigrationOptions.cutoverOption) {
      updateSelectedMigrationOptions('cutoverOption')(false)
    }

    onChange('cutoverStartTime')('')
    onChange('cutoverEndTime')('')
  }, [
    isPowerOffThenCopy,
    selectedMigrationOptions.cutoverOption,
    onChange,
    updateSelectedMigrationOptions
  ])

  const getMinEndTime = useCallback(() => {
    let minDate = params.cutoverStartTime
    if (selectedMigrationOptions.dataCopyStartTime) {
      // Which ever is greater
      minDate =
        dayjs(params.cutoverStartTime).diff(dayjs(params.dataCopyStartTime), 'seconds') > 0
          ? params.cutoverStartTime
          : params.dataCopyStartTime
    }

    // Disabled selection of time in the past
    const computedMin = dayjs(minDate).add(1, 'minute')
    const now = dayjs()
    return computedMin.isAfter(now) ? computedMin : now
  }, [params, selectedMigrationOptions])

  return (
    <LocalizationProvider dateAdapter={AdapterDayjs}>
      <Accordion
        sx={{
          boxShadow: 'none', // Removes box shadow
          border: 'none', // Removes border
          '&:before': {
            display: 'none' // Removes the default divider line before the accordion
          }
        }}
        defaultExpanded
      >
        <AccordionSummary
          expandIcon={<ExpandMoreIcon />}
          aria-controls="panel2-content"
          id="panel2-header"
        >
          {showHeader ? (
            <Step stepNumber="4" label="Migration Options (Optional)" sx={{ mb: '0' }} />
          ) : null}
        </AccordionSummary>
        <AccordionDetails>
          <FieldsContainer>
            {/* Data Copy */}
            <Fields>
              <FormControlLabel
                id="data-copy-method"
                label="Data Copy Method"
                control={
                  <Checkbox
                    checked={selectedMigrationOptions.dataCopyMethod}
                    onChange={(e) => {
                      const isChecked = e.target.checked
                      updateSelectedMigrationOptions('dataCopyMethod')(isChecked)
                      if (!isChecked) {
                        onChange('dataCopyMethod')('cold')
                      }
                    }}
                  />
                }
              />
              <Select
                size="small"
                disabled={!selectedMigrationOptions.dataCopyMethod}
                labelId="source-item-label"
                value={params?.dataCopyMethod || 'cold'}
                onChange={(e) => {
                  onChange('dataCopyMethod')(e.target.value)
                }}
              >
                {DATA_COPY_OPTIONS.map((item) => (
                  <MenuItem key={item.value} value={item.value}>
                    {item.label}
                  </MenuItem>
                ))}
              </Select>
            </Fields>

            {/* Data Copy Time Window */}
            <Fields>
              <FormControlLabel
                label={'Data Copy Start Time'}
                control={
                  <Checkbox
                    checked={selectedMigrationOptions?.dataCopyStartTime}
                    onChange={(e) => {
                      const isChecked = e.target.checked
                      updateSelectedMigrationOptions('dataCopyStartTime')(isChecked)
                      if (!isChecked) {
                        onChange('dataCopyStartTime')('')
                      }
                    }}
                  />
                }
              />
              <TimePicker
                label="Data Copy Start Time"
                identifier="dataCopyStartTime"
                params={params}
                errors={errors}
                getErrorsUpdater={getErrorsUpdater}
                onChange={onChange}
                disabled={!selectedMigrationOptions.dataCopyStartTime}
                required={!!selectedMigrationOptions.dataCopyStartTime}
                disablePast
              />
            </Fields>

            {/* Cutover settings*/}
            <Fields>
              <FormControlLabel
                id="data-copy-method"
                label="Cutover Options"
                control={
                  <Checkbox
                    checked={selectedMigrationOptions.cutoverOption}
                    disabled={isPowerOffThenCopy}
                    onChange={(e) => {
                      const isChecked = e.target.checked
                      updateSelectedMigrationOptions('cutoverOption')(isChecked)
                      if (!isChecked) {
                        onChange('cutoverOption')(CUTOVER_TYPES.IMMEDIATE)
                        onChange('cutoverStartTime')('')
                        onChange('cutoverEndTime')('')
                      }
                    }}
                  />
                }
              />
              <Select
                size="small"
                disabled={isPowerOffThenCopy || !selectedMigrationOptions?.cutoverOption}
                value={params?.cutoverOption || CUTOVER_TYPES.IMMEDIATE}
                onChange={(e) => {
                  onChange('cutoverOption')(e.target.value)
                }}
              >
                {VM_CUTOVER_OPTIONS.map((item) => (
                  <MenuItem key={item.value} value={item.value}>
                    {item.label}
                  </MenuItem>
                ))}
              </Select>
            </Fields>

            {isPowerOffThenCopy ? (
              <Alert severity="info" sx={{ mt: 2 }}>
                Cutover options are disabled for cold migration (Power off then copy) because the VM
                is powered off before copying. Cutover happens automatically after data copy
                completes.
              </Alert>
            ) : null}

            {params.cutoverOption === CUTOVER_TYPES.TIME_WINDOW &&
              selectedMigrationOptions.cutoverOption &&
              !isPowerOffThenCopy && (
                <Fields sx={{ mt: '20px', gridTemplateColumns: '1fr 1fr 1fr' }}>
                  <TimePicker
                    label="Cutover Start Time"
                    identifier="cutoverStartTime"
                    params={params}
                    errors={errors}
                    getErrorsUpdater={getErrorsUpdater}
                    onChange={onChange}
                    sx={{ ml: '32px' }}
                    required={params.cutoverOption === CUTOVER_TYPES.TIME_WINDOW}
                    disablePast
                  />
                  <TimePicker
                    label="Cutover End Time"
                    identifier="cutoverEndTime"
                    params={params}
                    errors={errors}
                    getErrorsUpdater={getErrorsUpdater}
                    onChange={onChange}
                    required={params.cutoverOption === CUTOVER_TYPES.TIME_WINDOW}
                    minDateTime={getMinEndTime()}
                    disablePast
                    helperText="Should be greater than data copy/cutover start time"
                  />
                </Fields>
              )}

            <Fields>
              <FormControlLabel
                label="Post Migration Script"
                control={
                  <Checkbox
                    checked={selectedMigrationOptions.postMigrationScript}
                    onChange={(e) => {
                      const isChecked = e.target.checked
                      updateSelectedMigrationOptions('postMigrationScript')(isChecked)
                      if (!isChecked) {
                        onChange('postMigrationScript')('')
                      }
                    }}
                  />
                }
              />
              <CustomTextField
                label="Post Migration Script"
                size="small"
                multiline
                rows={4}
                value={params?.postMigrationScript || ''}
                onChange={(e) => onChange('postMigrationScript')(String(e.target.value))}
                disabled={!selectedMigrationOptions.postMigrationScript}
                error={!!errors['postMigrationScript']}
                required={selectedMigrationOptions.postMigrationScript}
                placeholder="Enter your post-migration script here..."
              />
            </Fields>

            {/* Pre and Post Web Hooks */}
            {/* {PrePostWebHooksList.map((hook) => (
              <Fields key={`${hook.label}-${hook.identifier}`}>
                <PrePostWebHooks
                  label={hook.label}
                  identifier={hook.identifier}
                  params={params}
                  selectedMigrationOptions={selectedMigrationOptions}
                  updateSelectedMigrationOptions={updateSelectedMigrationOptions}
                  onChange={onChange}
                />
              </Fields>
            ))} */}
          </FieldsContainer>
        </AccordionDetails>
      </Accordion>
    </LocalizationProvider>
  )
}

const TimePicker = ({
  identifier,
  params,
  onChange,
  errors,
  getErrorsUpdater,
  helperText = '',
  ...restProps
}) => {
  const value = params?.[identifier] ? dayjs(params?.[identifier]) : null

  const handleTimeChange = useCallback(
    (newValue: dayjs.Dayjs | null, identifier) => {
      // Use format() with timezone offset instead of toISOString() which converts to UTC
      // This preserves the user's local timezone (e.g., "2025-11-20T12:40:00+05:30" for IST)
      const formattedTime = newValue?.format() ?? ''
      onChange(identifier)(formattedTime)
    },
    [onChange]
  )

  return (
    <DateTimePicker
      {...restProps}
      ampm={false}
      value={value}
      onChange={(newValue: dayjs.Dayjs | null) => handleTimeChange(newValue, identifier)}
      onError={(error) => {
        getErrorsUpdater(identifier)(error)
      }}
      slots={{
        textField: (props) => (
          <TextField
            {...props}
            size="small"
            required={restProps?.required}
            error={!!errors[identifier] && !restProps?.disabled} // Show error if validation fails
            helperText={!!errors[identifier] && !restProps?.disabled ? helperText : ''}
          />
        )
      }}
    />
  )
}

// const PrePostWebHooks = ({
//   label,
//   identifier,
//   params,
//   onChange,
//   selectedMigrationOptions,
//   updateSelectedMigrationOptions,
// }) => {
//   return (
//     <>
//       <FormControlLabel
//         label={label}
//         control={
//           <Checkbox
//             checked={selectedMigrationOptions?.[identifier]}
//             onChange={(e) => {
//               updateSelectedMigrationOptions(identifier)(e.target.checked)
//             }}
//           />
//         }
//       />
//       <CustomTextField
//         value={params?.[identifier] || ""}
//         onChange={(e) => onChange(identifier)(String(e.target.value))}
//       />
//     </>
//   )
// }
