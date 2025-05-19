package yamlprocessor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
)

type Package struct {
	Git    string `yaml:"git"`
	Commit string `yaml:"commit"`
}

// PackageEntry can be either a string or a Package
type PackageEntry struct {
	Package Package
	String  string
	IsStr   bool
}

// UnmarshalYAML implements custom unmarshaling for PackageEntry
func (p *PackageEntry) UnmarshalYAML(value *yaml.Node) error {
	// Try to unmarshal as string first
	var s string
	if err := value.Decode(&s); err == nil {
		p.String = s
		p.IsStr = true
		return nil
	}

	// If string fails, try as Package
	var pkg Package
	if err := value.Decode(&pkg); err != nil {
		return err
	}
	p.Package = pkg
	p.IsStr = false
	return nil
}

// MarshalYAML implements custom marshaling for PackageEntry
func (p PackageEntry) MarshalYAML() (interface{}, error) {
	if p.IsStr {
		return p.String, nil
	}
	return p.Package, nil
}

type YAMLData struct {
	Packages []PackageEntry `yaml:"packages"`
}

func ProcessYAML(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var data YAMLData
	err = yaml.NewDecoder(file).Decode(&data)
	if err != nil {
		return fmt.Errorf("failed to decode YAML: %w", err)
	}

	// Define colored output
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	magenta := color.New(color.FgMagenta).SprintFunc()
	red := color.New((color.FgRed)).SprintFunc()

	var all = false

	for i := range data.Packages {
		pkg := &data.Packages[i]
		if pkg.IsStr {
			// Handle string case if needed
			fmt.Printf(magenta("> ")+"Processing package: %s\n", green(pkg.String))
			continue
		}

		if pkg.Package.Git != "" {
			fmt.Print("\n")
			fmt.Printf(magenta("> ")+"Processing package: Git: %s, Commit: %s\n",
				green(pkg.Package.Git), yellow(pkg.Package.Commit))
			latestCommit, err := getLatestCommit(pkg.Package.Git)

			if err != nil {
				fmt.Printf(magenta("! ")+"Error: %v\n", red(err))
				continue
			}

			if pkg.Package.Commit != latestCommit {
				// Confirm with user before updating
				var response string = ""
				if !all {
					fmt.Printf(magenta("? ")+"Do you want to update commit from %s to %s? (Y/n/a/other hash): ", yellow(pkg.Package.Commit), yellow(latestCommit))
					fmt.Scanln(&response)

					if response == "a" {
						all = true
						response = "y"
					}
				}
				response = strings.ToLower(strings.TrimSpace(response))
				if response == "y" || response == "" {
					fmt.Printf(magenta("# ")+"Updating commit from %s to %s\n", yellow(pkg.Package.Commit), yellow(latestCommit))
					pkg.Package.Commit = latestCommit
				} else if response == "n" {
					fmt.Printf(magenta("✓ ")+"No update needed for %s\n", green(pkg.Package.Git))
				} else {
					fmt.Printf(magenta("# ")+"Updating commit from %s to %s\n", yellow(pkg.Package.Commit), yellow(response))
					pkg.Package.Commit = response
				}
			} else {
				fmt.Printf(magenta("✓ ")+"No update needed for %s\n", green(pkg.Package.Git))
			}
		}
	}

	fmt.Printf(magenta("> ")+"Writing results to %s\n", green(filePath))
	return writeResult(data, filePath)
}

func writeResult(data YAMLData, filePath string) error {
	originalContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read original YAML file: %w", err)
	}

	var originalNodes yaml.Node
	err = yaml.Unmarshal(originalContent, &originalNodes)
	if err != nil {
		return fmt.Errorf("failed to unmarshal original YAML: %w", err)
	}

	err = updateContent(&originalNodes, data)
	if err != nil {
		return err
	}

	output, err := yaml.Marshal(&originalNodes)
	if err != nil {
		return fmt.Errorf("failed to marshal updated YAML: %w", err)
	}
	err = os.WriteFile(filePath, output, 0644)
	if err != nil {
		return fmt.Errorf("failed to write updated YAML: %w", err)
	}

	return nil
}

func updateContent(originalNodes *yaml.Node, data YAMLData) error {
	mapping := originalNodes.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return fmt.Errorf("expected a mapping node, got %v", mapping.Kind)
	}

	for i := 0; i < len(mapping.Content); i += 2 {
		keyNode := mapping.Content[i]
		if keyNode.Value == "packages" {
			valueNode := mapping.Content[i+1]
			if valueNode.Kind == yaml.SequenceNode {
				valueNode.Content = nil // Clear the existing content
				for _, pkg := range data.Packages {
					pkgNode := &yaml.Node{
						Kind: yaml.MappingNode,
						Content: []*yaml.Node{
							{Kind: yaml.ScalarNode, Value: "git"},
							{Kind: yaml.ScalarNode, Value: pkg.Package.Git},
							{Kind: yaml.ScalarNode, Value: "commit"},
							{Kind: yaml.ScalarNode, Value: pkg.Package.Commit},
						},
					}
					valueNode.Content = append(valueNode.Content, pkgNode)
				}
			} else {
				return fmt.Errorf("expected a sequence node for 'packages', got %v", valueNode.Kind)
			}
		}
	}
	return nil
}

func getLatestCommit(gitURL string) (string, error) {
	command := fmt.Sprintf("git ls-remote %s HEAD", gitURL)
	latestCommit, err := exec.Command("bash", "-c", command).Output()
	if err != nil {
		return "", fmt.Errorf("error getting latest commit for %s: %w", gitURL, err)
	}
	return strings.Fields(string(latestCommit))[0], nil
}
