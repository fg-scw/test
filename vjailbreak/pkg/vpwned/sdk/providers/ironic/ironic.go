package ironic

import (
	"github.com/platform9/vjailbreak/pkg/vpwned/sdk/providers"
	"github.com/platform9/vjailbreak/pkg/vpwned/sdk/providers/base"
)

type IronicProvider struct {
	base.UnimplementedBaseProvider
}

func init() {
	providers.RegisterProvider("ironic", &IronicProvider{})
}
