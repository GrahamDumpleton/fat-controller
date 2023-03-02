package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/GrahamDumpleton/fat-controller/mc-ytt-bridge/pkg/bridge"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type TestCmdOptions struct {
	HandlersDirectory string
	ConfigurationFile string
	HandlerName       string
	ResourceName      string
	FunctionName      string
	InputFile         string
	OutputFormat      string
}

func (o *TestCmdOptions) Run() error {
	var err error

	br := bridge.YttBridge{
		HandlersDirectory: o.HandlersDirectory,
		ConfigurationFile: o.ConfigurationFile,
	}

	bre := bridge.YttBridgeEndpoint{
		HandlerName:  o.HandlerName,
		ResourceName: o.ResourceName,
		FunctionName: o.FunctionName,
	}

	// Read input file from stdin or filer as required.

	var body []byte

	if o.InputFile == "-" {
		body, err = io.ReadAll(os.Stdin)

		if err != nil {
			return fmt.Errorf("unable to read input from stdin")
		}
	} else {
		body, err = os.ReadFile(o.InputFile)

		if err != nil {
			return fmt.Errorf("unable to read input from %s", o.InputFile)
		}
	}

	// Interpret file as YAML and translate it to JSON.

	var input map[string]interface{}

	if err := yaml.Unmarshal(body, &input); err != nil {
		return fmt.Errorf("unable to decode input as yaml: %v", err)
	}

	body, err = json.Marshal(&input)

	if err != nil {
		return fmt.Errorf("unable to encode input as json: %v", err)
	}

	// Handle request.

	brr := bridge.YttBridgeRequest{
		RequestMethod: "POST",
		QueryParams:   map[string][]string{},
		RequestBody:   body,
	}

	res, err := br.ExecuteCommand(bre, brr)

	if err != nil {
		return fmt.Errorf("execution of ytt failed: %v", err)
	}

	// Write output as JSON or YAML as required.

	if o.OutputFormat == "json" {
		if len(res) == 0 {
			os.Stdout.WriteString("{}")
		} else {
			os.Stdout.Write(res)
		}

		os.Stdout.WriteString("\n")

		return nil
	}

	if len(res) == 0 {
		return nil
	}

	var output map[string]interface{}

	if err := json.Unmarshal(res, &output); err != nil {
		return fmt.Errorf("unable to decode output as json: %v", err)
	}

	res, err = yaml.Marshal(&output)

	if err != nil {
		return fmt.Errorf("unable to encode output as yaml: %v", err)
	}

	os.Stdout.Write(res)

	return nil
}

func init() {
	var o TestCmdOptions

	var c = &cobra.Command{
		Use:   "test",
		Short: "Test a single handler for processing a request",
		RunE:  func(_ *cobra.Command, _ []string) error { return o.Run() },
	}

	rootCmd.AddCommand(c)

	c.PersistentFlags().StringVarP(&o.HandlersDirectory, "handlers", "", "", "handlers root directory")
	c.PersistentFlags().StringVarP(&o.ConfigurationFile, "config", "", "", "handlers config file")

	c.MarkPersistentFlagRequired("handlers")

	c.PersistentFlags().StringVarP(&o.HandlerName, "handler", "", "", "handler name")
	c.PersistentFlags().StringVarP(&o.ResourceName, "resource", "", "", "resource name")
	c.PersistentFlags().StringVarP(&o.FunctionName, "function", "", "", "function name")

	c.MarkPersistentFlagRequired("handler")
	c.MarkPersistentFlagRequired("resource")
	c.MarkPersistentFlagRequired("function")

	c.PersistentFlags().StringVarP(&o.InputFile, "file", "f", "-", "input file")
	c.PersistentFlags().StringVarP(&o.OutputFormat, "output", "o", "yaml", "output format")
}
