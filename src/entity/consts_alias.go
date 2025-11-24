package entity

import consts "github.com/momo182/ssup/src/entity/consts"

// Re-export constants from the consts package to maintain compatibility
const (
	SSUP_WORK_FOLDER           = consts.SSUP_WORK_FOLDER
	MAIN_SCRIPT                = consts.MAIN_SCRIPT
	VARS_FILE                  = consts.VARS_FILE
	HASHED_PASS                = consts.HASHED_PASS
	INJECTED_COMMANDS_FILE     = consts.INJECTED_COMMANDS_FILE
	VERSION                    = consts.VERSION
	PassSeparator             = consts.PassSeparator
	TubeNameSeparator         = consts.TubeNameSeparator
	CsupPasswdEnv             = consts.CsupPasswdEnv
	CsupDoSudoEnv             = consts.CsupDoSudoEnv
)