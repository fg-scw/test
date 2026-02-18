package cli

import (
	"context"
	"fmt"

	"github.com/platform9/vjailbreak/pkg/vpwned/api/proto/v1/service"
	"github.com/platform9/vjailbreak/pkg/vpwned/sdk/targets/vcenter"
	"github.com/platform9/vjailbreak/pkg/vpwned/utils/tableprinter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var Creds = service.TargetAccessInfo{}

func populateCredsFromCMD(cmd *cobra.Command) {
	if val, err := cmd.Flags().GetString("host"); err == nil {
		Creds.HostnameOrIp = val
	}
	if val, err := cmd.Flags().GetString("port"); err == nil {
		Creds.Port = val
	}
	if val, err := cmd.Flags().GetString("datacenter"); err == nil {
		Creds.Datacenter = val
	}
	if val, err := cmd.Flags().GetString("username"); err == nil {
		Creds.Username = val
	}
	if val, err := cmd.Flags().GetString("password"); err == nil {
		Creds.Password = val
	}
}

var vcenterCmd = &cobra.Command{
	Use:   "vcenter",
	Short: "target vcenter",
	Long:  "target vcenter to fetch vcenter details and manage vcenter resources",
	Run: func(cmd *cobra.Command, args []string) {
		populateCredsFromCMD(cmd)
		fmt.Println(cmd.UsageString())
	},
}

var listVMsCmd = &cobra.Command{
	Use:   "list",
	Short: "list vcenter resources",
	Long:  "list vcenter resources",
	Run: func(cmd *cobra.Command, args []string) {
		populateCredsFromCMD(cmd)
		vcenter := vcenter.Vcenter{}
		res, err := vcenter.ListVMs(context.Background(), Creds)
		for k, v := range res {
			fmt.Println(k, v)
		}
		if err != nil {
			logrus.Error(err)
		}
	},
}

var getVMCmd = &cobra.Command{
	Use:   "get",
	Short: "get vcenter resource",
	Long:  "get vcenter resource",
	Run: func(cmd *cobra.Command, args []string) {
		populateCredsFromCMD(cmd)
		var vm_name string
		if val, err := cmd.Flags().GetString("vm_name"); err == nil {
			vm_name = val
		}
		if vm_name == "" {
			logrus.Error("vm_name is required")
			fmt.Println(cmd.UsageString())
			return
		}
		vcenter := vcenter.Vcenter{}
		res, err := vcenter.GetVM(context.Background(), Creds, vm_name)
		if err != nil {
			logrus.Error(err)
		}
		fmt.Println(res)
	},
}

var cordonESXIHostCmd = &cobra.Command{
	Use:   "cordon",
	Short: "cordon esxi host",
	Long:  "cordon esxi host",
	Run: func(cmd *cobra.Command, args []string) {
		populateCredsFromCMD(cmd)
		var esxi_name string
		if val, err := cmd.Flags().GetString("esxi_name"); err == nil {
			esxi_name = val
		}
		if esxi_name == "" {
			logrus.Error("esxi_name is required")
			fmt.Println(cmd.UsageString())
			return
		}
		vcenter := vcenter.Vcenter{}
		err := vcenter.CordonHost(context.Background(), Creds, esxi_name)
		if err != nil {
			logrus.Error(err)
		}
	},
}

var unCordonESXIHostCmd = &cobra.Command{
	Use:   "uncordon",
	Short: "uncordon esxi host",
	Long:  "uncordon esxi host",
	Run: func(cmd *cobra.Command, args []string) {
		populateCredsFromCMD(cmd)
		var esxi_name string
		if val, err := cmd.Flags().GetString("esxi_name"); err == nil {
			esxi_name = val
		}
		if esxi_name == "" {
			logrus.Error("esxi_name is required")
			fmt.Println(cmd.UsageString())
			return
		}
		vcenter := vcenter.Vcenter{}
		err := vcenter.UnCordonHost(context.Background(), Creds, esxi_name)
		if err != nil {
			logrus.Error(err)
		}
	},
}

var listHostsCmd = &cobra.Command{
	Use:   "list_hosts",
	Short: "list hosts",
	Long:  "list hosts",
	Run: func(cmd *cobra.Command, args []string) {
		populateCredsFromCMD(cmd)
		vcenter := vcenter.Vcenter{}
		res, err := vcenter.ListHosts(context.Background(), Creds)
		if err != nil {
			logrus.Error(err)
		}
		tableprinter.PrintAsTable(res.Hosts)
	},
}

func init() {
	rootCmd.AddCommand(vcenterCmd)
	vcenterCmd.PersistentFlags().StringP("host", "H", "", "Set the IP-Address or hostname where vcenter is running")
	vcenterCmd.PersistentFlags().StringP("port", "p", "443", "Set the sdk port to open")
	vcenterCmd.PersistentFlags().StringP("datacenter", "d", "", "Set the datacenter to target")
	vcenterCmd.PersistentFlags().StringP("username", "U", "", "Set the username to use")
	vcenterCmd.PersistentFlags().StringP("password", "P", "", "Set the password to use")
	//getVMCMD paramter
	getVMCmd.PersistentFlags().StringP("vm_name", "v", "", "Set the vm name to use")
	//Cordon Parameter
	cordonESXIHostCmd.Flags().StringP("esxi_name", "e", "", "Set the esxi name to use")
	//UnCordon Parameter
	unCordonESXIHostCmd.Flags().StringP("esxi_name", "e", "", "Set the esxi name to use")
	//List Hosts Parameter
	listHostsCmd.Flags().StringP("datacenter", "d", "", "Set the datacenter to use")
	vcenterCmd.AddCommand(listVMsCmd, getVMCmd, cordonESXIHostCmd, unCordonESXIHostCmd, listHostsCmd)
}
