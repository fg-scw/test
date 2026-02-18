package main

import (
	"fmt"
	"os"

	"github.com/platform9/vjailbreak/pkg/vpwned/cli"
)

/*
This will be the command syntax that we will try to in-corporate
vpwctl <provider/target> <command> <arguments>
when providing Target certain read/update operations are possible
For Eg:
vpwctl vcenter list esxi
vpwctl vcenter list datacenter
vpwctl vcenter update-esxi <esxi_name> <maintainence_mode_enter>
vpwctl vcenter update-esxi <esxi_name> <maintainence_mode_exit>
vpwctl vcenter find <vm_name> <optional: datacenter>

When providing the bmPoviders the following should be possible:
vpwctl maas list machine <optional: filter_by_state>
vpwctl maas set machine boot pxe/hdd/cdrom
vpwctl maas set machine power off/on/cycle/reset
vpwctl maas reclaim machine <machine_name> <os> <kernel_version>
*/
func main() {
	if err := cli.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
