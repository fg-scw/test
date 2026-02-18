package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	api "github.com/platform9/vjailbreak/pkg/vpwned/api/proto/v1/service"
	"github.com/platform9/vjailbreak/pkg/vpwned/sdk/providers"
	"github.com/platform9/vjailbreak/pkg/vpwned/utils/tableprinter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var creds = providers.BMAccessInfo{}
var currProvider providers.BMCProvider

func populateBMCredsFromCMD(cmd *cobra.Command) {
	if val, err := cmd.Flags().GetString("api_key"); err == nil {
		creds.APIKey = val
	}
	if val, err := cmd.Flags().GetString("base_url"); err == nil {
		creds.BaseURL = val
	}
	if val, err := cmd.Flags().GetBool("use_insecure"); err == nil {
		creds.UseInsecure = val
	}
	if val, err := cmd.Flags().GetString("username"); err == nil {
		creds.Username = val
	}
	if val, err := cmd.Flags().GetString("password"); err == nil {
		creds.Password = val
	}
}

func initProvider(name string) {
	logrus.Debug("initializing provider: ", name)
	cp, err := providers.GetProvider(name)
	if err != nil {
		logrus.Error(err)
		return
	}
	currProvider = cp
}

var providerCmd = &cobra.Command{
	Use:   "maas",
	Short: "target provider supported provider: maas",
	Long:  "target provider to fetch provider details and manage provider resources",
	Run: func(cmd *cobra.Command, args []string) {
		populateBMCredsFromCMD(cmd)
		cp, err := providers.GetProvider("maas")
		if err != nil {
			logrus.Error(err)
			fmt.Println(cmd.UsageString())
			return
		}
		currProvider = cp
	},
}

var listProvidersCmd = &cobra.Command{
	Use:   "list",
	Short: "list registered providers",
	Long:  "list registered providers",
	Run: func(cmd *cobra.Command, args []string) {
		populateBMCredsFromCMD(cmd)
		initProvider(cmd.Parent().Use)
		res := providers.GetProviders()
		for k, v := range res {
			fmt.Println(k, v)
		}
	},
}

var connectProviderCmd = &cobra.Command{
	Use:   "connect",
	Short: "connect to provider",
	Long:  "connect to provider",
	Run: func(cmd *cobra.Command, args []string) {
		populateBMCredsFromCMD(cmd)
		initProvider(cmd.Parent().Use)
		if currProvider == nil {
			logrus.Error("provider not found")
			fmt.Println(cmd.UsageString())
			return
		}
		err := currProvider.Connect(creds)
		if err != nil {
			logrus.Error(err)
			return
		}
		fmt.Printf("Connected to %s on %s\n", currProvider.WhoAmI(), creds.BaseURL)
	},
}

// var disconnectProviderCmd = &cobra.Command{
// 	Use:   "disconnect",
// 	Short: "disconnect from provider",
// 	Long:  "disconnect from provider",
// 	Run: func(cmd *cobra.Command, args []string) {
// 		populateCredsFromCMD(cmd)
// 		providers := providers.Providers{}
// 		err := providers.Disconnect(Creds)
// 		if err != nil {
// 			logrus.Error(err)
// 		}
// 	},
// }

var setProviderPowerCmd = &cobra.Command{
	Use:   "set_power",
	Short: "set provider power",
	Long:  "set provider power",
	Run: func(cmd *cobra.Command, args []string) {
		populateBMCredsFromCMD(cmd)
		initProvider(cmd.Parent().Use)
		var machine_id string
		var action int32
		if val, err := cmd.Flags().GetString("machine_id"); err == nil {
			machine_id = val
		}
		if val, err := cmd.Flags().GetString("action"); err == nil {
			action = api.PowerStatus_value[val]
		}
		if currProvider == nil {
			logrus.Error("provider not found")
			fmt.Println(cmd.UsageString())
			return
		}
		err := currProvider.Connect(creds)
		if err != nil {
			logrus.Error(err)
		}
		err = currProvider.SetResourcePower(context.Background(), machine_id, api.PowerStatus(action))
		if err != nil {
			logrus.Error(err)
		}
	},
}

var listProviderResourcesCmd = &cobra.Command{
	Use:   "list_resources",
	Short: "list provider resources",
	Long:  "list provider resources",
	Run: func(cmd *cobra.Command, args []string) {
		populateBMCredsFromCMD(cmd)
		initProvider(cmd.Parent().Use)
		if currProvider == nil {
			logrus.Error("provider not found")
			fmt.Println(cmd.UsageString())
			return
		}
		err := currProvider.Connect(creds)
		if err != nil {
			logrus.Error(err)
			return
		}

		logrus.Infof("Listing resources")
		res, err := currProvider.ListResources(context.Background())
		if err != nil {
			logrus.Error(err)
			return
		}
		tableprinter.PrintAsTable(res, "Id", "Fqdn", "PowerState", "Hostname", "Status", "MacAddress", "HardwareUuid")
	},
}

