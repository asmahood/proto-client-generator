package cmd

import (
	"log"
	"os"
	"path/filepath"

	"github.com/asmahood/proto-client-generator/util"
	"github.com/spf13/cobra"
)

var (
	language   string
	service    string
	private    bool
	outputPath string
)

/*
Command workflow:

1. Validate language flag is one of the support SDK languages

2. Validate service is a valid microservice in the stack

3. Setup temporary directories. This will be used to pull down services from Github, and to generate the code into

4. Pull source code from Github and clone into the temp directory

5. Copy proto file from either public/ or private/ (based on flag)

6. Run protoc generation command based on language specified

7. Copy generated files to output path

8. Clean up temporary directories

*/

var rootCmd = &cobra.Command{
	Use:     "generate-clients",
	Short:   "Use to generate server/client code from protobuf files",
	Long:    ``,
	Example: "generate-clients -l ruby -s catalog -o ./namara-ruby/lib/rpc/catalog",
	Run: func(cmd *cobra.Command, args []string) {
		// Validate we can generate code for the inputted language
		if valid := util.IsValidLanguage(language); !valid {
			log.Fatalf("Error: Client code generation is not supported for '%s'\n", language)
		}

		// Validate that a public service exists for this service
		if valid := util.IsValidPublicService(service); !private && !valid {
			log.Fatalf("Error: The service '%s' does not have a public protobuf defined\n", service)
		}

		// If we are generating private code, validate the service has defined a private protobuf
		if valid := util.IsValidPrivateService(service); private && !valid {
			log.Fatalf("Error: The service '%s' does not have a private protobuf defined\n", service)
		}


		// Create temporary directory to download service source code to
		tmpDir, err := os.MkdirTemp(os.TempDir(), "client-generation-")
		if err != nil {
			log.Fatalf("Error: Cannot create temporary directory: %s\n", err.Error())
		}
		defer util.CleanUpDirectories(tmpDir)
		log.Printf("Created temporary directory %s", tmpDir)

		// Create protobuf directory to hold .proto files
		protoDir := filepath.Join(tmpDir, "proto")
		err = os.Mkdir(protoDir, os.ModeDir)
		if err != nil {
			util.CleanUpDirectories(tmpDir)
			log.Fatalf("Error: Cannot create protobuf directory: %s", err.Error())
		}

		// Clone service source into temp directory
		serviceDir, err := util.CloneService(service, tmpDir)
		if err != nil {
			util.CleanUpDirectories(tmpDir)
			log.Fatalf("Error: %s", err.Error())
		}

		// Copy either public or private proto file into the proto directory
		err = util.CopyProtobuf(service, serviceDir, protoDir, private)
		if err != nil {
			util.CleanUpDirectories(tmpDir)
			log.Fatalf("Error: %s", err.Error())
		}

		// Generate client code based on lanaguage
		err = util.GenerateCode(language, service, protoDir)
		if err != nil {
			util.CleanUpDirectories(tmpDir)
			log.Fatalf("Error: %s", err.Error())
		}

		// Copy generated files to output directory
		err = util.CopyGeneratedFiles(protoDir, outputPath)
		if err != nil {
			util.CleanUpDirectories(tmpDir)
			log.Fatalf("Error: %s", err.Error())
		}
	},
}

func init() {
	// Initialize command flags
	rootCmd.Flags().StringVarP(&language, "language", "l", "", "The language of the generated output code. Valid values are: golang, ruby, python, javascript")
	rootCmd.Flags().StringVarP(&service, "service", "s", "all", "The service to generate client code for. Currently generating for all services is not supported")
	rootCmd.Flags().StringVarP(&outputPath, "output", "o", "", "The path to output the generated code. This path is relative to your current working directory")
	rootCmd.Flags().BoolVarP(&private, "private", "p", false, "Will use private protobuf files to generate code instead of public protobufs")
	rootCmd.MarkFlagRequired("language")
	rootCmd.MarkFlagRequired("output")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
