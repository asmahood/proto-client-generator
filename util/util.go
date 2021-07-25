package util

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	LanguageGo         = "golang"
	LanguageRuby       = "ruby"
	LanguagePython     = "python"
	LanguageJava       = "java"
	LanguageJavascript = "javascript"

	ServiceAudit         = "audit"
	ServiceAuthorization = "authorization"
	ServiceCatalog       = "catalog"
	ServiceCategory      = "category"
	ServiceDataspec      = "dataspec"
	ServiceExports       = "exports"
	ServiceGrants        = "grants"
	ServiceJabba         = "jabba"
	ServiceOrganizations = "organizations"
	ServiceParser        = "parser"
	ServiceQuery         = "query"
	ServiceReferences    = "references"
	ServiceSearch        = "search"
	ServiceSources       = "sources"
	ServiceTaskrunner    = "taskrunner"
	ServiceUploads       = "uploads"
	ServiceWarehouses    = "warehouses"
)

// IsValidLanguage returns true if lang is supported to generate client code. Returns false otherwise
func IsValidLanguage(lang string) bool {
	switch lang {
	case LanguageGo, LanguageRuby, LanguagePython, LanguageJava, LanguageJavascript:
		return true
	default:
		return false
	}
}

// IsValidPublicService returns true if s has a public protobuf defined. Returns false otherwise.
func IsValidPublicService(s string) bool {
	switch s {
	case ServiceAuthorization, ServiceCatalog, ServiceCategory, ServiceDataspec, ServiceExports, ServiceGrants, ServiceOrganizations,
		ServiceQuery, ServiceReferences, ServiceSearch, ServiceSources, ServiceUploads, ServiceWarehouses:
		return true
	default:
		return false
	}
}

// IsValidPrivateService returns true if s has a private protobuf defined. Returns false otherwise.
func IsValidPrivateService(s string) bool {
	switch s {
	case ServiceAudit, ServiceJabba, ServiceParser, ServiceCatalog, ServiceCategory, ServiceExports, ServiceGrants,
		ServiceOrganizations, ServiceQuery, ServiceReferences, ServiceSources, ServiceWarehouses, ServiceTaskrunner:
		return true
	default:
		return false
	}
}

func CleanUpDirectories(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		log.Fatalf("Error: Could not remove directory '%s': %s", dir, err.Error())
	}
}

func CloneService(service string, dir string) (string, error) {
	src := filepath.Join(dir, service)
	err := exec.Command("git", "clone", fmt.Sprintf("git@github.com:asmahood/%s.git", service), src).Run()
	if err != nil {
		return "", fmt.Errorf("failed to clone service: %s", err.Error())
	}

	return src, nil
}

func CopyProtobuf(service string, serviceDir string, protoDir string, private bool) error {
	serviceProtoDir := ""
	if private {
		serviceProtoDir = filepath.Join(serviceDir, "proto", "private")
	} else {
		serviceProtoDir = filepath.Join(serviceDir, "proto", "public")
	}

	files, err := os.ReadDir(serviceProtoDir)
	if err != nil {
		return fmt.Errorf("failed to read service protobuf directory: %s", err.Error())
	}

	for _, f := range files {
		// Ignore any files that are not protobuf files
		if filepath.Ext(f.Name()) != ".proto" {
			continue
		}

		src, err := os.Open(filepath.Join(serviceProtoDir, f.Name()))
		if err != nil {
			return fmt.Errorf("cannot open source protobuf file: %s", err.Error())
		}
		defer src.Close()

		dst, err := os.Create(filepath.Join(protoDir, fmt.Sprintf("%s.proto", service)))
		if err != nil {
			return fmt.Errorf("cannot create protobuf file: %s", err.Error())
		}
		defer dst.Close()

		_, err = io.Copy(dst, src)
		if err != nil {
			return fmt.Errorf("cannot copy protobuf file: %s", err)
		}
	}

	return nil
}

func goGenerateCmd(service string, dir string) *exec.Cmd {
	return exec.Command("protoc", fmt.Sprintf("--twirp_out=paths=source_relative:%s", dir), fmt.Sprintf("--go_out=paths=source_relative:%s", dir), fmt.Sprintf("--proto_path=%s", dir), filepath.Join(dir, fmt.Sprintf("%s.proto", service)))
}

func rubyGenerateCmd(service string, dir string) *exec.Cmd {
	return exec.Command("protoc", fmt.Sprintf("--proto_path=%s", dir), fmt.Sprintf("--twirp_ruby_out=%s", dir), fmt.Sprintf("--ruby_out=%s", dir), filepath.Join(dir, fmt.Sprintf("%s.proto", service)))
}

func pythonGenerateCmd(service string, dir string) *exec.Cmd {
	return exec.Command("protoc", fmt.Sprintf("--proto_path=%s", dir), fmt.Sprintf("--twirpy_out=%s", dir), fmt.Sprintf("--python_out=%s", dir), filepath.Join(dir, fmt.Sprintf("%s.proto", service)))
}

func javascriptGenerateCmd(service string, dir string) *exec.Cmd {
	return exec.Command("protoc", fmt.Sprintf("--proto_path=%s", dir), fmt.Sprintf("--twirp_js_out=%s", dir), fmt.Sprintf("--js_out=import_style=commonjs,binary:%s", dir), filepath.Join(dir, fmt.Sprintf("%s.proto", service)))
}

func GenerateCode(language string, service string, dir string) error {
	var protocCmd *exec.Cmd
	switch language {
	case LanguageGo:
		protocCmd = goGenerateCmd(service, dir)
	case LanguageRuby:
		protocCmd = rubyGenerateCmd(service, dir)
	case LanguagePython:
		protocCmd = pythonGenerateCmd(service, dir)
	case LanguageJavascript:
		protocCmd = javascriptGenerateCmd(service, dir)
	default:
		return errors.New("no command has been implemented for this language")
	}

	out, err := protocCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to pipe command output: %s", err.Error())
	}
	errOut, err := protocCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to pipe command error output: %s", err.Error())
	}

	err = protocCmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start client generator: %s", err.Error())
	}

	logs, err := ioutil.ReadAll(out)
	if err != nil {
		return fmt.Errorf("failed to read output from command: %s", err.Error())
	} else if len(logs) > 0 {
		log.Printf("\n\n%s\n\n", logs)
	}

	logs, err = io.ReadAll(errOut)
	if err != nil {
		return fmt.Errorf("failed to read error from command: %s", err.Error())
	} else if len(logs) > 0 {
		log.Printf("Generator encountered error:\n\n%s\n", logs)
	}

	err = protocCmd.Wait()
	if err != nil {
		return fmt.Errorf("failed to run generator command: %s", err.Error())
	}

	return nil
}

func CopyGeneratedFiles(protoDir string, outputPath string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot locate current working directory: %s", err)
	}

	files, err := os.ReadDir(protoDir)
	if err != nil {
		return fmt.Errorf("failed to read protobuf directory: %s", err.Error())
	}

	for _, f := range files {
		// Do not copy any .proto files to the output
		if filepath.Ext(f.Name()) == ".proto" {
			continue
		}

		src, err := os.Open(filepath.Join(protoDir, f.Name()))
		if err != nil {
			return fmt.Errorf("failed to open generated file: %s", err.Error())
		}

		dst, err := os.Create(filepath.Join(cwd, outputPath, f.Name()))
		if err != nil {
			return fmt.Errorf("failed to create generated file in output: %s", err.Error())
		}

		_, err = io.Copy(dst, src)
		if err != nil {
			return fmt.Errorf("failed to copy generated file to output: %s", err.Error())
		}
	}

	return nil
}
