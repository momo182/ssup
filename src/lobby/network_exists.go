package lobby

import (
	"fmt"
	"os"

	"github.com/momo182/ssup/src/entity"
)

// EnsureNetworkExists checks if a network exists
// If it doesn't exist it prints a help message and exits the program
func EnsureNetworkExists(netName string, conf *entity.Supfile, helpMenu entity.HelpDisplayer) {
	_, ok := conf.Networks.Get(netName)
	if !ok {
		helpMenu.ShowNetwork = true
		helpMenu.Show(conf)
		fmt.Printf("Network: '%s' does not exist", netName)
		os.Exit(25)
	}
}
