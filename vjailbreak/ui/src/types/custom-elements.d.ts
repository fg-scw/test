import React from 'react'

declare global {
  namespace JSX {
    interface IntrinsicElements {
      'cds-icon': React.DetailedHTMLProps<React.HTMLAttributes<HTMLElement>, HTMLElement> & {
        shape?: string
        status?: string
        size?: string
        badge?: string
        solid?: boolean
        direction?: string
      }
    }
  }
}

export {}