var getResourceInfoCMD = &cobra.Command{
	Use:   "get_resource_info",
	Short: "get provider resource info",
	Long:  "get provider resource info",
	Run: func(cmd *cobra.Command, args []string) {
		populateBMCredsFromCMD(cmd)
		initProvider(cmd.Parent().Use)
		var resource_id string
		if val, err := cmd.Flags().GetString("resource_id"); err == nil {
			resource_id = val
		}
		if resource_id == "" {
			logrus.Error("resource_id is required")
			fmt.Println(cmd.UsageString())
			return
		}
		if currProvider == nil {
			logrus.Error("provider not found")
			fmt.Println(cmd.UsageString())
			return
		}
		err := currProvider.Connect(creds)
		if err != nil {
			logrus.Error(err)
		}

		logrus.Infof("Getting resource info")
		res, err := currProvider.GetResourceInfo(context.Background(), resource_id)
		if err != nil {
			logrus.Error(err)
			return
		}
		fmt.Printf("%+v\n", res)
	},
}

var listBootSourceCmd = &cobra.Command{
	Use:   "list_boot_source",
	Short: "list provider boot source",
	Long:  "list provider boot source",
	Run: func(cmd *cobra.Command, args []string) {
		populateBMCredsFromCMD(cmd)
		initProvider(cmd.Parent().Use)
		if currProvider == nil {
			logrus.Error("provider not found")
			fmt.Println(cmd.UsageString())
			return
		}
		err := currProvider.Connect(creds)
		if err != nil {
			logrus.Error(err)
		}

		logrus.Infof("Listing boot sources")
		res, err := currProvider.ListBootSource(context.Background(), api.ListBootSourceRequest{
			AccessInfo: &api.BMProvisionerAccessInfo{
				BaseUrl:     creds.BaseURL,
				ApiKey:      creds.APIKey,
				UseInsecure: creds.UseInsecure,
			},
		})
		if err != nil {
			logrus.Error(err)
		}
		b, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			logrus.Errorf("Failed to marshal boot source info: %v", err)
			return
		}
		fmt.Println(string(b))
	},
}

var reclaimBMCmd = &cobra.Command{
	Use:   "reclaim",
	Short: "reclaim provider resource",
	Long:  "reclaim provider resource",
	Run: func(cmd *cobra.Command, args []string) {
		populateBMCredsFromCMD(cmd)
		initProvider(cmd.Parent().Use)
		var resource_id, user_data string
		erase_disk := false
		manual_power_control := false
		boot_source_id := int32(0)
		if val, err := cmd.Flags().GetString("resource_id"); err == nil {
			resource_id = val
		}
		if resource_id == "" {
			logrus.Error("resource_id is required")
			fmt.Println(cmd.UsageString())
			return
		}
		if val, err := cmd.Flags().GetBool("erase_disk"); err == nil {
			erase_disk = val
		}
		if val, err := cmd.Flags().GetBool("manual_power_control"); err == nil {
			manual_power_control = val
		}
		if val, err := cmd.Flags().GetString("user_data"); err == nil {
			user_data = val
		}
		if val, err := cmd.Flags().GetInt("boot_source_id"); err == nil {
			boot_source_id = int32(val)
		}
		var ipmi_interface *api.IpmiType
		if val, err := cmd.Flags().GetString("ipmi_interface"); err == nil {
			switch strings.ToLower(val) {
			case "lan":
				ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_Lan{}}
			case "lanplus":
				ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_Lanplus{}}
			case "openipmi":
				ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_OpenIpmi{}}
			case "tool":
				ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_Tool{}}
			}
		}
		if currProvider == nil {
			logrus.Error("provider not found")
			fmt.Println(cmd.UsageString())
			return
		}
		err := currProvider.Connect(creds)
		if err != nil {
			logrus.Error(err)
		}

		logrus.Infof("Reclaiming resource %s", resource_id)
		err = currProvider.ReclaimBM(context.Background(), api.ReclaimBMRequest{
			AccessInfo: &api.BMProvisionerAccessInfo{
				BaseUrl:     creds.BaseURL,
				ApiKey:      creds.APIKey,
				UseInsecure: creds.UseInsecure,
			},
			ResourceId:         resource_id,
			EraseDisk:          erase_disk,
			UserData:           user_data,
			ManualPowerControl: manual_power_control,
			BootSource: &api.BootsourceSelections{
				BootSourceID: boot_source_id,
			},
			IpmiInterface: ipmi_interface,
		})
		if err != nil {
			logrus.Error(err)
		}
	},
}

