import {
  Alert,
  Box,
  Checkbox,
  Divider,
  FormControlLabel,
  MenuItem,
  Select,
  styled,
  Typography
} from '@mui/material'
import customTypography from '../../theme/typography'
import { AdapterDayjs } from '@mui/x-date-pickers/AdapterDayjs'
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider'
import dayjs from 'dayjs'
import { useCallback, useEffect } from 'react'
import { useFormContext } from 'react-hook-form'
import { RHFDateTimeField, RHFTextField, Step, TextField } from 'src/shared/components/forms'
import { FieldErrors, FormValues, SelectedMigrationOptionsType } from './MigrationForm'
import { OpenstackCreds } from 'src/api/openstack-creds/model'
import { CUTOVER_TYPES, DATA_COPY_OPTIONS, VM_CUTOVER_OPTIONS } from './constants'
import { IntervalField } from 'src/shared/components/forms'
import { useSettingsConfigMapQuery } from 'src/hooks/api/useSettingsConfigMapQuery'

const SectionBlock = styled(Box)(({ theme }) => ({
  display: 'grid',
  gap: theme.spacing(1.25)
}))

const SectionHeaderRow = styled(Box)(({ theme }) => ({
  display: 'flex',
  alignItems: 'baseline',
  justifyContent: 'space-between',
  gap: theme.spacing(2),
  marginBottom: theme.spacing(1)
}))

const OptionRow = styled(Box)(({ theme }) => ({
  display: 'grid',
  gridTemplateColumns: 'minmax(240px, 1fr) minmax(240px, 1fr)',
  gap: theme.spacing(2),
  alignItems: 'start',
  padding: `${theme.spacing(1)} 0`,
  [theme.breakpoints.down('md')]: {
    gridTemplateColumns: '1fr'
  }
}))

const OptionLeft = styled(Box)(({ theme }) => ({
  display: 'grid',
  gap: theme.spacing(0.5)
}))

const OptionHelp = styled(Typography)(({ theme }) => ({
  color: theme.palette.text.secondary,
  marginLeft: 32
}))

const CustomTextField = styled(TextField)(() => ({
  '& .MuiOutlinedInput-root': {
    ...customTypography.monospace
  }
}))

// Interfaces
export interface MigrationOptionsPropsInterface {
  params: FormValues & { useFlavorless?: boolean; useGPU?: boolean }
  onChange: (key: string) => (value: unknown) => void
  openstackCredentials?: OpenstackCreds
  selectedMigrationOptions: SelectedMigrationOptionsType
  updateSelectedMigrationOptions: (
    key:
      | keyof SelectedMigrationOptionsType
      | 'postMigrationAction.suffix'
      | 'postMigrationAction.folderName'
  ) => (value: unknown) => void

  errors: FieldErrors
  getErrorsUpdater: (key: string | number) => (value: string) => void
  stepNumber: string
  showHeader?: boolean
}

// TODO - Commented out the non-required field from the options for now
// const PrePostWebHooksList = [
//   { label: "Pre data-copy web hook", identifier: "preDataCopyWebHook" },
//   { label: "Post data-copy web hook", identifier: "postDataCopyWebHook" },
//   { label: "Pre cutover web hook", identifier: "preCutoverWebHook" },
//   { label: "Post cutover web hook", identifier: "postCutoverWebHook" },
// ]

