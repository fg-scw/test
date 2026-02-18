import { createContext, useContext } from 'react'

type MigrationFormType = 'standard' | 'rolling'

export interface MigrationFormContextValue {
  openMigrationForm: (type: MigrationFormType) => void
}

export const MigrationFormContext = createContext<MigrationFormContextValue | undefined>(undefined)

export const useMigrationFormActions = () => {
  const context = useContext(MigrationFormContext)

  if (!context) {
    throw new Error('useMigrationFormActions must be used within a MigrationFormContext provider')
  }

  return context
}
