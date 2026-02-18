// Copyright © 2024 The vjailbreak authors

package esxissh

import (
	"context"
	"encoding/xml"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/platform9/vjailbreak/v2v-helper/pkg/utils"
	"golang.org/x/crypto/ssh"
)

// SSHOperation represents the type of SSH operation
type SSHOperation string

const (
	SSHOperationClone   SSHOperation = "clone"
	SSHOperationStatus  SSHOperation = "status"
	SSHOperationCleanup SSHOperation = "cleanup"
)

// VmkfstoolsTask represents the result of a vmkfstools operation
type VmkfstoolsTask struct {
	TaskId            string `json:"taskId"`
	Pid               int    `json:"pid"`
	ExitCode          string `json:"exitCode"`
	LastLine          string `json:"lastLine"`
	Stderr            string `json:"stdErr"`
	LogFile           string `json:"logFile"`
	RDMDescriptorPath string `json:"rdmDescriptorPath"`
}

// XMLResponse represents the XML response structure
type XMLResponse struct {
	XMLName   xml.Name  `xml:"o"`
	Structure Structure `xml:"structure"`
}

// Structure represents the structure element in the XML response
type Structure struct {
	TypeName string  `xml:"typeName,attr"`
	Fields   []Field `xml:"field"`
}

// Field represents a field in the XML response
type Field struct {
	Name   string `xml:"name,attr"`
	String string `xml:"string"`
}

type Client struct {
	hostname       string
	username       string
	sshClient      *ssh.Client
	commandTimeout time.Duration
}

func NewClient() *Client {
	return &Client{
		commandTimeout: 5 * time.Minute, // Increased for long-running operations
	}
}

func NewClientWithTimeout(timeout time.Duration) *Client {
	return &Client{
		commandTimeout: timeout,
	}
}

func (c *Client) SetCommandTimeout(timeout time.Duration) {
	c.commandTimeout = timeout
}

func (c *Client) Connect(ctx context.Context, hostname, username string, privateKey []byte) error {
	c.hostname = hostname
	c.username = username

	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		// WARNING: InsecureIgnoreHostKey bypasses host key verification.
		// This is acceptable for ESXi hosts which typically use self-signed certificates,
		// but in a production environment with higher security requirements, consider
		// implementing proper host key verification using ssh.FixedHostKey().
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         2 * time.Minute, // Increased timeout for ESXi connections
	}

	addr := net.JoinHostPort(hostname, "22")
	dialer := &net.Dialer{}
	netConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server: %w", err)
	}

	if deadline, ok := ctx.Deadline(); ok {
		if err := netConn.SetDeadline(deadline); err != nil {
			_ = netConn.Close()
			return fmt.Errorf("failed to set connection deadline: %w", err)
		}
	}

	cc, chans, reqs, err := ssh.NewClientConn(netConn, addr, config)
	if err != nil {
		_ = netConn.Close()
		return fmt.Errorf("failed to establish SSH client connection: %w", err)
	}
	c.sshClient = ssh.NewClient(cc, chans, reqs)
	return nil
}

func (c *Client) Disconnect() error {
	if c.sshClient == nil {
		return nil
	}

	err := c.sshClient.Close()
	c.sshClient = nil

	return err
}

func (c *Client) IsConnected() bool {
	return c.sshClient != nil
}

// ExecuteCommand runs a command on the ESXi host and returns stdout
// Uses the client's configured command timeout
func (c *Client) ExecuteCommand(command string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.commandTimeout)
	defer cancel()
	return c.ExecuteCommandWithContext(ctx, command)
}

// ExecuteCommandWithContext runs a command with a custom context (for timeout control)
func (c *Client) ExecuteCommandWithContext(ctx context.Context, command string) (string, error) {
	if c.sshClient == nil {
		return "", fmt.Errorf("not connected to ESXi host")
	}

	session, err := c.sshClient.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Channel to receive command result
	type result struct {
		output []byte
		err    error
	}
	resultChan := make(chan result, 1)

	// Run command in goroutine
	go func() {
		output, err := session.CombinedOutput(command)
		resultChan <- result{output: output, err: err}
	}()

	// Wait for either command completion or context cancellation/timeout
	select {
	case <-ctx.Done():
		// Try to kill the session
		_ = session.Signal(ssh.SIGKILL)
		_ = session.Close()
		return "", fmt.Errorf("command cancelled or timed out: %w", ctx.Err())
	case res := <-resultChan:
		if res.err != nil {
			return string(res.output), fmt.Errorf("command failed: %w", res.err)
		}
		return string(res.output), nil
	}
}

