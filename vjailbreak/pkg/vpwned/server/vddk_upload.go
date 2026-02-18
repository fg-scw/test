package server

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	maxUploadSize    = 500 * 1024 * 1024
	vddkUploadDir    = "/home/ubuntu"
	vddkInstallDir   = "/home/ubuntu/vmware-vix-disklib-distrib"
)

// VDDKStatusResponse represents the response for VDDK status check
type VDDKStatusResponse struct {
	Uploaded bool   `json:"uploaded"`
	Path     string `json:"path,omitempty"`
	Version  string `json:"version,omitempty"`
	Message  string `json:"message"`
}

type VDDKUploadResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	FilePath      string `json:"file_path,omitempty"`
	ExtractedPath string `json:"extracted_path,omitempty"`
}

// getVDDKVersion extracts the VDDK version from the libvixDiskLib shared library filename
func getVDDKVersion() string {
	const fn = "getVDDKVersion"

	// Find the libvixDiskLib.so* file in lib64 directory
	libDir := filepath.Join(vddkInstallDir, "lib64")
	pattern := filepath.Join(libDir, "libvixDiskLib.so.*")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		logrus.WithField("func", fn).WithError(err).Debug("Failed to glob for libvixDiskLib.so")
		return ""
	}
	if len(matches) == 0 {
		logrus.WithField("func", fn).Debug("Could not find libvixDiskLib.so")
		return ""
	}

	// Find the actual library file (not a symlink) with the full version like libvixDiskLib.so.8.0.0
	var libPath string
	for _, match := range matches {
		info, err := os.Lstat(match)
		if err != nil {
			logrus.WithField("func", fn).WithError(err).Debug("Failed to stat library file")
			continue
		}
		// Skip symlinks, we want the actual file
		if info.Mode()&os.ModeSymlink == 0 {
			// Prefer the one with the longest name (most specific version)
			if len(match) > len(libPath) {
				libPath = match
			}
		}
	}

	if libPath == "" {
		logrus.WithFields(logrus.Fields{
			"func":    fn,
			"matches": matches,
		}).Debug("No non-symlink libvixDiskLib.so file found")
		return ""
	}

	// Extract version from filename: libvixDiskLib.so.8.0.0 -> 8.0.0
	basename := filepath.Base(libPath)
	version := strings.TrimPrefix(basename, "libvixDiskLib.so.")

	logrus.WithFields(logrus.Fields{
		"func":    fn,
		"version": version,
		"libPath": libPath,
	}).Debug("Extracted VDDK version from filename")

	return version
}

