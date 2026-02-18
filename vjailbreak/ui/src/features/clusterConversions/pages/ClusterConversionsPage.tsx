import RollingMigrationsTable from '../components/RollingMigrationsTable'
import { useMigrationsQuery } from 'src/hooks/api/useMigrationsQuery'
import {
  useESXIMigrationsQuery,
  ESXI_MIGRATIONS_QUERY_KEY
} from 'src/hooks/api/useESXIMigrationsQuery'
import { useRollingMigrationPlansQuery } from 'src/hooks/api/useRollingMigrationPlansQuery'
import { THIRTY_SECONDS } from 'src/constants'
import { useRollingMigrationsStatusMonitor } from 'src/hooks/useRollingMigrationsStatusMonitor'

export default function ClusterConversionsPage() {
  const { data: migrations, refetch: refetchMigrations } = useMigrationsQuery()

  const { data: esxiMigrations, refetch: refetchESXIMigrations } = useESXIMigrationsQuery({
    queryKey: ESXI_MIGRATIONS_QUERY_KEY,
    refetchInterval: THIRTY_SECONDS,
    staleTime: 0,
    refetchOnMount: true
  })

  const { data: rollingMigrationPlans, refetch: refetchRollingMigrationPlans } =
    useRollingMigrationPlansQuery({
      refetchInterval: THIRTY_SECONDS,
      staleTime: 0,
      refetchOnMount: true
    })

  useRollingMigrationsStatusMonitor(rollingMigrationPlans)

  return (
    <RollingMigrationsTable
      rollingMigrationPlans={rollingMigrationPlans || []}
      esxiMigrations={esxiMigrations || []}
      migrations={migrations || []}
      refetchRollingMigrationPlans={refetchRollingMigrationPlans}
      refetchESXIMigrations={refetchESXIMigrations}
      refetchMigrations={refetchMigrations}
    />
  )
}
