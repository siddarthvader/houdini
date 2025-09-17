package schema

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"zombiezen.com/go/sqlite"

	"code.houdinigraphql.com/packages/houdini-core/config"
	"code.houdinigraphql.com/plugins"
)

type Directive struct {
	Name        string
	Internal    bool
	Repeatable  bool
	Description string
	Arguments   []*Argument
	Locations   []string
}

type Argument struct {
	Name          string
	Type          string
	TypeModifiers string
	DefaultValue  string
}

func GenerateDefinitionFiles(
	ctx context.Context,
	db plugins.DatabasePool[config.PluginConfig],
	fs afero.Fs,
	sortKeys bool,
) error {
	projectConfig, err := db.ProjectConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get project config: %w", err)
	}

	// generate schema.graphql
	err = generateSchemaFile(ctx, db, fs, projectConfig)
	if err != nil {
		return err
	}

	// generate documents.gql
	err = generateDocumentsFile(ctx, db, fs, projectConfig)
	if err != nil {
		return err
	}

	// generate enum files (enums.js, enums.d.ts, index files)
	err = generateEnumFiles(ctx, db, fs, projectConfig)
	if err != nil {
		return err
	}

	return nil
}

func generateSchemaFile(ctx context.Context, db plugins.DatabasePool[config.PluginConfig], fs afero.Fs, projectConfig plugins.ProjectConfig) error {
	directives := make(map[string]*Directive)
	errs := &plugins.ErrorList{}
	customTypes := make(map[string]bool)

	var schemaString strings.Builder
	// get all internal directives
	err := db.StepQuery(ctx, `
		SELECT name, internal, repeatable, description
		FROM directives
		WHERE internal = 1
	`, nil, func(stmt *sqlite.Stmt) {
		name := stmt.ColumnText(0)
		internal := stmt.ColumnInt(1) == 1
		repeatable := stmt.ColumnInt(2) == 1
		description := stmt.ColumnText(3)

		// create directive struct to collect data
		directive := &Directive{
			Name:        name,
			Internal:    internal,
			Repeatable:  repeatable,
			Description: description,
			Arguments:   []*Argument{},
			Locations:   []string{},
		}
		directives[name] = directive

		// collect arguments first
		argErr := db.StepQuery(ctx, `
				SELECT name, type, type_modifiers, default_value
				FROM directive_arguments
				WHERE parent = $directive
			`, map[string]any{"directive": name}, func(stmt *sqlite.Stmt) {
			arg := &Argument{
				Name:          stmt.ColumnText(0),
				Type:          stmt.ColumnText(1),
				TypeModifiers: stmt.ColumnText(2),
				DefaultValue:  stmt.ColumnText(3),
			}
			directive.Arguments = append(directive.Arguments, arg)

			// collect custom types (skip built-in GraphQL scalars)
			if !isBuiltInScalar(arg.Type) {
				customTypes[arg.Type] = true
			}
		})
		if argErr != nil {
			errs.Append(plugins.WrapError(argErr))
			return
		}

		// collect locations
		locErr := db.StepQuery(ctx, `
				SELECT location
				FROM directive_locations
				WHERE directive = $directive
			`, map[string]any{"directive": name}, func(stmt *sqlite.Stmt) {
			location := stmt.ColumnText(0)
			directive.Locations = append(directive.Locations, location)
		})
		if locErr != nil {
			errs.Append(plugins.WrapError(locErr))
			return
		}

		if description != "" && description != "null" {
			// writing the desc as a comment
			schemaString.WriteString(fmt.Sprintf("\"\"\"%s\"\"\"\n", description))
		}

		schemaString.WriteString(fmt.Sprintf("directive @%s", name))

		// arguments in parentheses
		if len(directive.Arguments) > 0 {
			schemaString.WriteString("(")
			for i, arg := range directive.Arguments {
				if i > 0 {
					schemaString.WriteString(", ")
				}
				schemaString.WriteString(fmt.Sprintf("%s: %s%s", arg.Name, arg.Type, arg.TypeModifiers))
			}
			schemaString.WriteString(")")

			// add repeatable keyword
			if repeatable {
				schemaString.WriteString(" repeatable")
			}

			// ddd locations
			schemaString.WriteString(" on ")
			schemaString.WriteString(strings.Join(directive.Locations, " | "))

		}

		schemaString.WriteString("\n\n")

	})
	if err != nil {
		return plugins.WrapError(err)
	}

	if errs.Error() != "" {
		return errs
	}

	// writing enum definitions for custom types referenced by directive arguments at the end of the file
	// writing at the end of the file(schema.graphql) cost us one more loop but it is cleaner
	for typeName := range customTypes {
		enumValues := []string{}

		// query enum values for this type
		enumErr := db.StepQuery(ctx, `
			SELECT value
			FROM enum_values
			WHERE parent = $typeName
			ORDER BY value
		`, map[string]any{"typeName": typeName}, func(stmt *sqlite.Stmt) {
			value := stmt.ColumnText(0)
			enumValues = append(enumValues, value)
		})

		if enumErr != nil {
			errs.Append(plugins.WrapError(enumErr))
			continue
		}

		//if we found enum values, write the enum definition
		if len(enumValues) > 0 {
			schemaString.WriteString(fmt.Sprintf("enum %s {\n", typeName))
			for _, value := range enumValues {
				schemaString.WriteString(fmt.Sprintf("  %s\n", value))
			}
			schemaString.WriteString("}\n\n")
		}
	}

	schemaFileLocation := projectConfig.DefinitionsSchemaPath()
	if schemaFileLocation == "" {
		return fmt.Errorf("schema file location not found in project config")
	}

	// Ensure the directory exists before writing the file
	dir := filepath.Dir(schemaFileLocation)
	err = fs.MkdirAll(dir, 0755)
	if err != nil {
		return plugins.WrapError(err)
	}

	err = afero.WriteFile(fs, schemaFileLocation, []byte(schemaString.String()), 0644)
	if err != nil {
		return plugins.WrapError(err)
	}
	return nil

}

