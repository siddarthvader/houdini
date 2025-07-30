package schema_test

import (
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.houdinigraphql.com/packages/houdini-core/config"
	"code.houdinigraphql.com/packages/houdini-core/plugin"
	"code.houdinigraphql.com/plugins/tests"
)

func TestDefinitionGeneration(t *testing.T) {
	tests.RunTable(t, tests.Table[config.PluginConfig]{
		Tests: []tests.Test[config.PluginConfig]{
			{
				Name: "Generates runtime definitions for each enum",
			},
		},
		Schema: `
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
			// read from the filesystem and confirm that the value matches our expectations
			config, err := p.DB.ProjectConfig(context.Background())
			assert.Nil(t, err)

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
		},
	})
}
