package integration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"golang.org/x/crypto/ssh"
)

func Foo() int {
	return 0
}

func RunMage() {
	cmd := exec.Command("mage")
	err := cmd.Run()
	out, _ := cmd.CombinedOutput()
	if err != nil {
		Fail(fmt.Sprintf("Unable to run 'mage', output: %s", string(out)))
	}
}

func RunTerraform() {
	command := exec.Command("terraform", "apply", "-auto-approve")
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Ω(err).ShouldNot(HaveOccurred())
	session.Wait(1800 * time.Second)
	Eventually(session).Should(gexec.Exit(0))
}

func SshKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err.Error())
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		panic(err.Error())
	}
	return ssh.PublicKeys(key)
}

func NatIPs() []string {
	output, err := exec.Command("terraform", "output", "-json", "nat_public_ips").Output()
	Ω(err).ShouldNot(HaveOccurred())
	var ips []string
	err = json.Unmarshal(output, &ips)
	Ω(err).ShouldNot(HaveOccurred())
	return ips
}

func InternalIPs() []string {
	output, err := exec.Command("terraform", "output", "-json", "internal_private_ips").Output()
	Ω(err).ShouldNot(HaveOccurred())
	var ips []string
	err = json.Unmarshal(output, &ips)
	Ω(err).ShouldNot(HaveOccurred())
	return ips
}

func NatA() string {
	ips := NatIPs()
	return ips[0]
}

func NatB() string {
	ips := NatIPs()
	return ips[1]
}

func stream(command string, session *ssh.Session) (output chan string, done chan bool, err error) {
	outReader, err := session.StdoutPipe()
	Ω(err).ShouldNot(HaveOccurred())
	errReader, err := session.StderrPipe()
	Ω(err).ShouldNot(HaveOccurred())
	outputReader := io.MultiReader(outReader, errReader)
	err = session.Start(command)
	Ω(err).ShouldNot(HaveOccurred())
	scanner := bufio.NewScanner(outputReader)
	outputChan := make(chan string)
	done = make(chan bool)
	go func(scanner *bufio.Scanner, out chan string, done chan bool) {
		defer close(outputChan)
		defer close(done)
		for scanner.Scan() {
			outputChan <- scanner.Text()
		}
		done <- true
		session.Close()
	}(scanner, outputChan, done)
	return outputChan, done, err
}

func Ssh(cmd string, host string) (outStr string) {
	sshConfig := &ssh.ClientConfig{
		User: "ubuntu",
		Auth: []ssh.AuthMethod{
			SshKeyFile("id_rsa"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	connection, err := ssh.Dial("tcp", host+":22", sshConfig)
	Ω(err).ShouldNot(HaveOccurred())
	session, err := connection.NewSession()
	Ω(err).ShouldNot(HaveOccurred())
	outChan, doneChan, err := stream(cmd, session)
	Ω(err).ShouldNot(HaveOccurred())
	stillGoing := true
	for stillGoing {
		select {
		case <-doneChan:
			stillGoing = false
		case line := <-outChan:
			outStr += line + "\n"
		}
	}
	session.Close()
	return outStr
}
