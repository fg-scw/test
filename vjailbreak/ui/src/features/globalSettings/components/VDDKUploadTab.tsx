import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import { Alert, Box, Button, Chip, IconButton, LinearProgress, Typography } from '@mui/material'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import ErrorIcon from '@mui/icons-material/Error'
import { FieldLabel, FileDropzone, SurfaceCard } from 'src/components'

export type VddkUploadStatus = 'idle' | 'uploading' | 'success' | 'error'

export type VDDKUploadTabProps = {
  selectedFile: File | null
  status: VddkUploadStatus
  progress: number
  message: string
  extractedPath?: string
  existingVddkPath?: string
  existingVddkVersion?: string
  existingVddkMessage?: string
  onFileSelected: (file: File | null) => void
  onClear: () => void
}

export default function VDDKUploadTab({
  selectedFile,
  status,
  progress,
  message,
  extractedPath,
  existingVddkPath,
  existingVddkVersion,
  onFileSelected,
  onClear
}: VDDKUploadTabProps) {
  const existingFileName = existingVddkPath ? existingVddkPath.split('/').filter(Boolean).pop() : ''

  const fileSizeLabel = selectedFile
    ? selectedFile.size >= 1024 * 1024
      ? `${(selectedFile.size / (1024 * 1024)).toFixed(1)} MB`
      : `${Math.round(selectedFile.size / 1024)} KB`
    : ''

  const statusChip =
    status === 'uploading' ? (
      <Chip label="Uploading" size="small" color="info" />
    ) : status === 'success' ? (
      <Chip label="Uploaded" size="small" color="success" />
    ) : status === 'error' ? (
      <Chip label="Needs attention" size="small" color="error" />
    ) : selectedFile ? (
      <Chip label="Ready to upload" size="small" color="default" />
    ) : (
      <Chip label="No file" size="small" color="default" />
    )

  return (
    <>
      {existingVddkPath ? (
        <Alert severity="info" sx={{ mb: 1 }}>
          <Typography variant="body2" fontWeight={600} sx={{ wordBreak: 'break-word' }}>
            Existing VDDK: {existingFileName || existingVddkPath}
          </Typography>
          {existingVddkVersion ? (
            <Typography variant="caption" sx={{ display: 'block' }}>
              Version: {existingVddkVersion}
            </Typography>
          ) : null}
          {/* {existingVddkMessage ? (
            <Typography variant="caption" sx={{ display: 'block' }}>
              {existingVddkMessage}
            </Typography>
          ) : null} */}
        </Alert>
      ) : null}
      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: { xs: '1fr', md: 'minmax(0, 1fr) 360px' },
          gap: 2,
          alignItems: 'start',
          width: '100%'
        }}
      >
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
          <FieldLabel label="VDDK File" tooltip="Upload a VDDK tar or tar.gz file (max 500MB)" />
          <Box data-tour="vddk-dropzone">
            <FileDropzone
              accept=".tar,.tgz,application/x-tar,application/gzip,application/x-gzip,application/x-compressed-tar"
              file={selectedFile}
              placeholder="Drag and drop VDDK file here"
              helperText="or click to browse"
              caption="Supported formats: .tar, .tar.gz (max 500MB)"
              onFileSelected={onFileSelected}
              disabled={status === 'uploading'}
              data-testid="vddk-file-dropzone"
            />
          </Box>
        </Box>

        <SurfaceCard
          variant="card"
          title={
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Typography variant="subtitle2" fontWeight={700}>
                Upload details
              </Typography>
              {statusChip}
            </Box>
          }
          actions={
            selectedFile && status !== 'success' ? (
              <IconButton
                size="small"
                onClick={onClear}
                disabled={status === 'uploading'}
                aria-label="Remove selected file"
              >
                <DeleteOutlineIcon fontSize="small" />
              </IconButton>
            ) : null
          }
          sx={{
            borderRadius: 2,
            border: (theme) => `1px solid ${theme.palette.divider}`,
            height: 'fit-content',
            mt: 3
          }}
          data-testid="vddk-upload-details"
        >
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <Typography variant="body2" fontWeight={600} sx={{ wordBreak: 'break-word' }}>
              {selectedFile ? selectedFile.name : 'No file selected'}
            </Typography>

            {selectedFile ? (
              <Typography variant="body2" color="text.secondary">
                Size: {fileSizeLabel}
              </Typography>
            ) : (
              <Typography variant="body2" color="text.secondary">
                Select a tar/tar.gz file. The upload runs when you click Save.
              </Typography>
            )}

            {status === 'uploading' ? (
              <Box sx={{ mt: 0.5 }}>
                <Typography variant="body2" color="text.secondary" gutterBottom>
                  {message}
                </Typography>
                <LinearProgress variant="determinate" value={progress} />
                <Typography
                  variant="caption"
                  color="text.secondary"
                  sx={{ mt: 0.5, display: 'block' }}
                >
                  {Math.round(progress)}% complete
                </Typography>
              </Box>
            ) : null}

            {selectedFile && status === 'idle' ? (
              <Typography variant="body2" color="text.secondary">
                Ready. Click Save to upload & extract.
              </Typography>
            ) : null}

            {/* {selectedFile ? (
            <Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
              <Button
                variant="text"
                color="inherit"
                size="small"
                onClick={onClear}
                disabled={status === 'uploading'}
              >
                Remove file
              </Button>
            </Box>
          ) : null} */}
          </Box>
        </SurfaceCard>

        {status === 'success' && (
          <Alert
            severity="success"
            icon={<CheckCircleIcon />}
            sx={{ mt: 2, gridColumn: { xs: '1', md: '1 / -1' } }}
            action={
              <Button color="inherit" size="small" onClick={onClear}>
                Replace
              </Button>
            }
          >
            <Typography variant="body2" fontWeight={600}>
              {message}
            </Typography>
            {extractedPath && (
              <Typography variant="caption" sx={{ mt: 1, display: 'block' }}>
                Extracted to: {extractedPath}
              </Typography>
            )}
          </Alert>
        )}

        {status === 'error' && (
          <Alert
            severity="error"
            icon={<ErrorIcon />}
            sx={{ mt: 2, gridColumn: { xs: '1', md: '1 / -1' } }}
            action={
              <Button color="inherit" size="small" onClick={onClear}>
                Try Again
              </Button>
            }
          >
            <Typography variant="body2" fontWeight={600}>
              {message}
            </Typography>
          </Alert>
        )}
      </Box>
    </>
  )
}
