import { NavigationItem } from '../types/navigation'
import MigrationIcon from '@mui/icons-material/SwapHoriz'
import AgentsIcon from '@mui/icons-material/Computer'
import CredentialsIcon from '@mui/icons-material/VpnKey'
import ClusterIcon from '@mui/icons-material/Hub'
import ConfigIcon from '@mui/icons-material/Settings'
import MonitoringIcon from '@mui/icons-material/Insights'
import DescriptionIcon from '@mui/icons-material/Description'
import { Storage } from '@mui/icons-material'
import SdStorageIcon from '@mui/icons-material/SdStorage'

export const navigationItems: NavigationItem[] = [
  {
    id: 'migrations',
    label: 'Migrations',
    path: '/dashboard/migrations',
    icon: <MigrationIcon />
  },
  {
    id: 'cluster-conversions',
    label: 'Cluster Conversions',
    path: '/dashboard/cluster-conversions',
    icon: <ClusterIcon />
  },
  {
    id: 'credentials',
    label: 'Credentials',
    path: '/dashboard/credentials',
    icon: <CredentialsIcon />
  },
  {
    id: 'agents',
    label: 'Agents',
    path: '/dashboard/agents',
    icon: <AgentsIcon />
  },
  {
    id: 'storage-management',
    label: 'Storage Management',
    path: '/dashboard/storage-management',
    icon: <SdStorageIcon />,
    badge: {
      label: 'Beta',
      color: 'warning',
      variant: 'outlined'
    }
  },
  {
    id: 'baremetal-config',
    label: 'Bare Metal Config',
    path: '/dashboard/baremetal-config',
    icon: <Storage />
  },
  {
    id: 'global-settings',
    label: 'Global Settings',
    path: '/dashboard/global-settings',
    icon: <ConfigIcon />
  },
  {
    id: 'monitoring',
    label: 'Monitoring',
    path: '/grafana',
    icon: <MonitoringIcon />,
    external: true
  },
  {
    id: 'docs',
    label: 'Documentation',
    path: '/docs',
    icon: <DescriptionIcon />,
    external: true,
    externalUrl: 'https://platform9.github.io/vjailbreak/introduction/getting_started/'
  }
]

export const getNavigationItemById = (id: string): NavigationItem | undefined => {
  return navigationItems.find((item) => item.id === id)
}

export const getNavigationItemByPath = (path: string): NavigationItem | undefined => {
  return navigationItems.find((item) => item.path === path)
}
