package network

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
)

type LoopbackAddress struct {
	Interface string `json:"interface"`
	IP        string `json:"ip"`
	Netmask   string `json:"netmask,omitempty"`
}

// GetHostLoopbackAddresses returns all configured loopback addresses on the host
func GetHostLoopbackAddresses() ([]LoopbackAddress, error) {
	switch runtime.GOOS {
	case "darwin":
		return getLoopbackAddressesDarwin()
	case "linux":
		return getLoopbackAddressesLinux()
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func getLoopbackAddressesDarwin() ([]LoopbackAddress, error) {
	cmd := exec.Command("ifconfig", "lo0")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run ifconfig: %w", err)
	}

	var addresses []LoopbackAddress
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "inet ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				ip := parts[1]
				// Include all 127.x.x.x addresses except 127.0.0.1
				if ip != "127.0.0.1" && strings.HasPrefix(ip, "127.") {
					addr := LoopbackAddress{
						Interface: "lo0",
						IP:        ip,
					}
					if len(parts) >= 4 && parts[2] == "netmask" {
						addr.Netmask = parts[3]
					}
					addresses = append(addresses, addr)
				}
			}
		}
	}
	
	return addresses, nil
}

func getLoopbackAddressesLinux() ([]LoopbackAddress, error) {
	cmd := exec.Command("ip", "addr", "show", "lo")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run ip command: %w", err)
	}

	var addresses []LoopbackAddress
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "inet ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				ipWithMask := parts[1]
				ip := strings.Split(ipWithMask, "/")[0]
				// Include all 127.x.x.x addresses except 127.0.0.1
				if ip != "127.0.0.1" && strings.HasPrefix(ip, "127.") {
					addresses = append(addresses, LoopbackAddress{
						Interface: "lo",
						IP:        ip,
						Netmask:   ipWithMask,
					})
				}
			}
		}
	}
	
	return addresses, nil
}

// IsValidLoopbackIP checks if an IP is a valid loopback address
func IsValidLoopbackIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	return strings.HasPrefix(ip, "127.") && ip != "127.0.0.1"
}

// IsLoopbackConfigured checks if a specific loopback IP is configured on the host
func IsLoopbackConfigured(ip string) (bool, error) {
	addresses, err := GetHostLoopbackAddresses()
	if err != nil {
		return false, err
	}
	
	for _, addr := range addresses {
		if addr.IP == ip {
			return true, nil
		}
	}
	
	return false, nil
}

// GenerateNmcliCommand generates an nmcli command to add a loopback address
func GenerateNmcliCommand(ip string) string {
	return fmt.Sprintf("sudo nmcli connection modify lo +ipv4.addresses %s/32", ip)
}

// GenerateNmcliCommands generates nmcli commands for multiple addresses
func GenerateNmcliCommands(ips []string) []string {
	var commands []string
	for _, ip := range ips {
		commands = append(commands, GenerateNmcliCommand(ip))
	}
	// Add the command to apply changes
	if len(commands) > 0 {
		commands = append(commands, "sudo nmcli connection up lo")
	}
	return commands
}