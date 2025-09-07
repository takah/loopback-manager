package manager

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/takah/loopback-manager/internal/config"
	"github.com/takah/loopback-manager/internal/network"
)

type Manager struct {
	config      *config.Config
	assignments map[string]string
	dataFile    string
}

type Repository struct {
	Org  string `json:"org"`
	Name string `json:"name"`
	IP   string `json:"ip,omitempty"`
}

func New(cfg *config.Config) *Manager {
	homeDir, _ := os.UserHomeDir()
	dataFile := filepath.Join(homeDir, ".config", "loopback-manager", "assignments.txt")
	
	m := &Manager{
		config:      cfg,
		assignments: make(map[string]string),
		dataFile:    dataFile,
	}
	
	m.loadAssignments()
	return m
}

func (m *Manager) loadAssignments() error {
	os.MkdirAll(filepath.Dir(m.dataFile), 0755)
	
	data, err := ioutil.ReadFile(m.dataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		parts := strings.Fields(line)
		if len(parts) == 3 {
			key := fmt.Sprintf("%s/%s", parts[0], parts[1])
			m.assignments[key] = parts[2]
		}
	}
	
	return nil
}

func (m *Manager) saveAssignments() error {
	os.MkdirAll(filepath.Dir(m.dataFile), 0755)
	
	var lines []string
	for key, ip := range m.assignments {
		parts := strings.Split(key, "/")
		if len(parts) == 2 {
			lines = append(lines, fmt.Sprintf("%s %s %s", parts[0], parts[1], ip))
		}
	}
	
	sort.Strings(lines)
	data := strings.Join(lines, "\n")
	
	return ioutil.WriteFile(m.dataFile, []byte(data), 0644)
}

func (m *Manager) List(jsonOutput bool) error {
	repos := m.getAllRepositories()
	
	if jsonOutput {
		output, err := json.MarshalIndent(repos, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(output))
		return nil
	}
	
	if len(repos) == 0 {
		fmt.Println("No repositories found.")
		return nil
	}
	
	fmt.Printf("%-30s %-15s %s\n", "Repository", "IP Address", "Status")
	fmt.Println(strings.Repeat("-", 60))
	
	for _, repo := range repos {
		status := "✓ Assigned"
		ipDisplay := repo.IP
		if repo.IP == "" {
			status = "✗ Not assigned"
			ipDisplay = "-"
		}
		fmt.Printf("%-30s %-15s %s\n", fmt.Sprintf("%s/%s", repo.Org, repo.Name), ipDisplay, status)
	}
	
	return nil
}

func (m *Manager) Scan(jsonOutput bool) error {
	unassigned := m.getUnassignedRepositories()
	
	if jsonOutput {
		output, err := json.MarshalIndent(unassigned, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(output))
		return nil
	}
	
	if len(unassigned) == 0 {
		fmt.Println("All repositories have IP assignments.")
		return nil
	}
	
	fmt.Printf("Found %d unassigned repositories:\n\n", len(unassigned))
	for _, repo := range unassigned {
		fmt.Printf("  - %s/%s\n", repo.Org, repo.Name)
	}
	
	fmt.Println("\nRun 'loopback-manager auto-assign' to assign IPs automatically.")
	
	return nil
}

func (m *Manager) Assign(org, repo, ip string) error {
	key := fmt.Sprintf("%s/%s", org, repo)
	
	if ip == "" {
		ip = m.getNextAvailableIP()
	}
	
	if !m.isValidIP(ip) {
		return fmt.Errorf("invalid IP address: %s", ip)
	}
	
	if existing := m.findRepositoryByIP(ip); existing != "" && existing != key {
		return fmt.Errorf("IP %s is already assigned to %s", ip, existing)
	}
	
	m.assignments[key] = ip
	
	if err := m.saveAssignments(); err != nil {
		return err
	}
	
	repoPath := filepath.Join(m.config.BaseDir, org, repo)
	if err := m.updateEnvFile(repoPath, ip); err != nil {
		fmt.Printf("Warning: Could not update .env file: %v\n", err)
	}
	
	fmt.Printf("Assigned %s to %s/%s\n", ip, org, repo)
	return nil
}

func (m *Manager) Remove(org, repo string) error {
	key := fmt.Sprintf("%s/%s", org, repo)
	
	if _, exists := m.assignments[key]; !exists {
		return fmt.Errorf("no IP assignment found for %s/%s", org, repo)
	}
	
	delete(m.assignments, key)
	
	if err := m.saveAssignments(); err != nil {
		return err
	}
	
	fmt.Printf("Removed IP assignment for %s/%s\n", org, repo)
	return nil
}

