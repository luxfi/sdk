// Copyright (C) 2021-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/luxdefi/node/api/health"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-cmd/cmd"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// RunCommand starts the command [bin] with the given [args] and returns the command to the caller
// TODO cmd package mentions we can do this more efficiently with cmd.NewCmdOptions rather than looping
// and calling Status().
func RunCommand(bin string, args ...string) (*cmd.Cmd, error) {
	log.Info("Executing", "cmd", fmt.Sprintf("%s %s", bin, strings.Join(args, " ")))

	curCmd := cmd.NewCmd(bin, args...)
	_ = curCmd.Start()

	// to stream outputs
	ticker := time.NewTicker(10 * time.Millisecond)
	go func() {
		prevLine := ""
		for range ticker.C {
			status := curCmd.Status()
			n := len(status.Stdout)
			if n == 0 {
				continue
			}

			line := status.Stdout[n-1]
			if prevLine != line && line != "" {
				fmt.Println("[streaming output]", line)
			}

			prevLine = line
		}
	}()

	return curCmd, nil
}

func RegisterPingTest() {
	ginkgo.It("ping the network", ginkgo.Label("ping"), func() {
		client := health.NewClient(DefaultLocalNodeURI)
		healthy, err := client.Readiness(context.Background(), nil)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(healthy.Healthy).Should(gomega.BeTrue())
	})
}

<<<<<<< HEAD
// RegisterNodeRun registers a before suite that starts an Lux Node process to use for the e2e tests
// and an after suite that stops the Lux Node process
func RegisterNodeRun() {
	// BeforeSuite starts an Lux Node process to use for the e2e tests
=======
// RegisterNodeRun registers a before suite that starts an Luxd process to use for the e2e tests
// and an after suite that stops the Luxd process
func RegisterNodeRun() {
	// BeforeSuite starts an Luxd process to use for the e2e tests
>>>>>>> b36c20f (Update executable to luxd)
	var startCmd *cmd.Cmd
	_ = ginkgo.BeforeSuite(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		wd, err := os.Getwd()
		gomega.Expect(err).Should(gomega.BeNil())
<<<<<<< HEAD
		log.Info("Starting Lux Node node", "wd", wd)
=======
		log.Info("Starting Luxd node", "wd", wd)
>>>>>>> b36c20f (Update executable to luxd)
		cmd, err := RunCommand("./scripts/run.sh")
		startCmd = cmd
		gomega.Expect(err).Should(gomega.BeNil())

		// Assumes that startCmd will launch a node with HTTP Port at [utils.DefaultLocalNodeURI]
		healthClient := health.NewClient(DefaultLocalNodeURI)
		healthy, err := health.AwaitReady(ctx, healthClient, HealthCheckTimeout, nil)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(healthy).Should(gomega.BeTrue())
<<<<<<< HEAD
		log.Info("Lux Node node is healthy")
=======
		log.Info("Luxd node is healthy")
>>>>>>> b36c20f (Update executable to luxd)
	})

	ginkgo.AfterSuite(func() {
		gomega.Expect(startCmd).ShouldNot(gomega.BeNil())
		gomega.Expect(startCmd.Stop()).Should(gomega.BeNil())
		// TODO add a new node to bootstrap off of the existing node and ensure it can bootstrap all subnets
		// created during the test
	})
}

// RunDefaultHardhatTests runs the hardhat tests in the given [testPath] on the blockchain with [blockchainID]
// [execPath] is the path where the test command is executed
func RunHardhatTests(ctx context.Context, blockchainID string, execPath string, testPath string) {
	chainURI := GetDefaultChainURI(blockchainID)
	RunHardhatTestsCustomURI(ctx, chainURI, execPath, testPath)
}

func RunHardhatTestsCustomURI(ctx context.Context, chainURI string, execPath string, testPath string) {
	log.Info(
		"Executing HardHat tests on blockchain",
		"testPath", testPath,
		"ChainURI", chainURI,
	)

	cmd := exec.Command("npx", "hardhat", "test", testPath, "--network", "local")
	cmd.Dir = execPath

	log.Info("Sleeping to wait for test ping", "rpcURI", chainURI)
	err := os.Setenv("RPC_URI", chainURI)
	gomega.Expect(err).Should(gomega.BeNil())
	log.Info("Running test command", "cmd", cmd.String())

	out, err := cmd.CombinedOutput()
	fmt.Printf("\nCombined output:\n\n%s\n", string(out))
	gomega.Expect(err).Should(gomega.BeNil())
}
