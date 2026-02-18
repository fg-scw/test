import { Box } from '@mui/material'
import { Step, RHFAutocomplete } from 'src/shared/components/forms'
import { FormGrid } from 'src/components'
import {
  OpenstackCreds,
  SecurityGroupOption,
  ServerGroupOption
} from 'src/api/openstack-creds/model'

interface SecurityGroupAndServerGroupProps {
  params: {
    vms?: any[]
    securityGroups?: string[]
    serverGroup?: string
  }
  onChange: (key: string) => (value: any) => void
  openstackCredentials?: OpenstackCreds
  stepNumber?: string
  showHeader?: boolean
}

export default function SecurityGroupAndServerGroup({
  params,
  onChange,
  openstackCredentials,
  stepNumber = '4',
  showHeader = true
}: SecurityGroupAndServerGroupProps) {
  const securityGroupOptions: SecurityGroupOption[] =
    openstackCredentials?.status?.openstack?.securityGroups || []

  const serverGroupOptions: ServerGroupOption[] =
    openstackCredentials?.status?.openstack?.serverGroups || []

  return (
    <Box>
      {showHeader ? (
        <Step stepNumber={stepNumber} label="Security Groups & Server Group (Optional)" />
      ) : null}
      <Box>
        <FormGrid minWidth={320} gap={3}>
          <Box>
            <RHFAutocomplete<SecurityGroupOption>
              name="securityGroups"
              multiple
              options={securityGroupOptions}
              label="Security Groups"
              placeholder={
                params.securityGroups && params.securityGroups.length > 0
                  ? ''
                  : 'Select Security Groups'
              }
              getOptionLabel={(option) =>
                option.requiresIdDisplay
                  ? `${option.name} (${option.id.substring(0, 8)}...)`
                  : option.name
              }
              getOptionValue={(option) => option.id}
              renderOptionLabel={(option) =>
                option.requiresIdDisplay
                  ? `${option.name} (${option.id.substring(0, 8)}...)`
                  : option.name
              }
              showCheckboxes
              onValueChange={(value) => onChange('securityGroups')(value)}
              data-testid="security-groups-autocomplete"
              labelProps={{ tooltip: 'Assign security groups to the selected VMs.' }}
            />
          </Box>

          {/* Right side: Server Group */}
          <Box>
            <RHFAutocomplete<ServerGroupOption>
              name="serverGroup"
              options={serverGroupOptions}
              label="Server Group"
              placeholder="Select Server Group"
              getOptionLabel={(option) => `${option.name} (${option.policy})`}
              getOptionValue={(option) => option.id}
              renderOptionLabel={(option) => `${option.name} (${option.policy})`}
              onValueChange={(value) => onChange('serverGroup')(value)}
              data-testid="server-group-autocomplete"
              labelProps={{ tooltip: 'Control VM affinity/anti-affinity placement.' }}
            />
          </Box>
        </FormGrid>
      </Box>
    </Box>
  )
}
