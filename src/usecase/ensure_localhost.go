package usecase

import (
	"github.com/clok/kemba"
	"github.com/gookit/goutil/strutil"
	"github.com/momo182/ssup/src/entity"
)

func EnsureLocalhost(supFile *entity.Supfile) {
	l := kemba.New("usecase::ensure_localhost").Printf
	var gotLocal bool = false
	for _, net := range supFile.Networks.Nets {
		for _, host := range net.Hosts {
			if strutil.HasOneSub(host.Host, []string{"localhost", "127.0.0.1"}) {
				gotLocal = true
				break
			}
		}
	}
	l("got localhost defined: %v", gotLocal)

	if !gotLocal {
		l("adding localhost network")
		// create localhost network
		supFile.Networks.Set("localhost", "localhost")
	}
}
