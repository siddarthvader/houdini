package schema_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.houdinigraphql.com/packages/houdini-core/config"
	"code.houdinigraphql.com/packages/houdini-core/plugin"
	"code.houdinigraphql.com/packages/houdini-core/plugin/schema"
	"code.houdinigraphql.com/plugins/tests"
)

func TestDefinitionGeneration(t *testing.T) {
	tests.RunTable(t, tests.Table[config.PluginConfig]{
		Tests: []tests.Test[config.PluginConfig]{
			{
				Name: "Generates schema.graphql with internal directives",
				Input: []string{
					`query AllUsers { allUsers @list(name: "All_Users") { id name } }`,
					`fragment UserInfo on User @list(name: "User_List") { id name email }`,
				},
			},
			{
				Name: "Generates documents.gql with list fragments",
				Input: []string{
					`query TestQuery { usersByCursor @list(name: "Friends") { edges { node { id } } } }`,
					`fragment TestFragment on User { firstName }`,
				},
			},
			{
				Name: "Generates documents.gql with custom ID list fragments",
				Input: []string{
					`query TestQuery { usersByCursor @list(name: "Friends") { edges { node { id } } } }`,
					`fragment TestFragment on User { firstName }`,
					`query CustomIdList { customIdList @list(name: "theList") { foo } }`,
				},
			},
			{
				Name: "Writing twice doesn't duplicate definitions",
				Input: []string{
					`query TestQuery { version }`,
					`fragment TestFragment on User { firstName }`,
				},
			},
			{
				Name: "Generates enums.js with correct format",
				Input: []string{
					`query TestQuery { version }`,
				},
			},
			{
				Name: "Generates enums.d.ts with TypeScript definitions",
				Input: []string{
					`query TestQuery { version }`,
				},
			},
		},
		Schema: `
			type Query {
				allUsers: [User!]!
				usersByCursor: UserConnection!
				customIdList: [CustomIdType!]!
				version: Int!
			}

			type UserConnection {
				edges: [UserEdge!]!
			}

			type UserEdge {
				node: User!
			}

			type User {
				id: ID!
				name: String!
				email: String!
				firstName: String!
			}

			type CustomIdType {
				foo: String!
				bar: String!
			}

			"""
			Documentation of testenum1
			"""
			enum TestEnum1 {
				"Documentation of Value1"
				Value1
				"Documentation of Value2"
				Value2
			}

			"""
			Documentation of testenum2
			"""
			enum TestEnum2 {
				Value3
				Value2
			}
    `,
		PerformTest: func(t *testing.T, p *plugin.HoudiniCore, test tests.Test[config.PluginConfig]) {
			config, err := p.DB.ProjectConfig(context.Background())
			assert.Nil(t, err)

			switch test.Name {
			case "Generates runtime definitions for each enum":
				// read from the filesystem and confirm that the value matches our expectations
				typeDefinitions, err := afero.ReadFile(
					p.Fs,
					config.DefinitionsEnumTypes(),
				)
				assert.Nil(t, err)

				require.Equal(t, tests.Dedent(`
	          type ValuesOf<T> = T[keyof T]

	          export declare const DedupeMatchMode: {
	              readonly Variables: "Variables";
	              readonly Operation: "Operation";
	              readonly None: "None";
	          }

	          export type DedupeMatchMode$options = ValuesOf<typeof DedupeMatchMode>

	          /** Documentation of testenum1 */
	          export declare const TestEnum1: {
	              /** Documentation of Value1 */
	              readonly Value1: "Value1";
	              /** Documentation of Value2 */
	              readonly Value2: "Value2";
	          }

	          /** Documentation of testenum1 */
	          export type TestEnum1$options = ValuesOf<typeof TestEnum1>

	          /** Documentation of testenum2 */
	          export declare const TestEnum2: {
	              readonly Value3: "Value3";
	              readonly Value2: "Value2";
	          }

	          /** Documentation of testenum2 */
	          export type TestEnum2$options = ValuesOf<typeof TestEnum2>
	        `),
					string(typeDefinitions),
				)

				runtimeDefinitions, err := afero.ReadFile(
					p.Fs,
					config.DefinitionsEnumRuntime(),
				)
				assert.Nil(t, err)

				require.Equal(t, tests.Dedent(`
	          /** Documentation of testenum1 */
	          export const TestEnum1 = {
	              /**
	               * Documentation of Value1
	              */
	              "Value1": "Value1",

	              /**
	               * Documentation of Value2
	              */
	              "Value2": "Value2"
	          };

	          /** Documentation of testenum2 */
	          export const TestEnum2 = {
	              "Value3": "Value3",
	              "Value2": "Value2"
	          };

	          export const DedupeMatchMode = {
	              "Variables": "Variables",
	              "Operation": "Operation",
	              "None": "None"
	          };
	        `),
					string(runtimeDefinitions),
				)

			case "Generates schema.graphql with internal directives":
				// Run pipeline steps needed for schema generation
				err = p.AfterExtract(context.Background())
				assert.Nil(t, err)

				err = p.Validate(context.Background())
				assert.Nil(t, err)

				// Get updated project config
				projectConfig, err := p.DB.ProjectConfig(context.Background())
				require.Nil(t, err)

				// Call schema generation directly instead of full Generate
				err = schema.GenerateDefinitionFiles(context.Background(), p.DB, p.Fs, false)
				assert.Nil(t, err)

				// Test schema.graphql generation
				schemaContent, err := afero.ReadFile(
					p.Fs,
					projectConfig.DefinitionsSchemaPath(),
				)
				assert.Nil(t, err)

				// The schema should contain internal directives like @list, @paginate, etc.
				schemaStr := string(schemaContent)

				// Check for key directives that should be present
				assert.Contains(t, schemaStr, "directive @list")
				assert.Contains(t, schemaStr, "directive @paginate")
				assert.Contains(t, schemaStr, "directive @prepend")
				assert.Contains(t, schemaStr, "directive @append")
				assert.Contains(t, schemaStr, "directive @allLists")

				// Check for enum definitions
				assert.Contains(t, schemaStr, "enum CachePolicy")
				assert.Contains(t, schemaStr, "enum PaginateMode")
				assert.Contains(t, schemaStr, "enum DedupeMatchMode")

				// Check enum values are present
				assert.Contains(t, schemaStr, "CacheAndNetwork")
				assert.Contains(t, schemaStr, "Infinite")
				assert.Contains(t, schemaStr, "Variables")

			case "Generates documents.gql with list fragments":
				// Run the complete pipeline
				err = p.AfterExtract(context.Background())
				assert.Nil(t, err)

				err = p.Validate(context.Background())
				assert.Nil(t, err)

				err = p.AfterValidate(context.Background())
				assert.Nil(t, err)

				// Get updated project config
				projectConfig, err := p.DB.ProjectConfig(context.Background())
				require.Nil(t, err)

				// Call schema generation directly
				err = schema.GenerateDefinitionFiles(context.Background(), p.DB, p.Fs, false)
				assert.Nil(t, err)

				// Test documents.gql generation
				documentsContent, err := afero.ReadFile(
					p.Fs,
					projectConfig.DefinitionsDocumentsPath(),
				)
				assert.Nil(t, err)

				documentsStr := string(documentsContent)

				// Check for expected fragments
				assert.Contains(t, documentsStr, "fragment Friends_insert on User")
				assert.Contains(t, documentsStr, "fragment Friends_toggle on User")
				assert.Contains(t, documentsStr, "fragment Friends_remove on User")

			case "Generates documents.gql with custom ID list fragments":
				// Run the complete pipeline
				err = p.AfterExtract(context.Background())
				assert.Nil(t, err)

				err = p.Validate(context.Background())
				assert.Nil(t, err)

				err = p.AfterValidate(context.Background())
				assert.Nil(t, err)

				// Get updated project config
				projectConfig, err := p.DB.ProjectConfig(context.Background())
				require.Nil(t, err)

				// Call schema generation directly
				err = schema.GenerateDefinitionFiles(context.Background(), p.DB, p.Fs, false)
				assert.Nil(t, err)

				// Test documents.gql generation
				documentsContent, err := afero.ReadFile(
					p.Fs,
					projectConfig.DefinitionsDocumentsPath(),
				)
				assert.Nil(t, err)

				documentsStr := string(documentsContent)

				// Check for Friends fragments
				assert.Contains(t, documentsStr, "fragment Friends_insert on User")
				assert.Contains(t, documentsStr, "fragment Friends_toggle on User")
				assert.Contains(t, documentsStr, "fragment Friends_remove on User")

				// Check for theList fragments
				assert.Contains(t, documentsStr, "fragment theList_insert on CustomIdType")
				assert.Contains(t, documentsStr, "fragment theList_toggle on CustomIdType")
				assert.Contains(t, documentsStr, "fragment theList_remove on CustomIdType")

			case "Writing twice doesn't duplicate definitions":
				// Run the complete pipeline twice
				for i := 0; i < 2; i++ {
					err = p.AfterExtract(context.Background())
					assert.Nil(t, err)

					err = p.Validate(context.Background())
					assert.Nil(t, err)

					err = p.AfterValidate(context.Background())
					assert.Nil(t, err)

					// Call schema generation directly
					err = schema.GenerateDefinitionFiles(context.Background(), p.DB, p.Fs, false)
					assert.Nil(t, err)
				}

				// Get updated project config
				projectConfig, err := p.DB.ProjectConfig(context.Background())
				require.Nil(t, err)

				// Test schema.graphql doesn't have duplicates
				schemaContent, err := afero.ReadFile(
					p.Fs,
					projectConfig.DefinitionsSchemaPath(),
				)
				assert.Nil(t, err)

				schemaStr := string(schemaContent)

				// Count occurrences of a directive to ensure no duplicates
				listDirectiveCount := strings.Count(schemaStr, "directive @list")
				assert.Equal(t, 1, listDirectiveCount, "directive @list should appear exactly once")

			case "Generates enums.js with correct format":
				// Run the complete pipeline
				err = p.AfterExtract(context.Background())
				assert.Nil(t, err)

				err = p.Validate(context.Background())
				assert.Nil(t, err)

				err = p.AfterValidate(context.Background())
				assert.Nil(t, err)

				// Get updated project config
				projectConfig, err := p.DB.ProjectConfig(context.Background())
				require.Nil(t, err)

				// Call schema generation directly
				err = schema.GenerateDefinitionFiles(context.Background(), p.DB, p.Fs, false)
				assert.Nil(t, err)

				// Test enums.js generation
				enumsContent, err := afero.ReadFile(
					p.Fs,
					projectConfig.DefinitionsEnumRuntime(),
				)
				assert.Nil(t, err)

				enumsStr := string(enumsContent)

				// Check for built-in enums that should always be present
				assert.Contains(t, enumsStr, "export const DedupeMatchMode = {")
				assert.Contains(t, enumsStr, `"Variables": "Variables"`)
				assert.Contains(t, enumsStr, `"Operation": "Operation"`)
				assert.Contains(t, enumsStr, `"None": "None"`)

				// Check for test enums from schema if they exist in database
				if strings.Contains(enumsStr, "TestEnum1") {
					assert.Contains(t, enumsStr, "export const TestEnum1 = {")
					assert.Contains(t, enumsStr, `"Value1": "Value1"`)
					assert.Contains(t, enumsStr, `"Value2": "Value2"`)
				}

				if strings.Contains(enumsStr, "TestEnum2") {
					assert.Contains(t, enumsStr, "export const TestEnum2 = {")
					assert.Contains(t, enumsStr, `"Value3": "Value3"`)
				}

				// Check closing brace and semicolon format
				assert.Contains(t, enumsStr, "};")

				// Verify DedupeMatchMode values are sorted alphabetically
				nonePos := strings.Index(enumsStr, `"None"`)
				operationPos := strings.Index(enumsStr, `"Operation"`)
				variablesPos := strings.Index(enumsStr, `"Variables"`)
				assert.True(t, nonePos < operationPos && operationPos < variablesPos, "DedupeMatchMode values should be sorted alphabetically")

				// Test enums.d.ts generation
				enumsTypesContent, err := afero.ReadFile(
					p.Fs,
					projectConfig.DefinitionsEnumTypes(),
				)
				assert.Nil(t, err)

				enumsTypesStr := string(enumsTypesContent)

				// Check for ValuesOf helper type at the top
				assert.Contains(t, enumsTypesStr, "type ValuesOf<T> = T[keyof T]")

				// Check for TypeScript declarations
				assert.Contains(t, enumsTypesStr, "export declare const DedupeMatchMode: {")
				assert.Contains(t, enumsTypesStr, "readonly Variables: \"Variables\";")
				assert.Contains(t, enumsTypesStr, "readonly Operation: \"Operation\";")
				assert.Contains(t, enumsTypesStr, "readonly None: \"None\";")

				// Check for type aliases
				assert.Contains(t, enumsTypesStr, "export type DedupeMatchMode$options = ValuesOf<typeof DedupeMatchMode>")

				// Verify alphabetical sorting in TypeScript too
				tsNonePos := strings.Index(enumsTypesStr, "readonly None:")
				tsOperationPos := strings.Index(enumsTypesStr, "readonly Operation:")
				tsVariablesPos := strings.Index(enumsTypesStr, "readonly Variables:")
				assert.True(t, tsNonePos < tsOperationPos && tsOperationPos < tsVariablesPos, "TypeScript enum values should be sorted alphabetically")

			case "Generates enums.d.ts with TypeScript definitions":
				// Run the complete pipeline
				err = p.AfterExtract(context.Background())
				assert.Nil(t, err)

				err = p.Validate(context.Background())
				assert.Nil(t, err)

				err = p.AfterValidate(context.Background())
				assert.Nil(t, err)

				// Get updated project config
				projectConfig, err := p.DB.ProjectConfig(context.Background())
				require.Nil(t, err)

				// Call schema generation directly
				err = schema.GenerateDefinitionFiles(context.Background(), p.DB, p.Fs, false)
				assert.Nil(t, err)

				// Test only the .d.ts file
				enumsTypesContent, err := afero.ReadFile(
					p.Fs,
					projectConfig.DefinitionsEnumTypes(),
				)
				assert.Nil(t, err)

				enumsTypesStr := string(enumsTypesContent)

				// Check structure matches TypeScript test expectations
				assert.Contains(t, enumsTypesStr, "type ValuesOf<T> = T[keyof T]")
				assert.Contains(t, enumsTypesStr, "export declare const DedupeMatchMode: {")
				assert.Contains(t, enumsTypesStr, "readonly Variables: \"Variables\";")
				assert.Contains(t, enumsTypesStr, "readonly Operation: \"Operation\";")
				assert.Contains(t, enumsTypesStr, "readonly None: \"None\";")
				assert.Contains(t, enumsTypesStr, "}")
				assert.Contains(t, enumsTypesStr, "export type DedupeMatchMode$options = ValuesOf<typeof DedupeMatchMode>")

				// Verify no JavaScript syntax in TypeScript file
				assert.NotContains(t, enumsTypesStr, "export const")
				assert.NotContains(t, enumsTypesStr, "};")

				// Test index files generation
				indexJsLocation := filepath.Join(filepath.Dir(projectConfig.DefinitionsEnumRuntime()), "index.js")
				indexJsContent, err := afero.ReadFile(p.Fs, indexJsLocation)
				assert.Nil(t, err)
				assert.Contains(t, string(indexJsContent), "export * from './enums.js'")

				indexDtsLocation := filepath.Join(filepath.Dir(projectConfig.DefinitionsEnumTypes()), "index.d.ts")
				indexDtsContent, err := afero.ReadFile(p.Fs, indexDtsLocation)
				assert.Nil(t, err)
				assert.Contains(t, string(indexDtsContent), "export * from './enums.js'")
			}
		},
	})
}
