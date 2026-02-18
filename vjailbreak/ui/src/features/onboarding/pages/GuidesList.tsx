import { Link, Typography } from '@mui/material'
import { styled } from '@mui/system'

const GuidesSection = styled('div')(({ theme }) => ({
  display: 'grid',
  gap: theme.spacing(1),
  gridAutoRows: 'min-content',
  padding: theme.spacing(2)
}))

const GuidesSectionHeader = styled('div')(({ theme }) => ({
  display: 'grid',
  gridTemplateColumns: 'repeat(2, max-content)',
  gap: theme.spacing(2)
}))

const UnorderedList = styled('ul')(({ theme }) => ({
  listStyle: 'none',
  padding: 0,
  marginTop: theme.spacing(2),
  textAlign: 'left',
  '& li': {
    marginBottom: theme.spacing(2)
  }
}))

interface GuidesListProps {
  listHeader: string
  listIcon: React.ReactNode
  links: Link[]
}

interface Link {
  text: string
  url: string
}

export default function GuidesList({ listHeader, listIcon, links }: GuidesListProps) {
  return (
    <GuidesSection>
      <GuidesSectionHeader>
        {listIcon}
        <Typography variant="h6">{listHeader}</Typography>
      </GuidesSectionHeader>
      <UnorderedList>
        {links.map(({ text, url }) => {
          return (
            <li key={text}>
              <Link href={url} variant="body1">
                {text}
              </Link>
            </li>
          )
        })}
      </UnorderedList>
    </GuidesSection>
  )
}