export default function MigrationOptionsAlt({
  params,
  onChange,
  selectedMigrationOptions,
  openstackCredentials,
  updateSelectedMigrationOptions,
  errors,
  getErrorsUpdater,
  stepNumber,
  showHeader = true
}: MigrationOptionsPropsInterface) {
  const { setValue } = useFormContext()
  const { data: globalConfigMap } = useSettingsConfigMapQuery()

  const isStorageAcceleratedCopy = params?.storageCopyMethod === 'StorageAcceleratedCopy'

  // Iniitialize fields
  useEffect(() => {
    const defaultMethod = globalConfigMap?.data?.DEFAULT_MIGRATION_METHOD || 'cold'
    onChange('dataCopyMethod')(defaultMethod)
    onChange('cutoverOption')(CUTOVER_TYPES.IMMEDIATE)
  }, [globalConfigMap?.data?.DEFAULT_MIGRATION_METHOD, onChange])

  useEffect(() => {
    if (!isStorageAcceleratedCopy) return

    if (selectedMigrationOptions.dataCopyMethod) {
      updateSelectedMigrationOptions('dataCopyMethod')(false)
    }
    if (selectedMigrationOptions.dataCopyStartTime) {
      updateSelectedMigrationOptions('dataCopyStartTime')(false)
    }
    if (selectedMigrationOptions.cutoverOption) {
      updateSelectedMigrationOptions('cutoverOption')(false)
    }
    if (selectedMigrationOptions.periodicSyncEnabled) {
      updateSelectedMigrationOptions('periodicSyncEnabled')(false)
    }

    onChange('dataCopyStartTime')('')
    onChange('cutoverStartTime')('')
    onChange('cutoverEndTime')('')
    onChange('periodicSyncInterval')('')
  }, [
    isStorageAcceleratedCopy,
    onChange,
    selectedMigrationOptions.cutoverOption,
    selectedMigrationOptions.dataCopyMethod,
    selectedMigrationOptions.dataCopyStartTime,
    selectedMigrationOptions.periodicSyncEnabled,
    updateSelectedMigrationOptions
  ])

  const isPowerOffThenCopy = (params?.dataCopyMethod || 'cold') === 'cold'

  useEffect(() => {
    if (!isPowerOffThenCopy) return

    if (selectedMigrationOptions.cutoverOption) {
      updateSelectedMigrationOptions('cutoverOption')(false)
    }

    if (selectedMigrationOptions.periodicSyncEnabled) {
      updateSelectedMigrationOptions('periodicSyncEnabled')(false)
    }

    onChange('periodicSyncInterval')('')
    onChange('cutoverStartTime')('')
    onChange('cutoverEndTime')('')
  }, [
    isPowerOffThenCopy,
    selectedMigrationOptions.cutoverOption,
    selectedMigrationOptions.periodicSyncEnabled,
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

  const isPCD = openstackCredentials?.metadata?.labels?.['vjailbreak.k8s.pf9.io/is-pcd'] === 'true'

  return (
    <LocalizationProvider dateAdapter={AdapterDayjs}>
      <Box sx={{ display: 'grid', gap: 2 }}>
        {showHeader ? (
          <Step stepNumber={stepNumber} label="Migration Options (Optional)" sx={{ mb: 0 }} />
        ) : null}

        {!isStorageAcceleratedCopy ? (
          <>
            <SectionBlock>
              <SectionHeaderRow>
                <Typography variant="subtitle2">Data copy</Typography>
                <Typography variant="caption" color="text.secondary">
                  How data is transferred before cutover
                </Typography>
              </SectionHeaderRow>
              <Divider />

              <OptionRow>
                <OptionLeft>
                  <FormControlLabel
                    id="data-copy-method"
                    label="Data copy method"
                    control={
                      <Checkbox
                        checked={selectedMigrationOptions.dataCopyMethod}
                        onChange={(e) => {
                          const isChecked = e.target.checked
                          updateSelectedMigrationOptions('dataCopyMethod')(isChecked)
                          if (!isChecked) {
                            onChange('dataCopyMethod')('cold')
                            onChange('acknowledgeNetworkConflictRisk')(false)
                          }
                        }}
                      />
                    }
                  />
                  <OptionHelp variant="caption">Choose cold or warm migration behavior.</OptionHelp>
                </OptionLeft>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, width: '100%' }}>
                  <Select
                    size="small"
                    disabled={!selectedMigrationOptions.dataCopyMethod}
                    labelId="source-item-label"
                    value={params?.dataCopyMethod || 'cold'}
                    onChange={(e) => {
                      onChange('dataCopyMethod')(e.target.value)
                      if (e.target.value !== 'mock') {
                        onChange('acknowledgeNetworkConflictRisk')(false)
                      }
                    }}
                    fullWidth
                  >
                    {DATA_COPY_OPTIONS.map((item) => (
                      <MenuItem key={item.value} value={item.value}>
                        {item.label}
                      </MenuItem>
                    ))}
                  </Select>
                </Box>
              </OptionRow>

              {params?.dataCopyMethod === 'mock' && (
                <Alert severity="warning" sx={{ width: '100%' }}>
                  <Typography variant="body2" sx={{ fontWeight: 600, mb: 0.5 }}>
                    Warning
                  </Typography>
                  <Typography variant="body2">
                    Migration without shutting down the source VM may cause network conflicts.
                    Please acknowledge the risks involved in migrating the VM to same subnet without
                    source poweroff.
                  </Typography>
                  <FormControlLabel
                    control={
                      <Checkbox
                        checked={Boolean(params?.acknowledgeNetworkConflictRisk)}
                        onChange={(e) => {
                          onChange('acknowledgeNetworkConflictRisk')(e.target.checked)
                        }}
                        color="warning"
                        size="small"
                      />
                    }
                    label="I understand the risks"
                    sx={{
                      mt: 1,
                      ml: 0,
                      alignItems: 'center',
                      '& .MuiFormControlLabel-label': { typography: 'body2' },
                      '& .MuiCheckbox-root': { padding: '4px' }
                    }}
                  />
                </Alert>
              )}

              <OptionRow>
                <OptionLeft>
                  <FormControlLabel
                    label="Schedule data copy"
                    control={
                      <Checkbox
                        checked={selectedMigrationOptions?.dataCopyStartTime}
                        onChange={(e) => {
                          const isChecked = e.target.checked
                          updateSelectedMigrationOptions('dataCopyStartTime')(isChecked)
                          if (!isChecked) {
                            onChange('dataCopyStartTime')('')
                            setValue('dataCopyStartTime', '')
                          }
                        }}
                      />
                    }
                  />
                  <OptionHelp variant="caption">
                    Optionally start data copy at a specific time.
                  </OptionHelp>
                </OptionLeft>
                <TimePicker
                  label="Data Copy Start Time"
                  identifier="dataCopyStartTime"
                  errors={errors}
                  getErrorsUpdater={getErrorsUpdater}
                  disabled={!selectedMigrationOptions.dataCopyStartTime}
                  required={!!selectedMigrationOptions.dataCopyStartTime}
                  disablePast
                />
              </OptionRow>
            </SectionBlock>

            <SectionBlock>
              <SectionHeaderRow>
                <Typography variant="subtitle2">Cutover</Typography>
                <Typography variant="caption" color="text.secondary">
                  When and how to switch traffic to the migrated VM
                </Typography>
              </SectionHeaderRow>
              <Divider />

              {isPowerOffThenCopy ? (
                <Alert severity="info" sx={{ mt: 1 }}>
                  Cutover options are disabled for cold migration (Power off then copy) because the
                  VM is powered off before copying. Cutover happens automatically after data copy
                  completes.
                </Alert>
              ) : null}

              <OptionRow>
                <OptionLeft>
                  <FormControlLabel
                    id="data-copy-method"
                    label="Cutover option"
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
                            setValue('cutoverStartTime', '')
                            setValue('cutoverEndTime', '')
                            updateSelectedMigrationOptions('periodicSyncEnabled')(false)
                            onChange('periodicSyncInterval')('')
                            setValue('periodicSyncInterval', '')
                          }
                        }}
                      />
                    }
                  />
                  <OptionHelp variant="caption">
                    Choose immediate, windowed, or admin-initiated cutover.
                  </OptionHelp>
                </OptionLeft>
                <Box sx={{ display: 'grid', gap: 1 }}>
                  <Select
                    size="small"
                    disabled={isPowerOffThenCopy || !selectedMigrationOptions?.cutoverOption}
                    value={params?.cutoverOption || '0'}
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

                  {params.cutoverOption === CUTOVER_TYPES.TIME_WINDOW &&
                    selectedMigrationOptions.cutoverOption &&
                    !isPowerOffThenCopy && (
                      <Box
                        sx={{
                          display: 'grid',
                          gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' },
                          gap: 1
                        }}
                      >
                        <TimePicker
                          label="Cutover Start Time"
                          identifier="cutoverStartTime"
                          errors={errors}
                          getErrorsUpdater={getErrorsUpdater}
                          required={params.cutoverOption === CUTOVER_TYPES.TIME_WINDOW}
                          disablePast
                        />
                        <TimePicker
                          label="Cutover End Time"
                          identifier="cutoverEndTime"
                          errors={errors}
                          getErrorsUpdater={getErrorsUpdater}
                          required={params.cutoverOption === CUTOVER_TYPES.TIME_WINDOW}
                          minDateTime={getMinEndTime()}
                          disablePast
                          helperText="Should be greater than data copy/cutover start time"
                        />
                      </Box>
                    )}

                  {params.cutoverOption === CUTOVER_TYPES.ADMIN_INITIATED &&
                    selectedMigrationOptions.cutoverOption &&
                    !isPowerOffThenCopy && (
                      <Box sx={{ display: 'grid', gap: 1 }}>
                        <FormControlLabel
                          control={
                            <Checkbox
                              checked={selectedMigrationOptions.periodicSyncEnabled}
                              onChange={(e) => {
                                const isChecked = e.target.checked
                                updateSelectedMigrationOptions('periodicSyncEnabled')(isChecked)
                                if (isChecked) {
                                  onChange('periodicSyncInterval')(
                                    globalConfigMap?.data.PERIODIC_SYNC_INTERVAL
                                  )
                                } else {
                                  onChange('periodicSyncInterval')('')
                                  setValue('periodicSyncInterval', '')
                                }
                              }}
                            />
                          }
                          label="Periodic sync"
                        />
                        <IntervalField
                          label="Periodic Sync"
                          name="periodicSyncInterval"
                          value={String(
                            params.periodicSyncInterval &&
                              selectedMigrationOptions.periodicSyncEnabled
                              ? params.periodicSyncInterval
                              : ''
                          )}
                          onChange={(e) => {
                            onChange('periodicSyncInterval')(e.target.value?.trim() || '')
                          }}
                          error={errors.periodicSyncInterval}
                          getErrorsUpdater={getErrorsUpdater}
                          disabled={!selectedMigrationOptions.periodicSyncEnabled}
                        />
                      </Box>
                    )}
                </Box>
              </OptionRow>
            </SectionBlock>
          </>
        ) : null}

        <SectionBlock>
          <SectionHeaderRow>
            <Typography variant="subtitle2">Post-migration actions</Typography>
            <Typography variant="caption" color="text.secondary">
              Optional cleanup and organization in vCenter
            </Typography>
          </SectionHeaderRow>
          <Divider />

          <OptionRow>
            <OptionLeft>
              <FormControlLabel
                label="Rename VMware VM"
                control={
                  <Checkbox
                    checked={!!selectedMigrationOptions.postMigrationAction?.renameVm}
                    onChange={(e) => {
                      const isChecked = e.target.checked

                      setValue(
                        'postMigrationActionSuffix',
                        isChecked ? params.postMigrationAction?.suffix || '_migrated_to_pcd' : ''
                      )

                      updateSelectedMigrationOptions('postMigrationAction')({
                        ...selectedMigrationOptions.postMigrationAction,
                        renameVm: isChecked,
                        suffix: isChecked
                          ? true
                          : selectedMigrationOptions.postMigrationAction?.suffix
                      })
                      onChange('postMigrationAction')({
                        ...params.postMigrationAction,
                        renameVm: isChecked,
                        suffix: isChecked
                          ? params.postMigrationAction?.suffix || '_migrated_to_pcd'
                          : undefined
                      })
                    }}
                  />
                }
              />
              <OptionHelp variant="caption">
                Append a suffix to the source VM name after migration.
              </OptionHelp>
            </OptionLeft>
            <RHFTextField
              name="postMigrationActionSuffix"
              label="VM Rename Suffix"
              disabled={!selectedMigrationOptions.postMigrationAction?.renameVm}
              placeholder="_migrated_to_pcd"
            />
          </OptionRow>

          <OptionRow>
            <OptionLeft>
              <FormControlLabel
                label="Move to folder in VMware"
                control={
                  <Checkbox
                    checked={!!selectedMigrationOptions.postMigrationAction?.moveToFolder}
                    onChange={(e) => {
                      const isChecked = e.target.checked

                      setValue(
                        'postMigrationActionFolderName',
                        isChecked ? params.postMigrationAction?.folderName || 'vjailbreakedVMs' : ''
                      )

                      updateSelectedMigrationOptions('postMigrationAction')({
                        ...selectedMigrationOptions.postMigrationAction,
                        moveToFolder: isChecked,
                        folderName: isChecked
                          ? true
                          : selectedMigrationOptions.postMigrationAction?.folderName
                      })
                      onChange('postMigrationAction')({
                        ...params.postMigrationAction,
                        moveToFolder: isChecked,
                        folderName: isChecked
                          ? params.postMigrationAction?.folderName || 'vjailbreakedVMs'
                          : undefined
                      })
                    }}
                  />
                }
              />
              <OptionHelp variant="caption">
                Organize migrated VMs into a vCenter folder.
              </OptionHelp>
            </OptionLeft>
            <RHFTextField
              name="postMigrationActionFolderName"
              label="Folder Name"
              disabled={!selectedMigrationOptions.postMigrationAction?.moveToFolder}
              placeholder="vjailbreakedVMs"
            />
          </OptionRow>
        </SectionBlock>

        <SectionBlock>
          <SectionHeaderRow>
            <Typography variant="subtitle2">Network and IP behavior</Typography>
            <Typography variant="caption" color="text.secondary">
              Reduce IP conflicts and handle edge cases
            </Typography>
          </SectionHeaderRow>
          <Divider />

          <OptionRow>
            <OptionLeft>
              <FormControlLabel
                label="Disconnect source VM network"
                control={
                  <Checkbox
                    checked={params?.disconnectSourceNetwork || false}
                    onChange={(e) => {
                      onChange('disconnectSourceNetwork')(e.target.checked)
                    }}
                  />
                }
              />
              <OptionHelp variant="caption">
                Disconnect NICs on the source VM to prevent IP conflicts.
              </OptionHelp>
            </OptionLeft>
            <Box />
          </OptionRow>

          <OptionRow>
            <OptionLeft>
              <FormControlLabel
                label="Fallback to DHCP"
                control={
                  <Checkbox
                    checked={params?.fallbackToDHCP || false}
                    onChange={(e) => {
                      onChange('fallbackToDHCP')(e.target.checked)
                    }}
                  />
                }
              />
              <OptionHelp variant="caption">Use DHCP if static IP cannot be preserved.</OptionHelp>
            </OptionLeft>
            <Box />
          </OptionRow>

          <OptionRow>
            <OptionLeft>
              <FormControlLabel
                label="Persist source network interfaces"
                control={
                  <Checkbox
                    checked={params?.networkPersistence || false}
                    onChange={(e) => {
                      onChange('networkPersistence')(e.target.checked)
                    }}
                  />
                }
              />
              <OptionHelp variant="caption">
                Retain the source VM's network interface names
              </OptionHelp>
            </OptionLeft>
            <Box />
          </OptionRow>
        </SectionBlock>

        {isPCD ? (
          <SectionBlock>
            <SectionHeaderRow>
              <Typography variant="subtitle2">PCD options</Typography>
              <Typography variant="caption" color="text.secondary">
                Flavour selection helpers
              </Typography>
            </SectionHeaderRow>
            <Divider />

            <OptionRow>
              <OptionLeft>
                <FormControlLabel
                  label="Use GPU-enabled flavours"
                  control={
                    <Checkbox
                      checked={params?.useGPU || false}
                      onChange={(e) => {
                        const isChecked = e.target.checked
                        updateSelectedMigrationOptions('useGPU')(isChecked)
                        onChange('useGPU')(isChecked)
                      }}
                    />
                  }
                />
                <OptionHelp variant="caption">
                  Migration may fail if a suitable GPU flavour is not found. Ignored if a flavour is
                  explicitly assigned per-VM.
                </OptionHelp>
              </OptionLeft>
              <Box />
            </OptionRow>

            <OptionRow>
              <OptionLeft>
                <FormControlLabel
                  label="Use dynamic hotplug-enabled flavors"
                  control={
                    <Checkbox
                      checked={params?.useFlavorless || false}
                      onChange={(e) => {
                        const isChecked = e.target.checked
                        updateSelectedMigrationOptions('useFlavorless')(isChecked)
                        onChange('useFlavorless')(isChecked)
                      }}
                    />
                  }
                />
                <OptionHelp variant="caption">
                  Uses the base flavor ID configured in PCD.
                </OptionHelp>
              </OptionLeft>
              <Box />
            </OptionRow>
          </SectionBlock>
        ) : null}

        <SectionBlock>
          <SectionHeaderRow>
            <Typography variant="subtitle2">Post-migration script</Typography>
            <Typography variant="caption" color="text.secondary">
              Run a script after migration completes
            </Typography>
          </SectionHeaderRow>
          <Divider />

          <OptionRow>
            <OptionLeft>
              <FormControlLabel
                label="Enable script"
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
              <OptionHelp variant="caption">Provide a script to run after migration.</OptionHelp>
            </OptionLeft>
            <CustomTextField
              // label="Script"
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
          </OptionRow>
        </SectionBlock>

        {/*
            Pre and Post Web Hooks
// ...
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
      </Box>
    </LocalizationProvider>
  )
}

const TimePicker = ({ identifier, errors, getErrorsUpdater, helperText = '', ...restProps }) => {
  return (
    <RHFDateTimeField
      name={identifier}
      label={restProps.label}
      disabled={restProps.disabled}
      required={restProps.required}
      disablePast={restProps.disablePast}
      minDateTime={restProps.minDateTime}
      helperText={!!errors[identifier] && !restProps?.disabled ? helperText : ''}
      onPickerError={(error) => {
        getErrorsUpdater(identifier)(error)
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
