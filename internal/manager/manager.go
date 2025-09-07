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