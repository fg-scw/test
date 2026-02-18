import axios from 'src/api/axios'

export interface VddkUploadResponse {
  success?: boolean
  message?: string
  file_path?: string
  extracted_path?: string
}

export interface VddkStatusResponse {
  uploaded: boolean
  path?: string
  message?: string
  version?: string
}

export type UploadProgressHandler = (progress: number) => void

export const uploadVddkFile = async (
  file: File,
  { onProgress }: { onProgress?: UploadProgressHandler } = {}
): Promise<VddkUploadResponse> => {
  const formData = new FormData()
  formData.append('vddk_file', file)

  return axios.post<VddkUploadResponse>({
    endpoint: '/dev-api/sdk/vpw/v1/vddk/upload',
    data: formData,
    config: {
      headers: {
        'Content-Type': 'multipart/form-data'
      },
      onUploadProgress: (event) => {
        if (!event.total) return
        const progress = (event.loaded / event.total) * 100
        onProgress?.(progress)
      }
    }
  })
}

export const getVddkStatus = async (): Promise<VddkStatusResponse> => {
  return axios.get<VddkStatusResponse>({
    endpoint: '/dev-api/sdk/vpw/v1/vddk/status'
  })
}