func (c *Client) TestConnection() error {
	if c.sshClient == nil {
		return fmt.Errorf("not connected")
	}

	// Run a simple command to test
	output, err := c.ExecuteCommand("esxcli system version get")
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	if output == "" {
		return fmt.Errorf("connection test returned no output")
	}

	return nil
}

// StartVmkfstoolsClone starts a vmkfstools clone operation from source VMDK to target LUN
// This will automatically use StorageAcceleratedCopy if available on the storage array
// The clone runs in the background and returns immediately
func (c *Client) StartVmkfstoolsClone(sourceVMDK, targetLUN string) (*VmkfstoolsTask, error) {
	utils.PrintLog(fmt.Sprintf("Starting vmkfstools clone: source=%s, target=%s", sourceVMDK, targetLUN))

	// First check if source exists
	checkCmd := fmt.Sprintf("ls -l %s 2>&1", sourceVMDK)
	checkOutput, _ := c.ExecuteCommand(checkCmd)
	if !strings.Contains(checkOutput, sourceVMDK) {
		return nil, fmt.Errorf("source VMDK does not exist: %s", sourceVMDK)
	}

	// Create target directory if it doesn't exist
	targetDir := targetLUN[:strings.LastIndex(targetLUN, "/")]
	mkdirCmd := fmt.Sprintf("mkdir -p %s", targetDir)
	_, err := c.ExecuteCommand(mkdirCmd)
	if err != nil {
		utils.PrintLog(fmt.Sprintf("WARNING: Failed to create target directory: %v", err))
	} else {
		utils.PrintLog(fmt.Sprintf("Created target directory: %s", targetDir))
	}

	// Create a log file for vmkfstools output so we can debug failures
	logFile := fmt.Sprintf("/tmp/vmkfstools_clone_%d.log", time.Now().Unix())

	// Run vmkfstools in background
	// Use vmkfstools -i (clone/import) which automatically uses XCOPY on StorageAcceleratedCopy-capable storage
	// -d thin creates thin provisioned disk (can also use eagerzeroedthick, zeroedthick)
	// Capture output to log file for debugging
	command := fmt.Sprintf("vmkfstools -i %s %s -d thin >%s 2>&1 & echo $!", sourceVMDK, targetLUN, logFile)

	output, err := c.ExecuteCommand(command)
	if err != nil {
		return nil, fmt.Errorf("failed to start clone: %w", err)
	}

	pid := strings.TrimSpace(output)
	utils.PrintLog(fmt.Sprintf("Started vmkfstools clone with PID: %s, log: %s", pid, logFile))

	// Return task info with PID so we can check status later
	task := &VmkfstoolsTask{
		TaskId:   fmt.Sprintf("vmkfstools-clone-%s", pid),
		Pid:      0, // Will be parsed from output
		LastLine: fmt.Sprintf("Clone started with PID %s, log: %s", pid, logFile),
	}

	// Try to parse PID
	if pidInt, err := fmt.Sscanf(pid, "%d", &task.Pid); err == nil && pidInt == 1 {
		utils.PrintLog(fmt.Sprintf("Parsed PID: %d", task.Pid))
	}

	return task, nil
}

