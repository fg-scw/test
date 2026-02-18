import { useMemo } from 'react'
import { VmData } from 'src/features/migration/api/migration-templates/model'
import { RdmDisk } from 'src/api/rdm-disks/model'

interface RdmConfigValidationProps {
  selectedVMs: VmData[]
  rdmDisks: RdmDisk[]
}

interface RdmConfigValidationResult {
  hasValidationError: boolean
  errorMessage: string
  hasRdmVMs: boolean
  hasPowerStateError: boolean
  powerStateErrorMessage: string
  hasConfigError: boolean
  configErrorMessage: string
  hasSelectionError: boolean
  selectionErrorMessage: string
  missingVMs: string[]
  rdmGroups: Record<string, unknown>
  requiredVMs: string[]
  invalidRdmDisks: Array<{
    diskName: string
    ownerVMs: string[]
    missingFields: string[]
    hasPoweredOnVMs: boolean
    poweredOnVMs: string[]
  }>
}

export const useRdmConfigValidation = ({
  selectedVMs,
  rdmDisks
}: RdmConfigValidationProps): RdmConfigValidationResult => {
  const validationResult = useMemo(() => {
    // If no VMs are selected or no RDM disks exist, no validation errors
    if (selectedVMs.length === 0 || rdmDisks.length === 0) {
      return {
        hasValidationError: false,
        errorMessage: '',
        hasRdmVMs: false,
        hasPowerStateError: false,
        powerStateErrorMessage: '',
        hasConfigError: false,
        configErrorMessage: '',
        hasSelectionError: false,
        selectionErrorMessage: '',
        missingVMs: [],
        rdmGroups: {},
        requiredVMs: [],
        invalidRdmDisks: []
      }
    }

    // Get selected VM names
    const selectedVmNames = new Set(selectedVMs.map((vm) => vm.name))

    // Check if any selected VM has RDM disks
    const vmsWithRdm = selectedVMs.filter((vm) => vm.hasSharedRdm)
    const hasRdmVMs = vmsWithRdm.length > 0

    if (!hasRdmVMs) {
      return {
        hasValidationError: false,
        errorMessage: '',
        hasRdmVMs: false,
        hasPowerStateError: false,
        powerStateErrorMessage: '',
        hasConfigError: false,
        configErrorMessage: '',
        hasSelectionError: false,
        selectionErrorMessage: '',
        missingVMs: [],
        rdmGroups: {},
        requiredVMs: [],
        invalidRdmDisks: []
      }
    }

    // Power state validation
    const poweredOnRdmVMs = vmsWithRdm.filter((vm) => {
      if (!vm.vmState) return false
      const powerState = vm.vmState.toLowerCase()
      return powerState === 'running' || powerState === 'poweredon' || powerState === 'on'
    })

    const hasPoweredOnVMs = poweredOnRdmVMs.length > 0
    let powerStateErrorMessage = ''

    if (hasPoweredOnVMs) {
      const poweredOnVmNames = poweredOnRdmVMs.map((vm) => vm.name).join(', ')
      powerStateErrorMessage = `All VMs with shared RDM disks must be powered off for migration. Currently powered on: ${poweredOnVmNames}`
    }

    // Check each RDM disk that has selected VMs as owners for configuration issues
    const invalidRdmDisks: Array<{
      diskName: string
      ownerVMs: string[]
      missingFields: string[]
      hasPoweredOnVMs: boolean
      poweredOnVMs: string[]
    }> = []

    rdmDisks.forEach((rdmDisk) => {
      // Check if this RDM disk has any selected VMs as owners
      const relevantOwnerVMs = rdmDisk.spec.ownerVMs.filter((ownerVM) =>
        selectedVmNames.has(ownerVM)
      )

      if (relevantOwnerVMs.length === 0) {
        // This RDM disk doesn't affect selected VMs
        return
      }

      // Check for missing required fields
      const missingFields: string[] = []

      if (!rdmDisk.spec.openstackVolumeRef?.cinderBackendPool) {
        missingFields.push('cinderBackendPool')
      }

      if (!rdmDisk.spec.openstackVolumeRef?.volumeType) {
        missingFields.push('volumeType')
      }

      // Check if volumeRef (source) is missing or empty
      if (
        !rdmDisk.spec.openstackVolumeRef?.source ||
        Object.keys(rdmDisk.spec.openstackVolumeRef.source).length === 0
      ) {
        missingFields.push('volumeRef')
      }

      // Check if any owner VMs are powered on
      const poweredOnVMs = relevantOwnerVMs.filter((vmName) => {
        const vm = selectedVMs.find((v) => v.name === vmName)
        if (!vm) return false

        const powerState = vm.vmState?.toLowerCase()
        return powerState === 'running' || powerState === 'poweredon' || powerState === 'on'
      })

      // If there are missing fields, add to validation results
      if (missingFields.length > 0) {
        invalidRdmDisks.push({
          diskName: rdmDisk.spec.diskName,
          ownerVMs: relevantOwnerVMs,
          missingFields,
          hasPoweredOnVMs: poweredOnVMs.length > 0,
          poweredOnVMs
        })
      }
    })

    // Generate configuration error message if there are validation errors
    let configErrorMessage = ''
    const hasConfigError = invalidRdmDisks.length > 0

    if (hasConfigError) {
      const allMissingFields = Array.from(
        new Set(invalidRdmDisks.flatMap((disk) => disk.missingFields))
      )

      const allDiskNames = invalidRdmDisks.map((disk) => disk.diskName).join(', ')
      configErrorMessage = `Cannot submit migration plan: RDM disk${invalidRdmDisks.length > 1 ? 's' : ''} (${allDiskNames}) ${invalidRdmDisks.length > 1 ? 'require' : 'requires'} configuration (${allMissingFields.join(', ')}). Please configure the RDM disk${invalidRdmDisks.length > 1 ? 's' : ''} before proceeding.`
    }

    // Combined validation - either power state error or config error prevents submission
    const hasValidationError = hasPoweredOnVMs || hasConfigError

    // Priority: power state error message first, then config error
    const errorMessage = hasPoweredOnVMs ? powerStateErrorMessage : configErrorMessage

    return {
      hasValidationError,
      errorMessage,
      hasRdmVMs,
      hasPowerStateError: hasPoweredOnVMs,
      powerStateErrorMessage,
      hasConfigError,
      configErrorMessage,
      hasSelectionError: false,
      selectionErrorMessage: '',
      missingVMs: [],
      rdmGroups: {},
      requiredVMs: [],
      invalidRdmDisks
    }
  }, [selectedVMs, rdmDisks])

  return validationResult
}