func generateDocumentsFile(ctx context.Context, db plugins.DatabasePool[config.PluginConfig], fs afero.Fs, projectConfig plugins.ProjectConfig) error {

	// get all documents from docuemnt table joined with discovered list and that are fragments
	var documentString strings.Builder

	err := db.StepQuery(ctx, `
		SELECT d.printed
		FROM discovered_lists dl
		JOIN documents d ON dl.raw_document = d.raw_document
		WHERE d.kind = 'fragment'
	`, nil, func(stmt *sqlite.Stmt) {
		printed := stmt.ColumnText(0)
		documentString.WriteString(printed)
		documentString.WriteString("\n\n")
	})

	if err != nil {
		return plugins.WrapError(err)
	}

	documentsFileLocation := projectConfig.DefinitionsDocumentsPath()
	if documentsFileLocation == "" {
		return fmt.Errorf("documents file location not found in project config")
	}

	// Ensure the directory exists before writing the file
	dir := filepath.Dir(documentsFileLocation)
	err = fs.MkdirAll(dir, 0755)
	if err != nil {
		return plugins.WrapError(err)
	}

	err = afero.WriteFile(fs, documentsFileLocation, []byte(documentString.String()), 0644)
	if err != nil {
		return plugins.WrapError(err)
	}
	return nil
}

func generateEnumFiles(ctx context.Context, db plugins.DatabasePool[config.PluginConfig], fs afero.Fs, projectConfig plugins.ProjectConfig) error {
	var enumString strings.Builder

	err := db.StepQuery(ctx, `
		SELECT t.name,
		       'export const ' || t.name || ' = {' || char(10) ||
		       '    ' || GROUP_CONCAT('"' || ev.value || '": "' || ev.value || '"', ',' || char(10) || '    ' ORDER BY ev.value) ||
		       char(10) || '};' || char(10) || char(10) as enum_definition
		FROM types t
		JOIN enum_values ev ON ev.parent = t.name
		WHERE t.built_in = 0
		GROUP BY t.name
		ORDER BY t.name
	`, nil, func(stmt *sqlite.Stmt) {
		enumDefinition := stmt.ColumnText(1)
		enumString.WriteString(enumDefinition)
	})

	if err != nil {
		return plugins.WrapError(err)
	}

	enumsFileLocation := projectConfig.DefinitionsEnumRuntime()
	if enumsFileLocation == "" {
		return fmt.Errorf("enums file location not found in project config")
	}

	dir := filepath.Dir(enumsFileLocation)
	err = fs.MkdirAll(dir, 0755)
	if err != nil {
		return plugins.WrapError(err)
	}

	err = afero.WriteFile(fs, enumsFileLocation, []byte(enumString.String()), 0644)
	if err != nil {
		return plugins.WrapError(err)
	}

	return nil
}

// helper functions
func isBuiltInScalar(typeName string) bool {
	builtInScalars := map[string]bool{
		"String":  true,
		"Boolean": true,
		"Int":     true,
		"Float":   true,
		"ID":      true,
	}
	return builtInScalars[typeName]
}