// convertDatastorePathToFilesystemPath converts a vSphere datastore path to ESXi filesystem path
// e.g., "[datastore-name] vm-folder/vm.vmdk" -> "/vmfs/volumes/datastore-name/vm-folder/vm.vmdk"
// If the path is already a filesystem path (starts with /), it's returned unchanged
func convertDatastorePathToFilesystemPath(datastorePath string) string {
	datastorePath = strings.TrimSpace(datastorePath)

	// If already a filesystem path, return as-is
	if strings.HasPrefix(datastorePath, "/") {
		return datastorePath
	}

	// Parse vSphere datastore path format: [datastore-name] path/to/file.vmdk
	if !strings.HasPrefix(datastorePath, "[") {
		// Not a datastore path format, return as-is
		return datastorePath
	}

	// Find the closing bracket
	closeBracket := strings.Index(datastorePath, "]")
	if closeBracket == -1 {
		return datastorePath
	}

	// Extract datastore name (without brackets)
	datastoreName := datastorePath[1:closeBracket]

	// Extract the path after the datastore name (skip the space after ])
	remainingPath := strings.TrimSpace(datastorePath[closeBracket+1:])

	// Construct the filesystem path
	return fmt.Sprintf("/vmfs/volumes/%s/%s", datastoreName, remainingPath)
}

// StartVmkfstoolsRDMClone starts a vmkfstools clone operation from source VMDK to target raw device (RDM)
// This uses StorageAcceleratedCopy XCOPY to clone directly to a raw device without creating a datastore
// Command format: vmkfstools -i <source> -d rdm:<target_device> <rdm_descriptor_vmdk>
func (c *Client) StartVmkfstoolsRDMClone(sourceVMDK, targetDevicePath string) (*VmkfstoolsTask, error) {
	utils.PrintLog("=== Starting vmkfstools RDM clone ===")
	utils.PrintLog(fmt.Sprintf("Input source VMDK: %s", sourceVMDK))
	utils.PrintLog(fmt.Sprintf("Input target device path: %s", targetDevicePath))

	// Convert vSphere datastore path to ESXi filesystem path if needed
	// e.g., "[datastore-name] vm-folder/vm.vmdk" -> "/vmfs/volumes/datastore-name/vm-folder/vm.vmdk"
	sourceVMDKConverted := convertDatastorePathToFilesystemPath(sourceVMDK)
	utils.PrintLog(fmt.Sprintf("Source VMDK path (after conversion): %s", sourceVMDKConverted))

	// First check if source exists
	utils.PrintLog("Checking if source VMDK exists...")
	checkCmd := fmt.Sprintf("ls -l '%s' 2>&1", sourceVMDKConverted)
	utils.PrintLog(fmt.Sprintf("Executing: %s", checkCmd))
	checkOutput, _ := c.ExecuteCommand(checkCmd)
	utils.PrintLog(fmt.Sprintf("Source check output: %s", checkOutput))
	if strings.Contains(checkOutput, "No such file") {
		return nil, fmt.Errorf("source VMDK does not exist: %s (output: %s)", sourceVMDKConverted, checkOutput)
	}
	utils.PrintLog("Source VMDK exists")

	// Verify target device exists
	utils.PrintLog("Checking if target device exists...")
	checkDevCmd := fmt.Sprintf("ls -l %s 2>&1", targetDevicePath)
	utils.PrintLog(fmt.Sprintf("Executing: %s", checkDevCmd))
	checkDevOutput, _ := c.ExecuteCommand(checkDevCmd)
	utils.PrintLog(fmt.Sprintf("Target device check output: %s", checkDevOutput))
	if !strings.Contains(checkDevOutput, targetDevicePath) {
		return nil, fmt.Errorf("target device does not exist: %s", targetDevicePath)
	}
	utils.PrintLog("Target device exists")

	// Derive RDM descriptor path on the same datastore as source VMDK
	// Add timestamp to avoid conflicts on retry attempts (issue #1423)
	// e.g., /vmfs/volumes/pure-ds/pure-clone1/pure-clone1.vmdk -> /vmfs/volumes/pure-ds/pure-clone1/pure-clone1-rdm-1738082400.vmdk
	sourceDir := sourceVMDKConverted[:strings.LastIndex(sourceVMDKConverted, "/")]
	sourceBaseName := sourceVMDKConverted[strings.LastIndex(sourceVMDKConverted, "/")+1:]
	sourceNameWithoutExt := strings.TrimSuffix(sourceBaseName, ".vmdk")
	timestamp := time.Now().Unix()
	rdmDescriptor := fmt.Sprintf("%s/%s-rdm-%d.vmdk", sourceDir, sourceNameWithoutExt, timestamp)
	utils.PrintLog(fmt.Sprintf("Source directory: %s", sourceDir))
	utils.PrintLog(fmt.Sprintf("Source base name: %s", sourceBaseName))
	utils.PrintLog(fmt.Sprintf("RDM descriptor path (on datastore): %s", rdmDescriptor))

	// With timestamp-based naming, each retry uses a unique RDM descriptor filename
	// This prevents "file already exists" errors on migration retries (issue #1423)
	utils.PrintLog(fmt.Sprintf("Using unique RDM descriptor path with timestamp: %s", rdmDescriptor))

	// Create a log file for vmkfstools output
	logFile := fmt.Sprintf("/tmp/vmkfstools_rdm_clone_%d.log", time.Now().Unix())
	utils.PrintLog(fmt.Sprintf("Log file: %s", logFile))

	// Build the vmkfstools command
	// vmkfstools -i <source> -d rdm:<device> <rdm_descriptor>
	vmkfstoolsCmd := fmt.Sprintf("vmkfstools -i %s -d rdm:%s %s", sourceVMDKConverted, targetDevicePath, rdmDescriptor)
	utils.PrintLog("=== vmkfstools command ===")
	utils.PrintLog(vmkfstoolsCmd)
	utils.PrintLog("==========================")

	// Run vmkfstools in background
	command := fmt.Sprintf("%s >%s 2>&1 & echo $!", vmkfstoolsCmd, logFile)
	utils.PrintLog(fmt.Sprintf("Executing background command: %s", command))

	output, err := c.ExecuteCommand(command)
	if err != nil {
		utils.PrintLog(fmt.Sprintf("ERROR: Failed to start RDM clone: %v", err))
		return nil, fmt.Errorf("failed to start RDM clone: %w", err)
	}

	pid := strings.TrimSpace(output)
	utils.PrintLog(fmt.Sprintf("vmkfstools started with PID: %s", pid))
	utils.PrintLog(fmt.Sprintf("Log file location: %s", logFile))

	// Return task info with PID so we can check status later
	task := &VmkfstoolsTask{
		TaskId:            fmt.Sprintf("vmkfstools-rdm-clone-%s", pid),
		Pid:               0, // Will be parsed from output
		LastLine:          fmt.Sprintf("RDM clone started with PID %s, log: %s", pid, logFile),
		LogFile:           logFile,
		RDMDescriptorPath: rdmDescriptor,
	}

	// Try to parse PID
	if pidInt, err := fmt.Sscanf(pid, "%d", &task.Pid); err == nil && pidInt == 1 {
		utils.PrintLog(fmt.Sprintf("Parsed PID: %d", task.Pid))
	}

	utils.PrintLog("=== vmkfstools RDM clone started successfully ===")
	return task, nil
}