var setProviderBootDeviceCmd = &cobra.Command{
	Use:   "set_boot_to_pxe",
	Short: "set provider boot device to pxe",
	Long:  "set provider boot device to pxe",
	Run: func(cmd *cobra.Command, args []string) {
		populateBMCredsFromCMD(cmd)
		initProvider(cmd.Parent().Use)
		var resource_id string
		power_cycle := false
		if val, err := cmd.Flags().GetString("resource_id"); err == nil {
			resource_id = val
		}
		if resource_id == "" {
			logrus.Error("resource_id is required")
			fmt.Println(cmd.UsageString())
			return
		}
		if val, err := cmd.Flags().GetBool("power_cycle"); err == nil {
			power_cycle = val
		}
		var ipmi_interface *api.IpmiType
		if val, err := cmd.Flags().GetString("ipmi_interface"); err == nil {
			switch strings.ToLower(val) {
			case "lan":
				ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_Lan{}}
			case "lanplus":
				ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_Lanplus{}}
			case "openipmi":
				ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_OpenIpmi{}}
			case "tool":
				ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_Tool{}}
			}
		}
		if currProvider == nil {
			logrus.Error("provider not found")
			fmt.Println(cmd.UsageString())
			return
		}
		err := currProvider.Connect(creds)
		if err != nil {
			logrus.Error(err)
		}
		err = currProvider.SetBM2PXEBoot(context.Background(), resource_id, power_cycle, ipmi_interface)
		if err != nil {
			logrus.Error(err)
		}
		logrus.Infof("Set boot device to PXE for resource %s", resource_id)
	},
}

var deployBMCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploy provider resource",
	Long:  "deploy provider resource",
	Run: func(cmd *cobra.Command, args []string) {
		populateBMCredsFromCMD(cmd)
		initProvider(cmd.Parent().Use)
		var resource_id, user_data string
		boot_source_id := "jammy"
		if val, err := cmd.Flags().GetString("resource_id"); err == nil {
			resource_id = val
		}
		if resource_id == "" {
			logrus.Error("resource_id is required")
			fmt.Println(cmd.UsageString())
			return
		}
		if val, err := cmd.Flags().GetString("user_data"); err == nil {
			user_data = val
		}
		if val, err := cmd.Flags().GetString("release_name"); err == nil {
			boot_source_id = val
		}
		if currProvider == nil {
			logrus.Error("provider not found")
			fmt.Println(cmd.UsageString())
			return
		}
		err := currProvider.Connect(creds)
		if err != nil {
			logrus.Error(err)
		}
		_, err = currProvider.DeployMachine(context.Background(), api.DeployMachineRequest{
			AccessInfo: &api.BMProvisionerAccessInfo{
				BaseUrl:     creds.BaseURL,
				ApiKey:      creds.APIKey,
				UseInsecure: creds.UseInsecure,
			},
			ResourceId:    resource_id,
			UserData:      user_data,
			OsReleaseName: boot_source_id,
		})
		if err != nil {
			logrus.Error(err)
		}
	},
}

var stopBMCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop provider resource",
	Long:  "stop provider resource",
	Run: func(cmd *cobra.Command, args []string) {
		populateBMCredsFromCMD(cmd)
		initProvider(cmd.Parent().Use)
		var resource_id string
		if val, err := cmd.Flags().GetString("resource_id"); err == nil {
			resource_id = val
		}
		if resource_id == "" {
			logrus.Error("resource_id is required")
			fmt.Println(cmd.UsageString())
			return
		}
		if currProvider == nil {
			logrus.Error("provider not found")
			fmt.Println(cmd.UsageString())
			return
		}
		err := currProvider.Connect(creds)
		if err != nil {
			logrus.Error(err)
		}
		var ipmi_interface *api.IpmiType
		if val, err := cmd.Flags().GetString("ipmi_interface"); err == nil {
			switch strings.ToLower(val) {
			case "lan":
				ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_Lan{}}
			case "lanplus":
				ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_Lanplus{}}
			case "openipmi":
				ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_OpenIpmi{}}
			case "tool":
				ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_Tool{}}
			}
		}
		_, err = currProvider.StopBM(context.Background(), api.StopBMRequest{
			AccessInfo: &api.BMProvisionerAccessInfo{
				BaseUrl:     creds.BaseURL,
				ApiKey:      creds.APIKey,
				UseInsecure: creds.UseInsecure,
			},
			ResourceId:    resource_id,
			IpmiInterface: ipmi_interface,
		})
		if err != nil {
			logrus.Error(err)
		}
	},
}

