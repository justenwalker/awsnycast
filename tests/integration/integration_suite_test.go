package integration

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var internalIPs []string

func TestIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "Y" {
		t.Skip("Skipping Integration tests. Set INTEGRATION_TESTS=Y to run them")
	}
	RegisterFailHandler(Fail)

	//RunMage()
	//RunTerraform()
	internalIPs = InternalIPs()
	RunSpecs(t, "Integration Suite")
}
