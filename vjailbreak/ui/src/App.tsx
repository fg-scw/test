import { styled, Snackbar, Alert } from '@mui/material'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Route, Routes, useLocation, Navigate, useNavigate } from 'react-router-dom'
import Joyride, { CallBackProps, Step as JoyrideStep } from 'react-joyride'
import './assets/reset.css'
import { AppBar, DashboardLayout } from './components/layout'
import { RouteCompatibility } from './components/providers'
import MigrationFormDrawer from './features/migration/MigrationForm'
import RollingMigrationFormDrawer from './features/migration/RollingMigrationForm'
import MigrationsPage from './features/migration/pages/MigrationsPage'
import AgentsPage from './features/agents/pages/AgentsPage'
import CredentialsPage from './features/credentials/pages/CredentialsPage'
import ClusterConversionsPage from './features/clusterConversions/pages/ClusterConversionsPage'
import MaasConfigPage from './features/baremetalConfig/pages/MaasConfigPage'
import Onboarding from './features/onboarding/pages/Onboarding'
import GlobalSettingsPage from './features/globalSettings/pages/GlobalSettingsPage'
import StorageManagementPage from './features/storageManagement/pages/StorageManagementPage'
import { useVddkStatusQuery } from './hooks/api/useVddkStatusQuery'
import { useOpenstackCredentialsQuery } from './hooks/api/useOpenstackCredentialsQuery'
import { useVmwareCredentialsQuery } from './hooks/api/useVmwareCredentialsQuery'

const AppFrame = styled('div')(() => ({
  position: 'relative',
  display: 'grid',
  gridTemplateRows: 'auto 1fr',
  height: '100vh',
  overflow: 'hidden'
}))

const AppContent = styled('div')(({ theme }) => ({
  overflow: 'hidden',
  display: 'flex',
  flexDirection: 'column',
  flex: 1,
  [theme.breakpoints.up('lg')]: {
    maxWidth: '1600px',
    margin: '0 auto',
    width: '100%'
  }
}))

const GETTING_STARTED_DISMISSED_KEY = 'getting-started-dismissed'

type GuideMode = 'vddk' | 'credentials' | null

function useHasAnyCredentials() {
  const {
    data: vmwareCredentials,
    isLoading: vmwareLoading,
    isError: vmwareError
  } = useVmwareCredentialsQuery(undefined, { staleTime: 0, refetchOnMount: true })

  const {
    data: openstackCredentials,
    isLoading: openstackLoading,
    isError: openstackError
  } = useOpenstackCredentialsQuery(undefined, { staleTime: 0, refetchOnMount: true })

  const isLoading = vmwareLoading || openstackLoading
  const isError = vmwareError || openstackError

  const hasAnyCredentials = useMemo(() => {
    const vmwareCount = Array.isArray(vmwareCredentials) ? vmwareCredentials.length : 0
    const openstackCount = Array.isArray(openstackCredentials) ? openstackCredentials.length : 0
    return vmwareCount + openstackCount > 0
  }, [openstackCredentials, vmwareCredentials])

  return { hasAnyCredentials, isLoading, isError }
}

function DashboardIndexRedirect() {
  const { hasAnyCredentials, isLoading, isError } = useHasAnyCredentials()
  const vddkStatusQuery = useVddkStatusQuery({ refetchOnWindowFocus: false })
  const vddkLoading = vddkStatusQuery.isLoading
  const vddkError = vddkStatusQuery.isError
  const vddkUploaded = vddkStatusQuery.data?.uploaded === true

  if (isLoading || vddkLoading) return null
  if (isError) return <Navigate to="/dashboard/migrations" replace />

  // If the status endpoint fails, don't hard-block navigation.
  // Fall back to migrations/credentials behavior based on credentials only.
  if (vddkError) {
    return hasAnyCredentials ? (
      <Navigate to="/dashboard/migrations" replace />
    ) : (
      <Navigate to="/dashboard/credentials" replace />
    )
  }

  if (!vddkUploaded) {
    return <Navigate to="/dashboard/global-settings" state={{ tab: 'vddk' }} replace />
  }

  return hasAnyCredentials ? (
    <Navigate to="/dashboard/migrations" replace />
  ) : (
    <Navigate to="/dashboard/credentials" replace />
  )
}

