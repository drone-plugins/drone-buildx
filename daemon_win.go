//go:build windows
// +build windows

package docker

const dockerExe = "C:\\Windows\\system32\\docker.exe"
const dockerdExe = ""
const dockerHome = "C:\\ProgramData\\docker\\"

func (p Plugin) startDaemon() {
	// this is a no-op on windows
}