// GetCloneLog retrieves the vmkfstools clone log output
func (c *Client) GetCloneLog(pid int) (string, error) {
	if c.sshClient == nil {
		return "", fmt.Errorf("not connected to ESXi host")
	}

	// Try to find the log file for this PID
	// We use a pattern since we don't know the exact timestamp
	cmd := fmt.Sprintf("cat /tmp/vmkfstools_clone_*.log 2>/dev/null | tail -n 100")
	output, _ := c.ExecuteCommand(cmd)

	return output, nil
}

// CheckCloneStatus checks if a clone operation is still running
func (c *Client) CheckCloneStatus(pid int) (bool, error) {
	// ESXi uses BusyBox ps which doesn't support -p flag
	// Use kill -0 to check if process exists (doesn't actually send a signal)
	// Also try ps as fallback since kill -0 may not work reliably over SSH
	checkCmd := fmt.Sprintf("kill -0 %d 2>/dev/null && echo 'running' || ps -c | grep -w %d | grep -v grep && echo 'running'", pid, pid)
	output, _ := c.ExecuteCommand(checkCmd)

	isRunning := strings.Contains(output, "running")

	return isRunning, nil
}

// VerifyVMDKClone verifies that a cloned VMDK was created successfully
func (c *Client) VerifyVMDKClone(vmdkPath string) error {
	if c.sshClient == nil {
		return fmt.Errorf("not connected to ESXi host")
	}

	// Use test -f to check if file exists (more reliable than ls)
	descCmd := fmt.Sprintf("test -f %s && echo 'exists' || echo 'missing'", vmdkPath)
	descOutput, err := c.ExecuteCommand(descCmd)
	if err != nil || !strings.Contains(descOutput, "exists") {
		// Try with ls as fallback
		lsOutput, _ := c.ExecuteCommand(fmt.Sprintf("ls -l %s 2>&1", vmdkPath))
		return fmt.Errorf("descriptor file not found at %s (output: %s)", vmdkPath, lsOutput)
	}

	// Check flat file exists
	flatFile := strings.Replace(vmdkPath, ".vmdk", "-flat.vmdk", 1)
	flatCmd := fmt.Sprintf("test -f %s && echo 'exists' || echo 'missing'", flatFile)
	flatOutput, err := c.ExecuteCommand(flatCmd)
	if err != nil || !strings.Contains(flatOutput, "exists") {
		// Try with ls as fallback
		lsOutput, _ := c.ExecuteCommand(fmt.Sprintf("ls -l %s 2>&1", flatFile))
		return fmt.Errorf("flat file not found at %s (output: %s)", flatFile, lsOutput)
	}

	utils.PrintLog(fmt.Sprintf("VMDK clone verified: descriptor and flat files exist at %s", vmdkPath))
	return nil
}

