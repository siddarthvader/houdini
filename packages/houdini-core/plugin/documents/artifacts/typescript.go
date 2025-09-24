package artifacts

import (
	"context"
	"fmt"
	"log"
	"strings"

	"code.houdinigraphql.com/packages/houdini-core/config"
	"code.houdinigraphql.com/plugins"
	"github.com/spf13/afero"
)

func generateTypescriptDefinition(
	ctx context.Context,
	fs afero.Fs,
	db plugins.DatabasePool[config.PluginConfig],
	docs *CollectedDocuments,
	name string,
	selection []*CollectedSelection,
) error {

	projectConfig, err := db.ProjectConfig(ctx)
	if err != nil {
		return err
	}
	doc, exists := docs.Selections[name]
	if !exists {
		return fmt.Errorf("document %s not found", name)
	}

	// log.Printf("doc name %s and kind %v", doc.Name, doc.Kind)
	if doc.Kind == "operation" {
		generateOperationTypesDefs(doc, projectConfig)
	}

	if doc.Kind == "fragment" {
		generateFragmentTypesDefs(name, doc, projectConfig, fs)
	}

	for _, sel := range selection {
		// log.Printf("sel name %s and kind %v", sel.FieldName, sel.Kind)
		if sel.Kind == "fragment" {
			// generateFragmentTypesDefs(name, sel)
		}
	}

	// log.Printf("done generating typescript definition for %s", name)
	// log.Println("trying to see how it works")
	return nil
}

// generting types for fragment
func generateFragmentTypesDefs(name string, doc *CollectedDocument, projectConfig plugins.ProjectConfig, fs afero.Fs) error {
	// log.Printf("generateFragmentTypesDefs name: %s, document name %s, and document kind %s", name, doc.Name, doc.Kind)

	var fragmentString strings.Builder
	inputTypeName := fmt.Sprintf("%s$input", doc.Name)

	if len(doc.Variables) > 0 {
		// log.Printf("generateFragmentTypesDefs")
		processFragmentInputType(inputTypeName, doc.Variables, projectConfig)
	} else {
		log.Printf("no variables for %s", doc.Name)
		fragmentString.WriteString(fmt.Sprintf("export type %s$input = {}\n", doc.Name))
	}

	filaPath := projectConfig.ArtifactTypePath(name)
	log.Printf("filaPath %s", filaPath)
	err := afero.WriteFile(fs, filaPath, []byte(fragmentString.String()), 0644)
	if err != nil {
		return err
	}
	// write fragment string to file

	return nil
}

func processFragmentInputType(inputTypeName string, variables []*CollectedOperationVariable, projectConfig plugins.ProjectConfig) {
	log.Printf("processOperationInputType name: %s, variables %v", inputTypeName, variables)
}

// generting types for query, mutation, subscription
func generateOperationTypesDefs(doc *CollectedDocument, projectConfig plugins.ProjectConfig) error {
	return nil
}