var startBMCmd = &cobra.Command{
	Use:   "start",
	Short: "start provider resource",
	Long:  "start provider resource",
	Run: func(cmd *cobra.Command, args []string) {
		populateBMCredsFromCMD(cmd)
		initProvider(cmd.Parent().Use)
		var resource_id string
		if val, err := cmd.Flags().GetString("resource_id"); err == nil {
			resource_id = val
		}
		if resource_id == "" {
			logrus.Error("resource_id is required")
			fmt.Println(cmd.UsageString())
			return
		}
		if currProvider == nil {
			logrus.Error("provider not found")
			fmt.Println(cmd.UsageString())
			return
		}
		err := currProvider.Connect(creds)
		if err != nil {
			logrus.Error(err)
		}
		var ipmi_interface *api.IpmiType
		if val, err := cmd.Flags().GetString("ipmi_interface"); err == nil {
			switch strings.ToLower(val) {
			case "lan":
				ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_Lan{}}
			case "lanplus":
				ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_Lanplus{}}
			case "openipmi":
				ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_OpenIpmi{}}
			case "tool":
				ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_Tool{}}
			}
		}
		_, err = currProvider.StartBM(context.Background(), api.StartBMRequest{
			AccessInfo: &api.BMProvisionerAccessInfo{
				BaseUrl:     creds.BaseURL,
				ApiKey:      creds.APIKey,
				UseInsecure: creds.UseInsecure,
			},
			ResourceId:    resource_id,
			IpmiInterface: ipmi_interface,
		})
		if err != nil {
			logrus.Error(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(providerCmd)
	providerCmd.PersistentFlags().StringP("api_key", "k", "", "Set the API key to use")
	providerCmd.PersistentFlags().StringP("base_url", "b", "", "Set the base URL to use")
	providerCmd.PersistentFlags().StringP("use_insecure", "i", "", "Set the datacenter to target")
	providerCmd.PersistentFlags().StringP("username", "U", "", "Set the username to use")
	providerCmd.PersistentFlags().StringP("password", "P", "", "Set the password to use")
	//set paramters for setPower
	setProviderPowerCmd.Flags().StringP("machine_id", "m", "", "Set the machine ID to use")
	setProviderPowerCmd.Flags().StringP("action", "a", "", "Set the power state to use")
	// get resource
	getResourceInfoCMD.Flags().StringP("resource_id", "r", "", "Set the resource ID to use")
	// set boot device to PXE for BM
	setProviderBootDeviceCmd.Flags().StringP("resource_id", "r", "", "Set the resource ID to use")
	setProviderBootDeviceCmd.Flags().BoolP("power_cycle", "p", false, "Set the power cycle to use")
	setProviderBootDeviceCmd.Flags().StringP("ipmi_interface", "n", "lanplus", "Set the IPMI interface to use")
	// reclaim cmd
	reclaimBMCmd.Flags().StringP("resource_id", "r", "", "Set the resource ID to use")
	reclaimBMCmd.Flags().StringP("user_data", "d", "", "Set the user data to use")
	reclaimBMCmd.Flags().BoolP("erase_disk", "e", false, "Set the erase disk to use")
	reclaimBMCmd.Flags().IntP("boot_source_id", "s", 0, "Set the boot source ID to use")
	reclaimBMCmd.Flags().BoolP("manual_power_control", "m", false, "Set the manual power control to use")
	reclaimBMCmd.Flags().StringP("ipmi_interface", "n", "lanplus", "Set the IPMI interface to use")
	//deployBMCmd
	deployBMCmd.Flags().StringP("resource_id", "r", "", "Set the resource ID to use")
	deployBMCmd.Flags().StringP("user_data", "d", "", "Set the user data to use")
	deployBMCmd.Flags().StringP("os_release_name", "o", "", "Set the OS release name to use")
	//startBMCmd
	startBMCmd.Flags().StringP("resource_id", "r", "", "Set the resource ID to use")
	startBMCmd.Flags().StringP("ipmi_interface", "n", "lanplus", "Set the IPMI interface to use")
	//stopBMCmd
	stopBMCmd.Flags().StringP("resource_id", "r", "", "Set the resource ID to use")
	stopBMCmd.Flags().StringP("ipmi_interface", "n", "lanplus", "Set the IPMI interface to use")
	//add commands
	providerCmd.AddCommand(listBootSourceCmd, listProvidersCmd, connectProviderCmd,
		setProviderPowerCmd, listProviderResourcesCmd, getResourceInfoCMD,
		reclaimBMCmd, setProviderBootDeviceCmd, deployBMCmd, startBMCmd, stopBMCmd)
}
