package integration

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"os/exec"
	"time"
)

func Foo() int {
	return 0
}

func RunMake() {
	cmd := exec.Command("make")
	err := cmd.Run()
	out, _ := cmd.CombinedOutput()
	if err != nil {
		Fail(fmt.Sprintf("Unable to run 'make', output: %s", string(out)))
	}
}

func RunTerraform() {
	command := exec.Command("terraform", "apply")
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Ω(err).ShouldNot(HaveOccurred())
	session.Wait(1800 * time.Second)
	//Eventually(session, time.Second*300).Should(gexec.Exit(0))
}
