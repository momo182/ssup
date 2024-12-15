package usecase_test

import (
	"os"
	"testing"

	"github.com/clok/kemba"
	"github.com/momo182/ssup/src/entity"
	"github.com/momo182/ssup/src/usecase"
)

var debug = false
var makefile_input = "/Users/k.pechenenko/git/ssup/test/Supfile_makefile_mode.yml"
var normal_mode_input = "/Users/k.pechenenko/git/ssup/test/Supfile_normal_mode.yml"
var target_mode_input = "/Users/k.pechenenko/git/ssup/test/Supfile_normal_mode_target_no_affix.yml"
var target_mode_affixed = "/Users/k.pechenenko/git/ssup/test/Supfile_normal_mode_target_affixed.yml"

// (*entity.PlayBook)(0x14000198d68)({
//  plays: ([]entity.Play) (len=1 cap=1) {
//   (entity.Play) {
//    Nets: (*entity.Network)(0x140002c5b90)({
//     Env: (entity.EnvList) {
//      store: (map[string]string) <nil>
//     },
//     Inventory: (string) "",
//     Hosts: ([]entity.NetworkHost) (len=1 cap=1) {
//      (entity.NetworkHost) {
//       Host: (string) (len=9) "localhost",
//       User: (string) "",
//       Password: (string) "",
//       Tube: (string) "",
//       Env: (entity.EnvList) {
//        store: (map[string]string) <nil>
//       },
//       Sudo: (bool) false
//      }
//     },
//     Bastion: (string) "",
//     User: (string) "",
//     Password: (string) "",
//     IdentityFile: (string) ""
//    }),
//    Commands: ([]*entity.Command) (len=1 cap=1) {
//     (*entity.Command)(0x14000263760)({
//      Name: (string) (len=4) "cmd1",
//      Desc: (string) (len=17) "cmd 1 description",
//      Local: (string) (len=7) "exit 0\n",
//      Run: (string) (len=1) "\n",
//      Script: (string) "",
//      Upload: ([]*entity.Upload) <nil>,
//      Copy: (*entity.CopyOrder)(<nil>),
//      Stdin: (bool) false,
//      Once: (bool) false,
//      Serial: (int) 0,
//      Fetch: (*entity.FetchOrder)(<nil>),
//      Sudo: (bool) false,
//      SudoPass: (string) "",
//      Env: (entity.EnvList) {
//       store: (map[string]string) <nil>
//      },
//      RunOnce: (bool) false
//     })
//    }
//   }
//  }
// })

func TestOneCommandNoNetworks(t *testing.T) {
	args := []string{"cmd1"}

	initialArgs := &entity.InitialArgs{}
	initialArgs.DisableColor = true
	initialArgs.Supfile = makefile_input
	initialArgs.CommandArgs = args

	if debug {
		os.Setenv("DEBUG", "*")
	}

	conf := usecase.ReadSupfile(initialArgs)
	initData := entity.InitState{
		InitialArgs: initialArgs,
		Conf:        conf,
	}

	playbook, err := usecase.ParseInitialArgs(initData)
	if err != nil {
		t.Fatal(err)
	}

	// spew.Dump(playbook)

	// check we had one play
	expectedPlays := 1
	foundPlays := len(playbook.GetPlays())
	currPlay := playbook.GetPlays()[0]
	if foundPlays != expectedPlays {
		t.Errorf("Expected %d playbooks, found %d", expectedPlays, foundPlays)
	}
	// network must be localhost
	hostname := currPlay.Nets.Hosts[0].Host
	if hostname != "localhost" {
		t.Errorf("Expected %s playbooks, found %s", "localhost", hostname)
	}

	// check if local was added at all
	hosts := currPlay.Nets.Hosts
	foundLocal := false
	for _, host := range hosts {
		if host.Host == "localhost" {
			foundLocal = true
		}
	}

	if !foundLocal {
		t.Errorf("localhost was not found")
	}

	// get current network name
	env := currPlay.Nets.Env
	network := env.Get("SUP_NETWORK")
	if network != "localhost" {
		t.Errorf("expected local network to be current net, got: %s", network)
	}
	// currPlay.Nets.Hosts
	// currPlay.Commands
}