func (m *Manager) AutoAssign(execute bool) error {
	unassigned := m.getUnassignedRepositories()
	
	if len(unassigned) == 0 {
		fmt.Println("All repositories already have IP assignments.")
		return nil
	}
	
	if !execute {
		fmt.Println("DRY RUN MODE - No changes will be made")
		fmt.Println("To execute, run with --execute flag")
		fmt.Println()
	}
	
	fmt.Printf("Found %d unassigned repositories:\n\n", len(unassigned))
	
	usedIPs := make(map[string]bool)
	for _, ip := range m.assignments {
		usedIPs[ip] = true
	}
	
	nextIP := m.config.IPRange.Start
	for _, repo := range unassigned {
		// Find next available IP
		var ip string
		for i := nextIP; i <= m.config.IPRange.End; i++ {
			candidateIP := fmt.Sprintf("%s.%d", m.config.IPRange.Base, i)
			if !usedIPs[candidateIP] {
				ip = candidateIP
				nextIP = i + 1
				break
			}
		}
		
		if ip == "" {
			return fmt.Errorf("no more available IPs in range")
		}
		
		if execute {
			if err := m.Assign(repo.Org, repo.Name, ip); err != nil {
				return fmt.Errorf("failed to assign IP to %s/%s: %v", repo.Org, repo.Name, err)
			}
		} else {
			fmt.Printf("  Would assign %s to %s/%s\n", ip, repo.Org, repo.Name)
			usedIPs[ip] = true // Mark as used for dry run calculation
		}
	}
	
	if execute {
		fmt.Println("\nAll repositories have been assigned IPs.")
	} else {
		fmt.Printf("\nDRY RUN COMPLETE - Would assign %d IPs\n", len(unassigned))
		fmt.Println("To execute these assignments, run: loopback-manager auto-assign --execute")
	}
	return nil
}

func (m *Manager) CheckDuplicates() error {
	ipMap := make(map[string][]string)
	
	for key, ip := range m.assignments {
		ipMap[ip] = append(ipMap[ip], key)
	}
	
	duplicates := false
	for ip, repos := range ipMap {
		if len(repos) > 1 {
			duplicates = true
			fmt.Printf("Duplicate IP %s assigned to:\n", ip)
			for _, repo := range repos {
				fmt.Printf("  - %s\n", repo)
			}
		}
	}
	
	if !duplicates {
		fmt.Println("No duplicate IPs found.")
	}
	
	return nil
}

func (m *Manager) getAllRepositories() []Repository {
	var repos []Repository
	
	orgs, _ := ioutil.ReadDir(m.config.BaseDir)
	for _, org := range orgs {
		if !org.IsDir() || strings.HasPrefix(org.Name(), ".") {
			continue
		}
		
		orgPath := filepath.Join(m.config.BaseDir, org.Name())
		repoItems, _ := ioutil.ReadDir(orgPath)
		
		for _, repoItem := range repoItems {
			if !repoItem.IsDir() || strings.HasPrefix(repoItem.Name(), ".") {
				continue
			}
			
			repoPath := filepath.Join(orgPath, repoItem.Name())
			if m.hasDockerCompose(repoPath) {
				key := fmt.Sprintf("%s/%s", org.Name(), repoItem.Name())
				repos = append(repos, Repository{
					Org:  org.Name(),
					Name: repoItem.Name(),
					IP:   m.assignments[key],
				})
			}
		}
	}
	
	sort.Slice(repos, func(i, j int) bool {
		return fmt.Sprintf("%s/%s", repos[i].Org, repos[i].Name) < 
		       fmt.Sprintf("%s/%s", repos[j].Org, repos[j].Name)
	})
	
	return repos
}

func (m *Manager) getUnassignedRepositories() []Repository {
	var unassigned []Repository
	
	for _, repo := range m.getAllRepositories() {
		if repo.IP == "" {
			unassigned = append(unassigned, repo)
		}
	}
	
	return unassigned
}

func (m *Manager) hasDockerCompose(path string) bool {
	composeFiles := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}
	
	for _, file := range composeFiles {
		if _, err := os.Stat(filepath.Join(path, file)); err == nil {
			return true
		}
	}
	
	return false
}

func (m *Manager) getNextAvailableIP() string {
	usedIPs := make(map[string]bool)
	for _, ip := range m.assignments {
		usedIPs[ip] = true
	}
	
	for i := m.config.IPRange.Start; i <= m.config.IPRange.End; i++ {
		ip := fmt.Sprintf("%s.%d", m.config.IPRange.Base, i)
		if !usedIPs[ip] {
			return ip
		}
	}
	
	return ""
}