function App() {
  const location = useLocation()
  const navigate = useNavigate()
  const appContentRef = useRef<HTMLDivElement | null>(null)
  const [openMigrationForm, setOpenMigrationForm] = useState(false)
  const [migrationType, setMigrationType] = useState('standard')
  const [joyrideRun, setJoyrideRun] = useState(false)
  const [joyrideSnoozed, setJoyrideSnoozed] = useState(false)
  const [joyrideReady, setJoyrideReady] = useState(false)
  const [notification, setNotification] = useState({
    open: false,
    message: '',
    severity: 'success' as 'error' | 'info' | 'success' | 'warning'
  })
  const hideAppbar = location.pathname === '/onboarding' || location.pathname === '/'

  const { hasAnyCredentials, isLoading: credsLoading } = useHasAnyCredentials()
  const vddkStatusQuery = useVddkStatusQuery({ refetchOnWindowFocus: false })
  const vddkLoading = vddkStatusQuery.isLoading
  const vddkError = vddkStatusQuery.isError
  const vddkUploaded = vddkStatusQuery.data?.uploaded === true

  const missingCredentials = !hasAnyCredentials
  const missingVddk = !vddkUploaded
  const shouldShowGuide = missingCredentials || missingVddk

  const guideMode: GuideMode = useMemo(() => {
    if (missingVddk) return 'vddk'
    if (missingCredentials) return 'credentials'
    return null
  }, [missingCredentials, missingVddk])

  const guideConfig = useMemo(() => {
    if (guideMode === 'vddk') {
      return {
        path: '/dashboard/global-settings',
        target: '[data-tour="vddk-dropzone"]',
        placement: 'right' as const,
        spotlightPadding: 10,
        content:
          'Upload the VMware VDDK library from Global Settings (VDDK Upload tab). This is mandatory before adding credentials.',
        navState: { tab: 'vddk' as const }
      }
    }

    if (guideMode === 'credentials') {
      return {
        path: '/dashboard/credentials',
        target: '[data-tour="add-vmware-creds"]',
        placement: 'bottom' as const,
        spotlightPadding: 8,
        content:
          'Add your PCD and VMware credentials from the Credentials page. Then you can start migrations.',
        navState: undefined
      }
    }

    return {
      path: null as string | null,
      target: null as string | null,
      placement: 'center' as const,
      spotlightPadding: 0,
      content: '',
      navState: undefined as undefined
    }
  }, [guideMode])

  const isOnExpectedPage = guideConfig.path ? location.pathname === guideConfig.path : false

  const joyrideSteps: JoyrideStep[] = useMemo(() => {
    if (!guideConfig.target) return []
    return [
      {
        target: guideConfig.target,
        placement: guideConfig.placement,
        spotlightPadding: guideConfig.spotlightPadding,
        disableBeacon: true,
        content: guideConfig.content
      }
    ]
  }, [guideConfig.content, guideConfig.placement, guideConfig.spotlightPadding, guideConfig.target])

  useEffect(() => {
    // If the user navigates away while a step is active, stop Joyride immediately
    // to avoid react-floater trying to attach to an unmounted element.
    if (!isOnExpectedPage) {
      setJoyrideRun(false)
    }
  }, [isOnExpectedPage])

  useEffect(() => {
    // Mark Joyride as "ready" only when we're on the right page and the target exists.
    // The target can mount after the route change (tabs/content), so we observe DOM changes.
    if (!guideConfig.target || !isOnExpectedPage) {
      setJoyrideReady(false)
      return
    }

    const check = () => Boolean(document.querySelector(guideConfig.target!))
    if (check()) {
      setJoyrideReady(true)
      return
    }

    setJoyrideReady(false)

    const observeRoot =
      appContentRef.current ?? (document.getElementById('root') as HTMLElement | null)
    if (!observeRoot) return

    const observer = new MutationObserver(() => {
      if (!isOnExpectedPage) return
      if (check()) {
        setJoyrideReady(true)
        observer.disconnect()
      }
    })

    observer.observe(observeRoot, { childList: true, subtree: true })
    return () => observer.disconnect()
  }, [guideConfig.target, isOnExpectedPage])

  useEffect(() => {
    if (credsLoading || vddkLoading) return
    const dismissed = localStorage.getItem(GETTING_STARTED_DISMISSED_KEY) === 'true'

    if (!shouldShowGuide) {
      setJoyrideSnoozed(false)
      setJoyrideRun(false)
      return
    }

    // If we can't determine VDDK status, don't force redirects/popups.
    if (vddkError) {
      setJoyrideRun(false)
      return
    }

    if (dismissed || joyrideSnoozed) {
      setJoyrideRun(false)
      return
    }

    // Redirect + popup logic:
    // 1) VDDK missing: force user to Global Settings -> VDDK tab
    // 2) Else credentials missing: force user to Credentials page
    if (guideMode === 'vddk' && guideConfig.path === '/dashboard/global-settings') {
      if (location.pathname !== guideConfig.path) {
        navigate(guideConfig.path, { replace: true, state: guideConfig.navState })
      } else {
        const currentTab = (location.state as any)?.tab
        if (currentTab !== 'vddk') {
          navigate(location.pathname, { replace: true, state: guideConfig.navState })
        }
      }
      setJoyrideRun(true)
      return
    }

    if (guideMode === 'credentials' && guideConfig.path === '/dashboard/credentials') {
      if (location.pathname !== guideConfig.path) {
        navigate(guideConfig.path, { replace: true })
      }
      setJoyrideRun(true)
      return
    }

    setJoyrideRun(false)
  }, [
    credsLoading,
    vddkLoading,
    vddkError,
    location.pathname,
    joyrideSnoozed,
    guideMode,
    guideConfig.navState,
    guideConfig.path,
    navigate,
    shouldShowGuide
  ])

  const handleJoyrideCallback = (data: CallBackProps) => {
    const { action, status, type } = data

    // Close/X should hide temporarily (do not persist dismissal)
    // Joyride may emit close as action="close" or as a skipped status.
    if (action === 'close') {
      setJoyrideSnoozed(true)
      setJoyrideRun(false)
      return
    }

    if (action === 'skip') {
      localStorage.setItem(GETTING_STARTED_DISMISSED_KEY, 'true')
      setJoyrideRun(false)
      return
    }

    // If target isn't found, don't spam the user; hide temporarily.
    if (type === 'error:target_not_found') {
      setJoyrideSnoozed(true)
      setJoyrideRun(false)
      return
    }

    if (status === 'skipped') {
      // Joyride can report status=skipped for both Skip and Close.
      // If it wasn't an explicit skip action, treat it as a temporary snooze.
      setJoyrideSnoozed(true)
      setJoyrideRun(false)
      return
    }

    if (status === 'finished') {
      localStorage.setItem(GETTING_STARTED_DISMISSED_KEY, 'true')
      setJoyrideRun(false)
    }
  }

  const handleOpenMigrationForm = (open, type = 'standard') => {
    setOpenMigrationForm(open)
    setMigrationType(type)
  }

  const handleSuccess = (message: string) => {
    setNotification({
      open: true,
      message,
      severity: 'success'
    })
  }

  return (
    <AppFrame>
      <RouteCompatibility />
      <Joyride
        steps={joyrideReady && isOnExpectedPage ? joyrideSteps : []}
        run={joyrideRun && joyrideSteps.length > 0 && joyrideReady && isOnExpectedPage}
        stepIndex={0}
        continuous={false}
        showSkipButton
        disableScrolling
        scrollToFirstStep={false}
        floaterProps={{
          styles: {
            floater: {
              position: 'fixed'
            }
          }
        }}
        disableOverlayClose={false}
        callback={handleJoyrideCallback}
        styles={{
          options: {
            zIndex: 20000
          },
          buttonNext: {
            backgroundColor: 'gray'
          },
          buttonBack: {
            color: 'gray'
          },
          buttonSkip: {
            color: 'gray'
          }
        }}
      />
      <AppBar setOpenMigrationForm={handleOpenMigrationForm} hide={hideAppbar} />
      <AppContent ref={appContentRef}>
        {openMigrationForm && migrationType === 'standard' && (
          <MigrationFormDrawer
            open
            onClose={() => setOpenMigrationForm(false)}
            onSuccess={handleSuccess}
          />
        )}
        {openMigrationForm && migrationType === 'rolling' && (
          <RollingMigrationFormDrawer open onClose={() => setOpenMigrationForm(false)} />
        )}
        <Routes>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<DashboardLayout />}>
            <Route index element={<DashboardIndexRedirect />} />
            <Route path="migrations" element={<MigrationsPage />} />
            <Route path="agents" element={<AgentsPage />} />
            <Route path="credentials" element={<CredentialsPage />} />
            <Route path="cluster-conversions" element={<ClusterConversionsPage />} />
            <Route path="baremetal-config" element={<MaasConfigPage />} />
            <Route path="global-settings" element={<GlobalSettingsPage />} />
            <Route path="storage-management" element={<StorageManagementPage />} />
          </Route>
          <Route path="/onboarding" element={<Onboarding />} />
        </Routes>
      </AppContent>
      <Snackbar
        open={notification.open}
        autoHideDuration={6000}
        onClose={() => setNotification((prev) => ({ ...prev, open: false }))}
        anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
        sx={{
          top: { xs: 80, sm: 90 },
          right: { xs: 16, sm: 24 }
        }}
      >
        <Alert
          onClose={() => setNotification((prev) => ({ ...prev, open: false }))}
          severity={notification.severity}
          variant="filled"
          sx={{
            minWidth: '350px',
            fontSize: '1rem',
            fontWeight: 600,
            boxShadow: '0 8px 24px rgba(0, 0, 0, 0.25)',
            '& .MuiAlert-icon': {
              fontSize: '28px'
            },
            '& .MuiAlert-message': {
              fontSize: '1rem',
              fontWeight: 600
            }
          }}
        >
          {notification.message}
        </Alert>
      </Snackbar>
    </AppFrame>
  )
}

export default App
