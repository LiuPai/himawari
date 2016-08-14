package desktop

import (
	"os"
	"runtime"
	"strings"
	"os/exec"
	"syscall"
	"fmt"
)

// SetBackground change desktop background to image on path
func SetBackground(path string) bool {
	env := Environment()
	switch env {
	case "gnome", "unity", "cinnamon", "pantheon", "gnome-classic":
		if env == "unity" {
			_ = exec.Command("gesettings", "set",
				"org.gnome.desktop.background",
				"draw-background", "false").Run()
		}
		_ = exec.Command("gsettings", "set",
			"org.gnome.desktop.background",
			"picture-uri", "file://" + path).Run()
		_ = exec.Command("gsettings", "set",
			"org.gnome.desktop.background",
			"picture-options", "scaled").Run()
		_ = exec.Command("gsettings", "set",
			"org.gnome.desktop.background",
			"primary-color", "FFFFFF").Run()
	case "mate":
		_ = exec.Command("gsettings", "set",
			"org.mate.background",
			"picture-filename", path).Run()
	case "i3":
		_ = exec.Command("feh", "--bg-fill", path).Run()
	case "xfce4":
		// for display in xfce_displays {
		// 	_ = exec.Command("xfconf-query", "--channel",
		// 		"xfce4-desktop", "--property", display,
		// 		"--set", path).Run()
		// }
	case "lxde":
		_ = exec.Command("pcmanfm", "--wallpaper-mode=fit",
			"--set-wallpaper", path).Run()
	case "mac":
		_ = exec.Command("osascript", "-e", `tell application "System Events"
set theDesktops to a reference to every desktop
repeat with aDesktop in theDesktops
set the picture of aDesktop to` + path + `
end repeat
end tell`).Run()
		_ = exec.Command("killall", "dock").Run()
	case "windows":
		_ = exec.Command("REG", "ADD", "HKCU\\Control Panel\\Desktop",
			"/V", "Wallpaper", "/T", "REG_SZ", "/F",
			"/D", path).Run()
		_ = exec.Command("REG", "ADD", "HKCU\\Control Panel\\Desktop",
			"/V", "WallpaperStyle", "/T", "REG_SZ", "/F",
			"/D", "2").Run()
		_ = exec.Command("REG", "ADD", "HKCU\\Control Panel\\Desktop",
                        "/V", "TileWallpaper", "/T", "REG_SZ", "/F",
                        "/D", "0").Run()
                _ = exec.Command("RUNDLL32.EXE", "user32.dll",
			"UpdatePerUserSystemParameters").Run()
	default:
		switch {
		case hasProgram("feh"):
			_ = os.Setenv("DISPLAY", ":0")
			_ = exec.Command("feh", "--bg-max", path)
		case hasProgram("nitrogn"):
			_ = os.Setenv("DISPLAY", ":0")
			_ = exec.Command("nitrogen", "--restore")
		default:
			return false
		}
        }
	return true
}

// Environment get system desktop environment
func Environment() string {
	// http://stackoverflow.com/a/21213358/4466589
	// From http://stackoverflow.com/questions/2035657/what-is-my-current-desktop-environment
        // and http://ubuntuforums.org/showthread.php?t=652320
        // and http://ubuntuforums.org/showthread.php?t=652320
        // and http://ubuntuforums.org/showthread.php?t=1139057
        // check operation system
	switch runtime.GOOS {
	case "freebsd", "linux", "netbsd", "opendsd", "solaris", "dargonfly":
		session := strings.ToLower(os.Getenv("DESKTOP_SESSION"))
		switch session {
			case "gnome", "unity", "cinnamon", "mate", "xfce4",
			"lxde", "fluxbox", "blackbox", "openbox", "icewm",
			"jwm", "afterstep", "trinity", "kde", "pantheon",
			"gnome-classic", "i3":
			return session
                case "":
			gnomeSessionID := os.Getenv("GNOME_DESKTOP_SESSION_ID")
			switch {
			case os.Getenv("KDE_FULL_SESSION") == "true":
				return "kde"
				case gnomeSessionID != "" &&
					!strings.Contains(gnomeSessionID, "deprecated"):
				return "gnome2"
			case isRunning("xfce-mcs-manage"):
				return "xfce4"
			case isRunning("ksmserver"):
				return "kde"
			default:
				goto lastTry
			}
		default:
			switch {
				case strings.Contains(session, "xfce") ||
					strings.HasPrefix(session, "xubuntu"):
				return "xfce4"
			case strings.HasPrefix(session, "ubuntu"):
				return "unity"
			case strings.HasPrefix(session, "lubuntu"):
                                return "lxde"
			case strings.HasPrefix(session, "kubuntu"):
                                return "kde"
			case strings.HasPrefix(session, "razor"):
                                return "razor-qt"
			case strings.HasPrefix(session, "wmaker"):
                                return "windowmaker"
			case strings.HasPrefix(session, "ubuntu"):
                                return "unity"
			default:
				goto lastTry
			}
                }
        case "darwin":
                return "mac"
        default: // android nacl windows nacl plan9
		return runtime.GOOS
	}
lastTry:
	currentDesktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	switch currentDesktop {
	case "gnome", "unity", "kde", "gnome-classic", "mate":
		return currentDesktop
	case "xfce":
		return "xfce4"
	case "x-cinnamon":
		return "cinnamon"
	default:
		return "unknown"
	}
}

func isRunning(process string) bool {
	cmd := exec.Command("pidof", process)
	if err := cmd.Start(); err != nil {
		return false
	}
	if err := getCMDExit(cmd); err != nil {
		return false
	}
	return true
}

func getCMDExit(cmd *exec.Cmd) error {
	if err := cmd.Wait(); err != nil {
                if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				return fmt.Errorf("%s exit: %d",
					cmd.Path,
					status.ExitStatus())
			}
		}
		return err
        }
	return nil
}

func hasProgram(program string) bool {
	if _, err := exec.LookPath(program); err != nil {
		return false
	}
	return true
}
