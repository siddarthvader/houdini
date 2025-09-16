package schema_test

import (
	"context"
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
				Name: "Generates runtime definitions for each enum",
			},
			{
				Name: "Generates schema.graphql with internal directives",
				Input: []string{
					`query AllUsers { allUsers @list(name: "All_Users") { id name } }`,
					`fragment UserInfo on User @list(name: "User_List") { id name email }`,
				},
			},
		},
		Schema: `
			type Query {
				allUsers: [User!]!
				version: Int!
			}

			type User {
				id: ID!
				name: String!
				email: String!
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
			}
		},
	})
}
