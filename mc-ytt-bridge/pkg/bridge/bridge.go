/*
Package bridge provides a HTTP handler which maps requests to ytt command
execution.
*/
package bridge

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	yttcmd "github.com/vmware-tanzu/carvel-ytt/pkg/cmd/template"
	"github.com/vmware-tanzu/carvel-ytt/pkg/cmd/ui"
	"github.com/vmware-tanzu/carvel-ytt/pkg/files"
	"github.com/vmware-tanzu/carvel-ytt/pkg/yamlmeta"
)

// Encapsulates details as to the location and configuration of the bridge.
type YttBridge struct {
	HandlersDirectory string
	ConfigurationFile string
}

// Encapsulates details of a specific endpoint for handling requests.
type YttBridgeEndpoint struct {
	HandlerName  string
	ResourceName string
	FunctionName string
}

// Encapsulates details of a specific HTTP request handled by the bridge.
type YttBridgeRequest struct {
	RequestMethod string
	QueryParams   map[string][]string
	RequestBody   []byte
}

const mainTemplate = `
#@ load("@ytt:data", "data")
#@ load("@ytt:json", "json")
#@ load("@ytt:struct", "struct")
#@ load("@ytt:template", "template")
#@ load("%s/%s.star", handler="%s")
---
_: #@ template.replace(handler(**struct.encode(json.decode(data.read("__body__.yaml")))))
`

/*
Executes the Starlark function designated as the endpoint, providing the details
of the JSON HTTP request. The object returned from the Starlark function is
returned as a JSON byte string.

The Starlark function serving as the endpoint needs to be contained in the file
with path of form:

	handler-directory/handler/resource.star

where "handler" and "resource" are defined as components on the endpoint. The
name of the function should correspond to the "function" component of the
endpoint.

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
return as a formatted JSON byte string.
*/
func (br YttBridge) ExecuteCommand(bre YttBridgeEndpoint, brr YttBridgeRequest) ([]byte, error) {
	// Construct options for executing ytt. Rather than executing ytt on the
	// command line we are using it embedded within this process. First set up
	// data values using the supplied process configuration.

	templatingOptions := yttcmd.NewOptions()

	if br.ConfigurationFile != "" {
		templatingOptions.DataValuesFlags.FromFiles = []string{br.ConfigurationFile}
	}

	// Next construct input files for executing ytt. Note that we need to add a
	// sort order to the files so that the main input file is always output
	// first and to prevent ytt from doing a panic when sort order isn't
	// applied.

	var filesToProcess []*files.File
	var symlinkOptions files.SymlinkAllowOpts

	mainInput := []byte(fmt.Sprintf(mainTemplate, bre.HandlerName, bre.ResourceName, bre.FunctionName))
	mainInputFile := files.MustNewFileFromSource(files.NewBytesSource("__main__.yaml", mainInput))

	filesToProcess = append(filesToProcess, mainInputFile)

	bodyInputFile := files.MustNewFileFromSource(files.NewBytesSource("__body__.yaml", brr.RequestBody))

	filesToProcess = append(filesToProcess, bodyInputFile)

	randomData := make([]byte, 1024)

	_, err := rand.Read(randomData)

	if err != nil {
		return []byte{}, fmt.Errorf("failed to generate random data ytt: %s", err)
	}

	randomInputFile := files.MustNewFileFromSource(files.NewBytesSource("__random__.dat", randomData))

	filesToProcess = append(filesToProcess, randomInputFile)

	fileMarks := []string{
		"__body__.yaml:type=data",
		"__random__.dat:type=data",
	}

	templatingOptions.FileMarksOpts.FileMarks = fileMarks

	sourceInputFiles, err := files.NewSortedFilesFromPaths([]string{br.HandlersDirectory}, symlinkOptions)

	if err != nil {
		return []byte{}, fmt.Errorf("failed to construct sources for ytt: %s", err)
	}

	filesToProcess = append(filesToProcess, sourceInputFiles...)

	filesToProcess = files.NewSortedFiles(filesToProcess)

	// Execute ytt. Any errors and stderr will be written to log output. The
	// stdout will be captured and used in the response.

	logUI := ui.NewCustomWriterTTY(false, log.Writer(), log.Writer())

	output := templatingOptions.RunWithFiles(yttcmd.Input{Files: filesToProcess}, logUI)

	if output.Err != nil {
		return []byte{}, fmt.Errorf("execution of ytt failed: %s", output.Err)
	}

	// Return the response. Only the first document in the document set is used
	// in the response, which will be the main input file as it was set to be
	// first in order. There would only be more than one output file in the
	// document set if YAML files were wrongly included in the application
	// directory.

	if len(output.DocSet.Items) == 0 {
		return []byte{}, nil
	}

	var buf bytes.Buffer

	yamlmeta.NewJSONPrinter(&buf).Print(output.DocSet.Items[0])

	return buf.Bytes(), nil
}

/*
HTTP request handler which takes a JSON HTTP request and maps it to the
execution of the ytt command.

It is expected that the HTTP request handler is mapped using the chi router with
the "handler", "resource" and "function" extracted as path parameters from the
URL.

	router := chi.NewRouter()

	br := bridge.YttBridge{
	    HandlersDirectory: handlersDirectory,
	}

	router.Post("/{handler}/{resource}/{function}", br.HandleRequest)

The URL for a HTTP POST request would then be of the form:

	http://localhost:8080/handler/resource/function

The request body content should be a JSON object.

The details for the handler function, along with the query string parameters and
JSON request are mapped to execution of the ytt command using ExecuteCommand().
*/
func (br YttBridge) HandleRequest(w http.ResponseWriter, r *http.Request) {
	// Validate request body is JSON.

	contentType := r.Header.Get("Content-type")

	if contentType != "application/json" {
		log.Printf("Unexpected content type: %s\n", contentType)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Read in the entire request body.

	body, err := io.ReadAll(r.Body)

	if err != nil {
		log.Printf("Could not read request body: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Process the request.

	bre := YttBridgeEndpoint{
		HandlerName:  chi.URLParam(r, "handler"),
		ResourceName: chi.URLParam(r, "resource"),
		FunctionName: chi.URLParam(r, "function"),
	}

	brr := YttBridgeRequest{
		RequestMethod: r.Method,
		QueryParams:   r.URL.Query(),
		RequestBody:   body,
	}

	res, err := br.ExecuteCommand(bre, brr)

	if err != nil {
		log.Printf("Execution of ytt failed: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")

	w.Write(res)
}
