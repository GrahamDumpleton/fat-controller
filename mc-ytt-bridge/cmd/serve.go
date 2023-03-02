package cmd

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/GrahamDumpleton/fat-controller/mc-ytt-bridge/pkg/bridge"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/spf13/cobra"
)

const ServeCmdDescription = `
Launches a HTTP server for accepting requests with requests mapped to handlers
implemented as Starlark/YAML templates using Carvel ytt.

The specified handlers directory must contain one or more subdirectories. The
name of the subdirectory is used as the "handler" name. Inside of a handler
directory there can exist one or more Starlark code files with ".star"
extension. The basename part of this file name is used as the "resource" name.
Inside of resource file can exist one or more Starlark functions. The name of
the function is then used as the handler "function" name. To target the handler
function with a JSON HTTP POST request use a URL of the form:

	http://localhost:8080/handler/resource/function

The Starlark function will be executed with keyword arguments corresponding to
the keys in the JSON object supplied in the body of the HTTP request.

A handler function which echoes back the arguments supplied to the handler as
the response is:

	def echo(**kwargs):
		return kwargs
	end

The data values available via the "ytt" library API correspond to the process
configuration.

A handler function which echos back the data values as the response is:

	load("@ytt:data", "data")

	def values(**_):
		return data.values
	end

As all keys from the JSON object supplied in the body of the request are always
supplied as keyword arguments regardless of whether they are used via named
parameters, the handler function must also accept any extra keyword arguments
using a final argument of "**_" or similar. If this is not done then a server
error response will be generated.

The response from the handler function must always be a dictionary and will be
returned as the JSON response to the HTTP request.`

type ServeCmdOptions struct {
	ServerPort        int
	CertificatePath   string
	HandlersDirectory string
	ConfigurationFile string
}

func (o *ServeCmdOptions) Run() error {
	var err error

	certificateFile := ""
	privateKeyFile := ""

	if len(o.CertificatePath) > 0 {
		certificateFile = o.CertificatePath + ".crt"
		privateKeyFile = o.CertificatePath + ".key"

		if _, err := os.Stat(certificateFile); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("certificate file %s does not exist", certificateFile)
		}

		if _, err := os.Stat(privateKeyFile); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("private key file %s does not exist", privateKeyFile)
		}
	}

	// Setup HTTP server routing.

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	br := bridge.YttBridge{
		HandlersDirectory: o.HandlersDirectory,
		ConfigurationFile: o.ConfigurationFile,
	}

	router.Post("/{handler}/{resource}/{function}", br.HandleRequest)

	// Run the HTTP server.

	log.Printf("Using handlers directory: %s\n", o.HandlersDirectory)

	if len(o.CertificatePath) > 0 {
		log.Printf("Using TLS certificate: %s\n", o.CertificatePath)

		addr := ":8443"

		if o.ServerPort != 0 {
			addr = fmt.Sprintf(":%d", o.ServerPort)
		}

		log.Printf("HTTP server listening on: %s\n", addr)

		err = http.ListenAndServeTLS(addr, certificateFile, privateKeyFile, router)
	} else {
		addr := ":8080"

		if o.ServerPort != 0 {
			addr = fmt.Sprintf(":%d", o.ServerPort)
		}

		log.Printf("HTTP server listening on: %s\n", addr)

		err = http.ListenAndServe(addr, router)
	}

	if errors.Is(err, http.ErrServerClosed) {
		log.Printf("HTTP server was shutdown\n")
	} else if err != nil {
		log.Printf("Error starting HTTP server: %s\n", err)
		return err
	}

	return nil
}

func init() {
	var o ServeCmdOptions

	var c = &cobra.Command{
		Use:   "serve",
		Short: "Launches a HTTP server for accepting requests",
		Long:  ServeCmdDescription,
		RunE:  func(_ *cobra.Command, _ []string) error { return o.Run() },
	}

	rootCmd.AddCommand(c)

	c.PersistentFlags().IntVarP(&o.ServerPort, "port", "", 0, "server listener port")
	c.PersistentFlags().StringVarP(&o.CertificatePath, "certificate", "", "", "server TLS certificate path")
	c.PersistentFlags().StringVarP(&o.HandlersDirectory, "handlers", "", "", "handlers root directory")
	c.PersistentFlags().StringVarP(&o.ConfigurationFile, "config", "", "", "handlers config file")

	c.MarkPersistentFlagRequired("handlers")
}