// HandleVDDKStatus checks if VDDK has been uploaded and is available
func HandleVDDKStatus(w http.ResponseWriter, r *http.Request) {
	const fn = "HandleVDDKStatus"
	logrus.WithField("func", fn).Info("Checking VDDK status")

	if r.Method != http.MethodGet {
		logrus.WithField("func", fn).Errorf("Invalid method: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Check if VDDK directory has files
	// The daemonset ensures the directory exists, so we only need to check if it's empty
	files, err := os.ReadDir(vddkInstallDir)
	if err != nil || len(files) == 0 {
		if err != nil {
			logrus.WithField("func", fn).WithError(err).Info("VDDK directory not accessible")
		} else {
			logrus.WithField("func", fn).Info("VDDK directory is empty")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(VDDKStatusResponse{
			Uploaded: false,
			Message:  "VDDK has not been uploaded",
		})
		return
	}

	version := getVDDKVersion()

	logrus.WithFields(logrus.Fields{
		"func":       fn,
		"path":       vddkInstallDir,
		"file_count": len(files),
		"version":    version,
	}).Info("VDDK is available")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(VDDKStatusResponse{
		Uploaded: true,
		Path:     vddkInstallDir,
		Version:  version,
		Message:  "VDDK is available",
	})
}

func HandleVDDKUpload(w http.ResponseWriter, r *http.Request) {
	const fn = "HandleVDDKUpload"
	logrus.WithField("func", fn).Info("Starting VDDK tar file upload")

	if r.Method != http.MethodPost {
		logrus.WithField("func", fn).Errorf("Invalid method: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		logrus.WithField("func", fn).WithError(err).Error("File too large or failed to parse form")
		http.Error(w, "File too large. Maximum size is 500MB", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("vddk_file")
	if err != nil {
		logrus.WithField("func", fn).WithError(err).Error("Failed to get file from form")
		http.Error(w, "Failed to retrieve file from request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	logrus.WithFields(logrus.Fields{
		"func":     fn,
		"filename": handler.Filename,
		"size":     handler.Size,
	}).Info("Received VDDK file")

	// Validate file type - accept .tar, .tar.gz, and .tgz files
	filename := strings.ToLower(handler.Filename)
	validExtensions := []string{".tar", ".tar.gz", ".tgz"}
	isValid := false
	for _, ext := range validExtensions {
		if strings.HasSuffix(filename, ext) {
			isValid = true
			break
		}
	}
	if !isValid {
		logrus.WithField("func", fn).Errorf("Invalid file type: %s", handler.Filename)
		http.Error(w, "Invalid file type. Only .tar, .tar.gz, or .tgz files are allowed", http.StatusBadRequest)
		return
	}

	if err := os.MkdirAll(vddkUploadDir, 0755); err != nil {
		logrus.WithField("func", fn).WithError(err).Error("Failed to create upload directory")
		http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	destPath := filepath.Join(vddkUploadDir, handler.Filename)
	logrus.WithFields(logrus.Fields{
		"func":      fn,
		"dest_path": destPath,
	}).Info("Saving VDDK file")

	destFile, err := os.Create(destPath)
	if err != nil {
		logrus.WithField("func", fn).WithError(err).Error("Failed to create destination file")
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer destFile.Close()

	bytesWritten, err := io.Copy(destFile, file)
	if err != nil {
		logrus.WithField("func", fn).WithError(err).Error("Failed to write file")
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	logrus.WithFields(logrus.Fields{
		"func":          fn,
		"bytes_written": bytesWritten,
		"dest_path":     destPath,
	}).Info("Successfully saved VDDK file")

	// Extract the tar file directly to vddkUploadDir
	// The tar file contains vmware-vix-disklib-distrib/ as the top-level directory
	extractDir := vddkUploadDir
	logrus.WithFields(logrus.Fields{
		"func":        fn,
		"extract_dir": extractDir,
	}).Info("Extracting VDDK tar file")

	if err := extractTarFile(destPath, extractDir); err != nil {
		logrus.WithField("func", fn).WithError(err).Error("Failed to extract tar file")
		http.Error(w, fmt.Sprintf("File uploaded but extraction failed: %v", err), http.StatusInternalServerError)
		return
	}

	logrus.WithFields(logrus.Fields{
		"func":        fn,
		"extract_dir": extractDir,
	}).Info("Successfully extracted VDDK file")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := fmt.Sprintf(`{"success": true, "message": "VDDK file uploaded and extracted successfully", "file_path": "%s", "extracted_path": "%s"}`, destPath, extractDir)
	w.Write([]byte(response))
}

// isPathWithinRoot checks if a path (after resolving symlinks) is within the root directory
func isPathWithinRoot(root, path string) (bool, error) {
	// Clean and make paths absolute
	cleanRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return false, fmt.Errorf("failed to get absolute path for root: %w", err)
	}

	cleanPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return false, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Try to resolve symlinks if the path exists
	// If it doesn't exist yet, that's okay - we'll check the cleaned path
	resolvedPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		// Path doesn't exist yet or can't be resolved - use the cleaned path
		resolvedPath = cleanPath
	}

	// Get relative path from root to the resolved path
	relPath, err := filepath.Rel(cleanRoot, resolvedPath)
	if err != nil {
		return false, fmt.Errorf("failed to get relative path: %w", err)
	}

	// Check if the relative path tries to escape (starts with ..)
	if strings.HasPrefix(relPath, ".."+string(os.PathSeparator)) || relPath == ".." {
		return false, nil
	}

	return true, nil
}

// extractTarFile extracts a tar or tar.gz file to the specified destination
func extractTarFile(srcPath, destDir string) error {
	const fn = "extractTarFile"
	logrus.WithFields(logrus.Fields{
		"func":     fn,
		"src":      srcPath,
		"dest_dir": destDir,
	}).Info("Starting tar extraction")

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create extraction directory: %w", err)
	}

	// Open the tar file
	file, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open tar file: %w", err)
	}
	defer file.Close()

	var tarReader *tar.Reader

	// Check if it's a gzipped tar file
	if strings.HasSuffix(srcPath, ".gz") || strings.HasSuffix(srcPath, ".tgz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		tarReader = tar.NewReader(gzReader)
	} else {
		tarReader = tar.NewReader(file)
	}

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct the full path
		target := filepath.Join(destDir, header.Name)

		// Ensure the path is within destDir (security check with symlink resolution)
		withinRoot, err := isPathWithinRoot(destDir, target)
		if err != nil {
			return fmt.Errorf("failed to validate path %s: %w", header.Name, err)
		}
		if !withinRoot {
			return fmt.Errorf("illegal file path in tar (path traversal attempt): %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}
			logrus.WithField("func", fn).Debugf("Created directory: %s", target)

		case tar.TypeReg:
			// Create parent directory if it doesn't exist
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", target, err)
			}

			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file %s: %w", target, err)
			}
			outFile.Close()

			// Set file permissions
			if err := os.Chmod(target, os.FileMode(header.Mode)); err != nil {
				logrus.WithField("func", fn).Warnf("Failed to set permissions for %s: %v", target, err)
			}
			logrus.WithField("func", fn).Debugf("Extracted file: %s", target)

		case tar.TypeSymlink:
			// Validate symlink target to prevent path traversal
			// Reject absolute symlink targets
			if filepath.IsAbs(header.Linkname) {
				logrus.WithField("func", fn).Warnf("Skipping symlink with absolute target: %s -> %s", header.Name, header.Linkname)
				break
			}

			// For relative symlinks, resolve where they would point to
			// The symlink target is relative to the directory containing the symlink
			symlinkTargetPath := filepath.Join(filepath.Dir(target), header.Linkname)
			withinRoot, err := isPathWithinRoot(destDir, symlinkTargetPath)
			if err != nil {
				logrus.WithField("func", fn).Warnf("Failed to validate symlink target %s -> %s: %v", header.Name, header.Linkname, err)
				break
			}
			if !withinRoot {
				logrus.WithField("func", fn).Warnf("Skipping symlink pointing outside extraction root: %s -> %s", header.Name, header.Linkname)
				break
			}

			// Create parent directory if it doesn't exist
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for symlink %s: %w", target, err)
			}

			// Remove existing file/symlink if it exists
			os.Remove(target)

			// Create the symlink
			if err := os.Symlink(header.Linkname, target); err != nil {
				logrus.WithField("func", fn).Warnf("Failed to create symlink %s -> %s: %v", target, header.Linkname, err)
			} else {
				logrus.WithField("func", fn).Debugf("Created symlink: %s -> %s", target, header.Linkname)
			}

		case tar.TypeLink:
			// Construct the link target path
			linkTarget := filepath.Join(destDir, header.Linkname)

			// Validate hard link target to prevent path traversal
			withinRoot, err := isPathWithinRoot(destDir, linkTarget)
			if err != nil {
				logrus.WithField("func", fn).Warnf("Failed to validate hard link target %s -> %s: %v", header.Name, header.Linkname, err)
				break
			}
			if !withinRoot {
				logrus.WithField("func", fn).Warnf("Skipping hard link pointing outside extraction root: %s -> %s", header.Name, header.Linkname)
				break
			}

			// Create parent directory if it doesn't exist
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for hard link %s: %w", target, err)
			}

			// Remove existing file/link if it exists
			os.Remove(target)

			// Create the hard link
			if err := os.Link(linkTarget, target); err != nil {
				logrus.WithField("func", fn).Warnf("Failed to create hard link %s -> %s: %v", target, linkTarget, err)
			} else {
				logrus.WithField("func", fn).Debugf("Created hard link: %s -> %s", target, linkTarget)
			}

		default:
			logrus.WithField("func", fn).Warnf("Unsupported file type %v for %s", header.Typeflag, header.Name)
		}
	}

	logrus.WithFields(logrus.Fields{
		"func":     fn,
		"dest_dir": destDir,
	}).Info("Successfully completed tar extraction")

	return nil
}