// GetVMDKSize returns the size of a VMDK in bytes by reading the descriptor
func (c *Client) GetVMDKSize(vmdkPath string) (int64, error) {
	if c.sshClient == nil {
		return 0, fmt.Errorf("not connected to ESXi host")
	}

	// Read VMDK descriptor to find size
	descriptor, err := c.ExecuteCommand(fmt.Sprintf("cat %s 2>/dev/null", vmdkPath))
	if err != nil || descriptor == "" {
		return 0, fmt.Errorf("failed to read VMDK descriptor: %w", err)
	}

	// Look for lines like: RW 8388608 VMFS "disk-flat.vmdk"
	// The number is in 512-byte sectors
	for _, line := range strings.Split(descriptor, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "-flat.vmdk") && strings.Contains(line, "VMFS") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if sectors, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
					utils.PrintLog(fmt.Sprintf("Found VMDK size in descriptor: %d sectors (%d bytes)", sectors, sectors*512))
					return sectors * 512, nil // Convert sectors to bytes
				}
			}
		}
	}

	return 0, fmt.Errorf("could not find size in VMDK descriptor")
}

// CheckVMDKExists checks if a VMDK already exists at the given path
func (c *Client) CheckVMDKExists(vmdkPath string) (bool, error) {
	if c.sshClient == nil {
		return false, fmt.Errorf("not connected to ESXi host")
	}

	output, _ := c.ExecuteCommand(fmt.Sprintf("ls -l %s 2>/dev/null", vmdkPath))
	return output != "", nil
}

// DeleteVMDKFiles removes a VMDK and its flat file (for cleanup before clone)
func (c *Client) DeleteVMDKFiles(vmdkPath string) error {
	if c.sshClient == nil {
		return fmt.Errorf("not connected to ESXi host")
	}

	// Delete flat file
	flatFile := strings.Replace(vmdkPath, ".vmdk", "-flat.vmdk", 1)
	_, _ = c.ExecuteCommand(fmt.Sprintf("rm -f %s", flatFile))

	// Delete descriptor
	_, err := c.ExecuteCommand(fmt.Sprintf("rm -f %s", vmdkPath))
	if err != nil {
		return fmt.Errorf("failed to delete VMDK files: %w", err)
	}

	utils.PrintLog(fmt.Sprintf("Deleted existing VMDK files at %s", vmdkPath))
	return nil
}

