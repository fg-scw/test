import AddIcon from '@mui/icons-material/Add'
import {
  Box,
  Button,
  FormControl,
  FormHelperText,
  InputLabel,
  MenuItem,
  Select,
  Typography
} from '@mui/material'
import { useState } from 'react'
import ResourceMappingTable from './ResourceMappingTable'

export interface ResourceMap {
  source: string
  target: string
}

interface ResourceMappingProps {
  label?: string
  sourceItems: string[]
  targetItems: string[]
  sourceLabel: string // Label for the source dropdown
  targetLabel: string // Label for the target dropdown
  values: ResourceMap[]
  onChange: (mappings: ResourceMap[]) => void
  error?: string
}

export default function ResourceMapping({
  label,
  sourceItems,
  targetItems,
  sourceLabel,
  targetLabel,
  values = [],
  onChange,
  error
}: ResourceMappingProps) {
  const [selectedSourceItem, setSelectedSourceItem] = useState('')
  const [selectedTargetItem, setSelectedTargetItem] = useState('')

  const handleAddMapping = () => {
    if (selectedSourceItem && selectedTargetItem) {
      const updatedMappings = [
        ...values,
        {
          source: selectedSourceItem,
          target: selectedTargetItem
        }
      ]

      onChange(updatedMappings)
      setSelectedSourceItem('')
      setSelectedTargetItem('')
    }
  }

  const handleDeleteMapping = (mapping: ResourceMap) => {
    const updatedMappings = values.filter(
      ({ source, target }) => mapping.source !== source || mapping.target !== target
    )

    onChange(updatedMappings)
  }

  // Filter out already mapped source and target items
  const availableSourceItems = sourceItems.filter(
    (item) => !values.some((mapping) => mapping.source === item)
  )
  const availableTargetItems = targetItems.filter(
    (item) => !values.some((mapping) => mapping.target === item)
  )

  return (
    <div>
      {label && <Typography variant="body1">{label}</Typography>}
      {values.length > 0 && (
        <ResourceMappingTable
          sourceLabel={sourceLabel}
          targetLabel={targetLabel}
          mappings={values}
          onDeleteRow={handleDeleteMapping}
        />
      )}
      <Box
        sx={{
          display: 'grid',
          mt: 2,
          mb: 2,
          gap: 2,
          gridTemplateColumns: '1fr 1fr max-content'
        }}
      >
        <FormControl fullWidth size="small" disabled={availableSourceItems.length === 0}>
          <InputLabel id="source-item-label">{sourceLabel}</InputLabel>
          <Select
            labelId="source-item-label"
            value={selectedSourceItem}
            onChange={(e) => setSelectedSourceItem(e.target.value)}
            label={sourceLabel}
          >
            {availableSourceItems.map((item) => (
              <MenuItem key={item} value={item}>
                {item}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <FormControl fullWidth size="small" disabled={availableTargetItems.length === 0}>
          <InputLabel id="target-item-label">{targetLabel}</InputLabel>
          <Select
            labelId="target-item-label"
            value={selectedTargetItem}
            onChange={(e) => setSelectedTargetItem(e.target.value)}
            label={targetLabel}
          >
            {availableTargetItems.map((item) => (
              <MenuItem key={item} value={item}>
                {item}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <Button
          variant="contained"
          color="primary"
          onClick={handleAddMapping}
          disabled={!selectedSourceItem || !selectedTargetItem}
          startIcon={<AddIcon />}
        >
          Add
        </Button>
      </Box>
      {!!error && <FormHelperText error>{error}</FormHelperText>}
    </div>
  )
}
