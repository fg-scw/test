import { useState, useEffect } from 'react'
import { getVmwareCredentialsList } from 'src/api/vmware-creds/vmwareCreds'
import { getVMwareClusters } from 'src/api/vmware-clusters/vmwareClusters'
import { getPCDClusters } from 'src/api/pcd-clusters'
import { getOpenstackCredentials } from 'src/api/openstack-creds/openstackCreds'
import { VJAILBREAK_DEFAULT_NAMESPACE } from 'src/api/constants'
import { VMwareCreds } from 'src/api/vmware-creds/model'
import { VMwareCluster } from 'src/api/vmware-clusters/model'
import { PCDCluster } from 'src/api/pcd-clusters/model'

export interface SourceDataItem {
  credName: string
  datacenter: string
  vcenterName: string
  clusters: {
    id: string
    name: string
    displayName: string
  }[]
}

export interface PcdDataItem {
  id: string
  name: string
  openstackCredName: string
  tenantName: string
}

interface ClusterDataState {
  sourceData: SourceDataItem[]
  pcdData: PcdDataItem[]
  loadingVMware: boolean
  loadingPCD: boolean
  error: string | null
}

interface ClusterDataActions {
  refetchSourceData: () => Promise<void>
  refetchPcdData: () => Promise<void>
  refetchAll: () => Promise<void>
}

export type UseClusterDataReturn = ClusterDataState & ClusterDataActions

export const useClusterData = (autoFetch: boolean = true): UseClusterDataReturn => {
  const [sourceData, setSourceData] = useState<SourceDataItem[]>([])
  const [pcdData, setPcdData] = useState<PcdDataItem[]>([])
  const [loadingVMware, setLoadingVMware] = useState(false)
  const [loadingPCD, setLoadingPCD] = useState(false)
  const [error, setError] = useState<string | null>(null)
  // Removed fetchSecrets function - no longer needed
  const fetchSourceData = async () => {
    setLoadingVMware(true)
    setError(null)
    try {
      const vmwareCreds = await getVmwareCredentialsList(VJAILBREAK_DEFAULT_NAMESPACE)

      if (!vmwareCreds || vmwareCreds.length === 0) {
        setSourceData([])
        return
      }

      const sourceDataPromises = vmwareCreds.map(async (cred: VMwareCreds) => {
        const credName = cred.metadata.name
        const fixedDatacenter = cred.spec.datacenter

        // Use hostName directly from credential spec instead of fetching from secret
        const vcenterName = cred.spec.hostName || credName

        const clustersResponse = await getVMwareClusters(VJAILBREAK_DEFAULT_NAMESPACE, credName)
        
        const clustersByDC: Record<string, typeof clustersResponse.items> = {}

        clustersResponse.items.forEach((cluster: VMwareCluster) => {
          const annotations = (cluster.metadata as any).annotations
          const dcAnnotation = annotations?.['vjailbreak.k8s.pf9.io/datacenter'] || fixedDatacenter || 'Unknown'
          if (!clustersByDC[dcAnnotation]) {
            clustersByDC[dcAnnotation] = []
          }
          clustersByDC[dcAnnotation].push(cluster)
        })

        return Object.keys(clustersByDC).map(dcName => {
           const clusters = clustersByDC[dcName].map((cluster: VMwareCluster) => ({
            id: `${credName}:${dcName}:${cluster.metadata.name}`,
            name: cluster.metadata.name,
            displayName: cluster.spec.name,
            datacenter: dcName
          }))

          return {
            credName,
            datacenter: dcName,
            vcenterName,
            clusters
          }
        })
      })

      const nestedResults = await Promise.all(sourceDataPromises)
      const newSourceData = nestedResults.flat().filter((item) => item.clusters.length > 0)
      
      setSourceData(newSourceData)
    } catch (error) {
      console.error('Failed to fetch VMware cluster data:', error)
      setError('Failed to fetch VMware cluster data')
    } finally {
      setLoadingVMware(false)
    }
  }

  const fetchPcdData = async () => {
    setLoadingPCD(true)
    setError(null)
    try {
      const pcdClusters = await getPCDClusters(VJAILBREAK_DEFAULT_NAMESPACE)

      if (!pcdClusters || pcdClusters.items.length === 0) {
        setPcdData([])
        return
      }

      const clusterDataPromises = pcdClusters.items.map(async (cluster: PCDCluster) => {
        const clusterName = cluster.spec.clusterName
        const openstackCredName =
          cluster.metadata.labels?.['vjailbreak.k8s.pf9.io/openstackcreds'] || ''

        let tenantName = ''
        if (openstackCredName) {
          try {
            const openstackCreds = await getOpenstackCredentials(
              openstackCredName,
              VJAILBREAK_DEFAULT_NAMESPACE
            )
            tenantName = openstackCreds?.spec?.projectName || ''
          } catch (error) {
            console.error(`Failed to fetch OpenStack credentials for ${openstackCredName}:`, error)
          }
        }

        return {
          id: cluster.metadata.name,
          name: clusterName,
          openstackCredName: openstackCredName,
          tenantName: tenantName
        }
      })

      const clusterData = await Promise.all(clusterDataPromises)
      setPcdData(clusterData)
    } catch (error) {
      console.error('Failed to fetch PCD clusters:', error)
      setError('Failed to fetch PCD clusters')
    } finally {
      setLoadingPCD(false)
    }
  }

  const refetchAll = async () => {
    // Fetch source and PCD data in parallel
    await Promise.all([fetchSourceData(), fetchPcdData()])
  }

  // Auto-fetch on mount if enabled
  useEffect(() => {
    if (autoFetch) {
      refetchAll()
    }
  }, [autoFetch])

  return {
    sourceData,
    pcdData,
    loadingVMware,
    loadingPCD,
    error,
    refetchSourceData: fetchSourceData,
    refetchPcdData: fetchPcdData,
    refetchAll
  }
}