// CheckVMKernelLogsForXCopy checks vmkernel logs for StorageAcceleratedCopy XCOPY operations
// Returns recent log lines containing XCOPY/StorageAcceleratedCopy indicators
func (c *Client) CheckVMKernelLogsForXCopy(timeWindowMinutes int) (string, error) {
	if c.sshClient == nil {
		return "", fmt.Errorf("not connected to ESXi host")
	}

	// Search for XCOPY-related keywords in vmkernel logs
	// Common indicators: "XCOPY", "StorageAcceleratedCopy", "ExtendedCopy", "vmw_ahci"
	cmd := fmt.Sprintf("tail -n 1000 /var/log/vmkernel.log | grep -i -E 'xcopy|StorageAcceleratedCopy|extendedcopy|clone' | tail -n 50")

	output, err := c.ExecuteCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to check vmkernel logs: %w", err)
	}

	return output, nil
}

// CheckStorageIOStats checks storage I/O statistics to verify StorageAcceleratedCopy usage
func (c *Client) CheckStorageIOStats(deviceName string) (string, error) {
	if c.sshClient == nil {
		return "", fmt.Errorf("not connected to ESXi host")
	}

	// Get StorageAcceleratedCopy statistics for a device
	cmd := fmt.Sprintf("esxcli storage core device vaai status get -d %s", deviceName)

	output, err := c.ExecuteCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to get StorageAcceleratedCopy stats: %w", err)
	}

	return output, nil
}

// RunEsxcliCommand runs an esxcli command with XML formatter and returns parsed results
// Returns a slice of maps, where each map represents a row with column name -> value
func (c *Client) RunEsxcliCommand(namespace string, args []string) ([]map[string]string, error) {
	if c.sshClient == nil {
		return nil, fmt.Errorf("not connected to ESXi host")
	}

	// Build command with XML formatter for structured output
	cmd := fmt.Sprintf("esxcli --formatter=xml %s %s", namespace, strings.Join(args, " "))
	utils.PrintLog(fmt.Sprintf("Running esxcli command: %s", cmd))

	output, err := c.ExecuteCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("esxcli command failed: %w", err)
	}

	return parseEsxcliXMLOutput(output)
}

// EsxcliListResponse represents the XML list response from esxcli
// Structure: <output><root><list><structure>...</structure></list></root></output>
type EsxcliListResponse struct {
	XMLName xml.Name   `xml:"output"`
	Root    EsxcliRoot `xml:"root"`
}

// EsxcliRoot represents the root element
type EsxcliRoot struct {
	List EsxcliStructureList `xml:"list"`
}

// EsxcliStructureList represents a list of structures
type EsxcliStructureList struct {
	Structures []EsxcliStructure `xml:"structure"`
}

// EsxcliStructure represents a single structure with fields
type EsxcliStructure struct {
	Fields []EsxcliField `xml:"field"`
}

// EsxcliField represents a field in the structure
type EsxcliField struct {
	Name   string `xml:"name,attr"`
	String string `xml:"string"`
}

// parseEsxcliXMLOutput parses XML output from esxcli --formatter=xml
func parseEsxcliXMLOutput(output string) ([]map[string]string, error) {
	var resp EsxcliListResponse
	if err := xml.Unmarshal([]byte(output), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse esxcli XML output: %w", err)
	}

	var results []map[string]string
	for _, structure := range resp.Root.List.Structures {
		row := make(map[string]string)
		for _, field := range structure.Fields {
			row[field.Name] = field.String
		}
		results = append(results, row)
	}

	return results, nil
}

// parseTaskResponse parses the XML response from vmkfstools operations
func parseTaskResponse(output string) (*VmkfstoolsTask, error) {
	var xmlResp XMLResponse
	if err := xml.Unmarshal([]byte(output), &xmlResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML: %w", err)
	}

	task := &VmkfstoolsTask{}
	for _, field := range xmlResp.Structure.Fields {
		switch field.Name {
		case "taskId":
			task.TaskId = field.String
		case "pid":
			var pid int
			if _, err := fmt.Sscanf(field.String, "%d", &pid); err == nil {
				task.Pid = pid
			}
		case "exitCode":
			task.ExitCode = field.String
		case "lastLine":
			task.LastLine = field.String
		case "stdErr":
			task.Stderr = field.String
		}
	}

	return task, nil
}
