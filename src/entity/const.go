package entity

import "os"

const CsupPasswdEnv = "SUP_PASSWORD"
const CsupDoSudoEnv = "SUP_SUDO"

const PassSeparator = " | "
const TubeNameSeparator = " << "
const SPEW_DEPTH = 1
const MAIN_SCRIPT = "_ssup_run"
const VARS_FILE = "_ssup_env"
const HASHED_PASS = "_ssup_pass"
const INJECTED_COMMANDS_FILE = "_ssup_commands"
const SSUP_WORK_FOLDER = ".local" + string(os.PathSeparator) + "ssup" + string(os.PathSeparator) + "run" + string(os.PathSeparator)
const VERSION = "0.5"
const SourceDirective = "#source://"