func TestTwoCommandsNoNetworks(t *testing.T) {
	args := []string{"cmd1", "cmd2"}
	initialArgs := &entity.InitialArgs{}
	initialArgs.DisableColor = true
	initialArgs.Supfile = makefile_input
	initialArgs.CommandArgs = args
	if debug {
		os.Setenv("DEBUG", "*")
	}

	conf := usecase.ReadSupfile(initialArgs)
	initData := entity.InitState{
		InitialArgs: initialArgs,
		Conf:        conf,
	}

	playbook, err := usecase.ParseInitialArgs(initData)
	if err != nil {
		t.Fatal(err)
	}
	expectedPlays := 2
	foundPlays := len(playbook.GetPlays())
	for _, currPlay := range playbook.GetPlays() {
		// spew.Dump(playbook)
		if foundPlays != expectedPlays {
			t.Errorf("Expected %d playbooks, found %d", expectedPlays, foundPlays)
		}
		// network must be localhost
		hostname := currPlay.Nets.Hosts[0].Host
		if hostname != "localhost" {
			t.Errorf("Expected %s playbooks, found %s", "localhost", hostname)
		}

		// check if local was added at all
		hosts := currPlay.Nets.Hosts
		foundLocal := false
		for _, host := range hosts {
			if host.Host == "localhost" {
				foundLocal = true
			}
		}

		if !foundLocal {
			t.Errorf("localhost was not found")
		}

		// get current network name
		env := currPlay.Nets.Env
		network := env.Get("SUP_NETWORK")
		if network != "localhost" {
			t.Errorf("expected local network to be current net, got: %s", network)
		}
	}
}

func TestTwoCommandsAndNetwork(t *testing.T) {
	args := []string{"remote1", "cmd1", "cmd2"}
	initialArgs := &entity.InitialArgs{}
	initialArgs.DisableColor = true
	initialArgs.Supfile = normal_mode_input
	initialArgs.CommandArgs = args
	if debug {
		os.Setenv("DEBUG", "*")
	}

	conf := usecase.ReadSupfile(initialArgs)
	initData := entity.InitState{
		InitialArgs: initialArgs,
		Conf:        conf,
	}

	playbook, err := usecase.ParseInitialArgs(initData)
	if err != nil {
		t.Fatal(err)
	}
	expectedPlays := 1
	foundPlays := len(playbook.GetPlays())
	// spew.Dump(playbook)
	if foundPlays != expectedPlays {
		t.Errorf("Expected %d playbooks, found %d", expectedPlays, foundPlays)
	}

	currPlay := playbook.GetPlays()[0]
	hostname := currPlay.Nets.Hosts[0].Host
	if hostname != "foo@1.2.3.4" {
		t.Errorf("Expected %s network name, found %s", "localhost", hostname)
	}

	// get current network name
	env := currPlay.Nets.Env
	network := env.Get("SUP_NETWORK")
	if network != "remote1" {
		t.Errorf("expected local network to be current net, got: %s", network)
	}
}

func TestTwoCmdsAndNet2(t *testing.T) {
	args := []string{"remote2", "cmd1", "cmd2"}
	initialArgs := &entity.InitialArgs{}
	initialArgs.DisableColor = true
	initialArgs.Supfile = normal_mode_input
	initialArgs.CommandArgs = args
	if debug {
		os.Setenv("DEBUG", "*")
	}

	conf := usecase.ReadSupfile(initialArgs)
	initData := entity.InitState{
		InitialArgs: initialArgs,
		Conf:        conf,
	}

	playbook, err := usecase.ParseInitialArgs(initData)
	if err != nil {
		t.Fatal(err)
	}
	expectedPlays := 1
	foundPlays := len(playbook.GetPlays())
	// spew.Dump(playbook)
	if foundPlays != expectedPlays {
		t.Errorf("Expected %d playbooks, found %d", expectedPlays, foundPlays)
	}
}

func TestTargetAndNoAffix(t *testing.T) {
	args := []string{"remote1", "target1"}
	initialArgs := &entity.InitialArgs{}
	initialArgs.DisableColor = true
	initialArgs.Supfile = target_mode_input
	initialArgs.CommandArgs = args
	if debug {
		os.Setenv("DEBUG", "*")
	}

	conf := usecase.ReadSupfile(initialArgs)
	initData := entity.InitState{
		InitialArgs: initialArgs,
		Conf:        conf,
	}

	playbook, err := usecase.ParseInitialArgs(initData)
	if err != nil {
		t.Fatal(err)
	}
	expectedPlays := 1
	foundPlays := len(playbook.GetPlays())
	// spew.Dump(playbook)
	if foundPlays != expectedPlays {
		t.Errorf("Expected %d playbooks, found %d", expectedPlays, foundPlays)
	}
}

func TestTargetAndAffix(t *testing.T) {
	l := kemba.New("test").Printf
	args := []string{"target1"}
	initialArgs := &entity.InitialArgs{}
	initialArgs.DisableColor = true
	initialArgs.Supfile = target_mode_affixed
	initialArgs.CommandArgs = args
	if debug {
		os.Setenv("DEBUG", "*")
	}

	conf := usecase.ReadSupfile(initialArgs)
	initData := entity.InitState{
		InitialArgs: initialArgs,
		Conf:        conf,
	}

	playbook, err := usecase.ParseInitialArgs(initData)
	if err != nil {
		t.Fatal(err)
	}
	expectedPlays := 2
	foundPlays := len(playbook.GetPlays())
	// spew.Dump(playbook)
	l("Expected %d playbooks, found %d", expectedPlays, foundPlays)

	if foundPlays != expectedPlays {
		t.Errorf("Expected %d playbooks, found %d", expectedPlays, foundPlays)
	}
}
