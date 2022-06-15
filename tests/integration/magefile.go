//go:build mage

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/target"
)

var (
	id_rsa                = filepath.Join(".", "id_rsa")
	id_rsa_pub            = filepath.Join(".", "id_rsa.pub")
	dot_terraform_applied = filepath.Join(".", ".terraform_applied")
	dot_terraform         = filepath.Join(".", ".terraform")
	awsnycast_bin         = filepath.Join(".", "awsnycast")
	playbook_zip          = filepath.Join(".", "files", "playbook.zip")
)

var Default = All

// Generate all prerequisite files
func All() {
	mg.SerialDeps(SSHKey, Terraform)
}

// Build awsnycast for integration testing
func Go() error {
	return sh.RunWithV(map[string]string{
		"GOOS":        "linux",
		"GOARCH":      "amd64",
		"CGO_ENABLED": "0",
	}, "go", "build", "-o", awsnycast_bin, "../../")
}

// Apply Terraform
func TerraformApplied() error {
	if ok, _ := target.Glob(dot_terraform_applied, "*.tf"); !ok {
		return nil
	}
	if err := sh.RunV("terraform", "apply"); err != nil {
		return err
	}
	time.Sleep(35 * time.Second)
	return os.WriteFile(dot_terraform_applied, nil, os.FileMode(0o644))
}

// Run Integration Tests
func Integration() error {
	mg.SerialDeps(Go, TerraformApplied)
	return sh.RunWithV(map[string]string{
		"INTEGRATION_TESTS": "Y",
	}, "go", "test", "./...")
}

// SSH into the NAT instance
func SSHNat(i int) error {
	body, err := sh.Output("terraform", "output", "-json", "nat_public_ips")
	if err != nil {
		return err
	}
	var ips []string
	if err := json.Unmarshal([]byte(body), &ips); err != nil {
		return err
	}
	if i < len(ips) && i >= 0 {
		cmd := exec.Command("ssh", "-A", "-i", id_rsa, fmt.Sprintf("ubuntu@%s", ips[i]))
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return fmt.Errorf("no ip")
}

// Download Terraform Modules
func Terraform() error {
	if ok, _ := target.Path(dot_terraform); !ok {
		return nil
	}
	if err := sh.RunV("terraform", "init"); err != nil {
		return err
	}
	return nil
}

// Generate RSA Public and Private Keys
func SSHKey() {
	mg.SerialDeps(IDRSA, IDRSAPub)
}

// Generate RSA Private Key
func IDRSA() error {
	if ok, _ := target.Path(id_rsa); !ok {
		return nil
	}
	return sh.RunV("ssh-keygen", "-t", "rsa", "-f", id_rsa, "-N", "")
}

// Generate RSA Public Key
func IDRSAPub() error {
	if ok, _ := target.Path(id_rsa_pub); !ok {
		return nil
	}
	pub, err := sh.Output("ssh-keygen", "-y", "-f", id_rsa)
	if err != nil {
		return err
	}
	return os.WriteFile(id_rsa_pub, []byte(pub), os.FileMode(0o644))
}

// Clean all generated files
func Clean() error {
	for _, target := range []string{dot_terraform, dot_terraform_applied, id_rsa_pub, id_rsa, playbook_zip, awsnycast_bin} {
		if err := os.RemoveAll(target); err != nil {
			return err
		}
	}
	return nil
}

// Destroy Terraform Environment
func TerraformDestroy() error {
	return sh.RunV("terraform", "apply", "-destroy", "-auto-approve")
}

// Destroy Terraform Environment and Clean generated Files
func RealClean() error {
	mg.SerialDeps(Terraform, SSHKey, TerraformDestroy, Clean)
	return nil
}