func (m *Manager) isValidIP(ip string) bool {
	if !strings.HasPrefix(ip, m.config.IPRange.Base+".") {
		return false
	}
	
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	
	return true
}

func (m *Manager) findRepositoryByIP(ip string) string {
	for key, assignedIP := range m.assignments {
		if assignedIP == ip {
			return key
		}
	}
	return ""
}

func (m *Manager) updateEnvFile(repoPath, ip string) error {
	envFile := filepath.Join(repoPath, ".env")
	
	content := fmt.Sprintf("LOOPBACK_IP=%s\n", ip)
	
	data, err := ioutil.ReadFile(envFile)
	if err == nil {
		lines := strings.Split(string(data), "\n")
		var newLines []string
		found := false
		
		for _, line := range lines {
			if strings.HasPrefix(line, "LOOPBACK_IP=") {
				newLines = append(newLines, fmt.Sprintf("LOOPBACK_IP=%s", ip))
				found = true
			} else if line != "" || len(newLines) > 0 {
				newLines = append(newLines, line)
			}
		}
		
		if !found {
			newLines = append([]string{fmt.Sprintf("LOOPBACK_IP=%s", ip)}, newLines...)
		}
		
		content = strings.Join(newLines, "\n")
	}
	
	return ioutil.WriteFile(envFile, []byte(content), 0644)
}

// ListHostLoopback lists all configured loopback addresses on the host
func (m *Manager) ListHostLoopback(jsonOutput bool) error {
	addresses, err := network.GetHostLoopbackAddresses()
	if err != nil {
		return fmt.Errorf("failed to get host loopback addresses: %w", err)
	}

	if jsonOutput {
		output, err := json.MarshalIndent(addresses, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(output))
		return nil
	}

	if len(addresses) == 0 {
		fmt.Println("No additional loopback addresses configured on host.")
		fmt.Println("(127.0.0.1 is excluded as it's the default loopback)")
		return nil
	}

	fmt.Printf("Host Loopback Addresses:\n")
	fmt.Printf("%-20s %s\n", "Interface", "IP Address")
	fmt.Println(strings.Repeat("-", 40))
	
	for _, addr := range addresses {
		fmt.Printf("%-20s %s\n", addr.Interface, addr.IP)
	}
	
	return nil
}

// SyncCheck checks consistency between assignments and host configuration
func (m *Manager) SyncCheck() error {
	// Get host loopback addresses
	hostAddresses, err := network.GetHostLoopbackAddresses()
	if err != nil {
		return fmt.Errorf("failed to get host loopback addresses: %w", err)
	}

	// Create a map of configured addresses
	hostIPMap := make(map[string]bool)
	for _, addr := range hostAddresses {
		hostIPMap[addr.IP] = true
	}

	// Check which assigned IPs are not configured on host
	var missingIPs []string
	assignedIPs := make(map[string]string) // IP -> repo mapping
	
	for repo, ip := range m.assignments {
		assignedIPs[ip] = repo
		if !hostIPMap[ip] {
			missingIPs = append(missingIPs, ip)
		}
	}

	// Sort for consistent output
	sort.Strings(missingIPs)

	// Report status
	fmt.Println("=== Loopback Address Consistency Check ===\n")
	
	fmt.Printf("Assigned addresses in config: %d\n", len(m.assignments))
	fmt.Printf("Loopback addresses on host:   %d\n", len(hostAddresses))
	fmt.Println()

	if len(missingIPs) == 0 {
		fmt.Println("✓ All assigned IP addresses are configured on the host.")
		return nil
	}

	fmt.Printf("⚠ Found %d assigned IP addresses not configured on host:\n\n", len(missingIPs))
	
	for _, ip := range missingIPs {
		repo := assignedIPs[ip]
		fmt.Printf("  %s (assigned to %s)\n", ip, repo)
	}

	fmt.Println("\n=== Configuration Commands ===\n")
	fmt.Println("To add these loopback addresses to your host:")
	fmt.Println()
	
	// Show NetworkManager commands
	fmt.Println("Using NetworkManager (if available):")
	nmcliCommands := network.GenerateNmcliCommands(missingIPs)
	for _, cmd := range nmcliCommands {
		fmt.Printf("  %s\n", cmd)
	}
	
	fmt.Println("\nAlternatively, using ip command directly:")
	for _, ip := range missingIPs {
		fmt.Printf("  sudo ip addr add %s/8 dev lo\n", ip)
	}
	
	fmt.Println("\nNote: These changes may not persist after reboot without proper configuration.")
	
	return nil
}