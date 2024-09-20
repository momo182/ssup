package usecase

import (
	"github.com/gookit/goutil/strutil"
	"github.com/momo182/ssup/src/entity"
)

func EnsureLocalhost(supFile *entity.Supfile) {
	var gotLocal bool = false
	for _, net := range supFile.Networks.Nets {
		for _, host := range net.Hosts {
			if strutil.HasOneSub(host.Host, []string{"localhost", "127.0.0.1"}) {
				gotLocal = true
				break
			}
		}
	}

	if !gotLocal {
		// create localhost network
		supFile.Networks.Set("localhost", "localhost")
	}
}
