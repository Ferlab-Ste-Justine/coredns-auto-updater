package etcdserver

import (
	"embed"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/hashicorp/terraform-exec/tfexec"
)

//go:embed terraform
var terraformFiles embed.FS

type EtcdServer struct {
	TerraformScriptsPath string
	TerraformExec *tfexec.Terraform
}

func copyProviderFile(terraformProviderFilePath string, terraformScriptsPath string) error {
	input, provErr := ioutil.ReadFile(terraformProviderFilePath)
	if provErr != nil {
		return errors.New(fmt.Sprintf("Error reading provider file: %s", provErr.Error()))
	}

	provErr = ioutil.WriteFile(path.Join(terraformScriptsPath, "provider.tf"), input, 0644)
	if provErr != nil {
		return errors.New(fmt.Sprintf("Error copying provider file: %s", provErr.Error()))
	}

	return nil
}

func copy(embeddedDir string, terraformScriptsPath string) error {
	elems, dirErr := terraformFiles.ReadDir(embeddedDir)
	if dirErr != nil {
		return errors.New(fmt.Sprintf("Error traversing embedded terraform directory: %s", dirErr.Error()))
	}

	for _, elem := range elems {
		elemSource := path.Join(embeddedDir, elem.Name())
		elemDest := path.Join(terraformScriptsPath, strings.TrimPrefix(path.Join(embeddedDir, elem.Name()), "terraform/"))
		if !elem.IsDir() {
			content, readErr := terraformFiles.ReadFile(elemSource)
			if readErr != nil {
				return errors.New(fmt.Sprintf("Error reading embedded terraform file: %s", readErr.Error()))
			}

			writeErr := ioutil.WriteFile(elemDest, content, 0644)
			if writeErr != nil {
				return errors.New(fmt.Sprintf("Error writing terraform file: %s", writeErr.Error()))
			}
		} else {
			err := os.MkdirAll(elemDest, os.ModePerm)
			if err != nil {
				return errors.New(fmt.Sprintf("Error creating terraform scripts directory: %s", err.Error()))
			}
			err = copy(elemSource, terraformScriptsPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewEtcdServer(terraformBinPath string, terraformProviderFilePath string, terraformScriptsPath string) (*EtcdServer, error) {
	err := os.MkdirAll(terraformScriptsPath, os.ModePerm)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error creating terraform scripts directory: %s", err.Error()))
	}

	err = copyProviderFile(terraformProviderFilePath, terraformScriptsPath)
	if err != nil {
		return nil, err
	}

	copyErr := copy("terraform", terraformScriptsPath)
	if copyErr != nil {
		return nil, copyErr
	}

	tf, err := tfexec.NewTerraform(terraformScriptsPath, terraformBinPath)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error Setting Up Terraform %s", err.Error()))
	}

	tf.SetStdout(os.Stdout)
	tf.SetStderr(os.Stderr)

	err = tf.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error Running Terraform Init: %s", err.Error()))
	}

	err = tf.Apply(context.Background())
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error Running Terraform apply: %s", err.Error()))
	}

	return &EtcdServer{TerraformScriptsPath: terraformScriptsPath, TerraformExec: tf}, nil
}

func (s *EtcdServer) Close() error {
	err := s.TerraformExec.Destroy(context.Background())
	if err != nil {
		return errors.New(fmt.Sprintf("Error running terraform destroy: %s", err.Error()))
	}

    err = os.RemoveAll(s.TerraformScriptsPath)
    if err != nil {
        return errors.New(fmt.Sprintf("Error cleaning up terraform scripts directory: %s", err.Error()))
    }

	return nil
}

func (s *EtcdServer) GetCaCert() ([]byte, error) {
	certPath := path.Join(s.TerraformScriptsPath, "certs", "ca.pem")
	return ioutil.ReadFile(certPath)
}

func (s *EtcdServer) GetRootCert() ([]byte, error) {
	certPath := path.Join(s.TerraformScriptsPath, "certs", "root.pem")
	return ioutil.ReadFile(certPath)
}

func (s *EtcdServer) GetRootKey() ([]byte, error) {
	keyPath := path.Join(s.TerraformScriptsPath, "certs", "root.key")
	return ioutil.ReadFile(keyPath)
}