package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/log"
	yamlutil "sigs.k8s.io/yaml"
)

// GetResource converts raw bytes to unstructured object
func GetResource(resourceBytes []byte) ([]*unstructured.Unstructured, error) {
	resources := make([]*unstructured.Unstructured, 0)
	var getErrString string

	files, splitDocError := SplitDocuments(resourceBytes)
	if splitDocError != nil {
		return nil, splitDocError
	}

	for _, resourceYaml := range files {
		resource, err := convertResourceToUnstructured(resourceYaml)
		if err != nil {
			if strings.Contains(err.Error(), "Object 'Kind' is missing") {
				log.Log.V(3).Info("skipping resource as kind not found")
				continue
			}
			getErrString = getErrString + err.Error() + "\n"
		}
		resources = append(resources, resource)
	}

	if getErrString != "" {
		return nil, errors.New(getErrString)
	}

	return resources, nil
}

func convertResourceToUnstructured(resourceYaml []byte) (*unstructured.Unstructured, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	_, metaData, decodeErr := decode(resourceYaml, nil, nil)

	if decodeErr != nil {
		if !strings.Contains(decodeErr.Error(), "no kind") {
			return nil, decodeErr
		}
	}
	fmt.Println("metadata: ", metaData.String())
	fmt.Println("resourceYaml: ", string(resourceYaml))

	resourceJSON, err := yamlutil.YAMLToJSON(resourceYaml)
	if err != nil {
		return nil, err
	}
	fmt.Println("resourceJson: ", string(resourceJSON))

	resource, err := bytesToUnstructured(resourceJSON)
	if err != nil {
		return nil, err
	}

	if decodeErr == nil {
		resource.SetGroupVersionKind(*metaData)
	}

	if resource.GetNamespace() == "" {
		resource.SetNamespace("default")
	}
	return resource, nil
}

// SplitDocuments reads the YAML bytes per-document, unmarshals the TypeMeta information from each document
// and returns a map between the GroupVersionKind of the document and the document bytes
func SplitDocuments(yamlBytes []byte) (documents [][]byte, error error) {
	buf := bytes.NewBuffer(yamlBytes)
	reader := yaml.NewYAMLReader(bufio.NewReader(buf))
	for {
		// Read one YAML document at a time, until io.EOF is returned
		b, err := reader.Read()
		fmt.Println("document bytes: ", string(b))
		if err == io.EOF || len(b) == 0 {
			break
		} else if err != nil {
			return documents, fmt.Errorf("unable to read yaml")
		}
		documents = append(documents, b)
	}
	return documents, nil
}

// bytesToUnstructured converts the resource to unstructured format
func bytesToUnstructured(data []byte) (*unstructured.Unstructured, error) {
	resource := &unstructured.Unstructured{}
	err := resource.UnmarshalJSON(data)
	if err != nil {
		return nil, err
	}
	return resource, nil
}
