package schema_test

import (
	"context"
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
				Name: "Generates runtime definitions for each enum",
				Input: []string{
					`query GetAppVersion { version }`,
				},
				Extra: map[string]any{
					"enumsTypesContains": []string{
						"type ValuesOf<T> = T[keyof T]",
						`export declare const BookingStatus: {
    readonly CANCELLED: "CANCELLED";
    readonly CAPTURING_PAYMENT: "CAPTURING_PAYMENT";
    readonly CONFIRMED: "CONFIRMED";
    readonly CREATING_ASSETS: "CREATING_ASSETS";
    readonly PAYMENT_PENDING: "PAYMENT_PENDING";
}

export type BookingStatus$options = ValuesOf<typeof BookingStatus>`,
						`export declare const GlobalSearchResultType: {
    readonly ACCOMMODATION: "ACCOMMODATION";
    readonly RESTAURANT: "RESTAURANT";
    readonly USER: "USER";
}

export type GlobalSearchResultType$options = ValuesOf<typeof GlobalSearchResultType>`,
					},
					"enumsContains": []string{
						`export const BookingStatus = {
    "CANCELLED": "CANCELLED",
    "CAPTURING_PAYMENT": "CAPTURING_PAYMENT",
    "CONFIRMED": "CONFIRMED",
    "CREATING_ASSETS": "CREATING_ASSETS",
    "PAYMENT_PENDING": "PAYMENT_PENDING"
};`,
						`export const GlobalSearchResultType = {
    "ACCOMMODATION": "ACCOMMODATION",
    "RESTAURANT": "RESTAURANT",
    "USER": "USER"
};`,
					},
				},
			},
			{
				Name: "Generates schema.graphql with internal directives",
				Input: []string{
					`query GetAllUsers { allUsers @list(name: "All_Users") { id name email } }`,
					`fragment UserBasicInfo on User @list(name: "User_List") { id name email userType }`,
				},
				Extra: map[string]any{
					"schemaContains": []string{
						"directive @list",
						"directive @paginate",
						"directive @prepend",
						"directive @append",
						"directive @allLists",
						"enum CachePolicy",
						"enum PaginateMode",
						"enum DedupeMatchMode",
						"CacheAndNetwork",
						"Infinite",
						"Variables",
					},
				},
			},
			{
				Name: "Generates documents.gql with list fragments",
				Input: []string{
					`query GetUserConnections { usersByCursor @list(name: "Friends") { edges { node { id name email } } } }`,
					`fragment UserProfile on User { firstName email userType }`,
				},
				Extra: map[string]any{
					"documentsContains": []string{
						"fragment Friends_insert on User",
						"fragment Friends_toggle on User",
						"fragment Friends_remove on User",
					},
				},
			},
			{
				Name: "Generates documents.gql with custom ID list fragments",
				Input: []string{
					`query GetUserConnections { usersByCursor @list(name: "Friends") { edges { node { id name } } } }`,
					`fragment UserProfile on User { firstName email }`,
					`query GetAllBookings { bookings @list(name: "AllBookings") { id status } }`,
				},
				Extra: map[string]any{
					"documentsContains": []string{
						"fragment Friends_insert on User",
						"fragment Friends_toggle on User",
						"fragment Friends_remove on User",
						"fragment AllBookings_insert on Booking",
						"fragment AllBookings_toggle on Booking",
						"fragment AllBookings_remove on Booking",
					},
				},
			},
			{
				Name: "Writing twice doesn't duplicate definitions",
				Input: []string{
					`query GetAppVersion { version }`,
					`fragment UserProfile on User { firstName email }`,
				},
				Extra: map[string]any{
					"runGenerationTwice": true,
					"listDirectiveCount": 1,
				},
			},
			{
				Name: "Generates enums.js with correct format",
				Input: []string{
					`query GetAppVersion { version }`,
				},
				Extra: map[string]any{
					"enumsContains": []string{
						"export const DedupeMatchMode = {",
						`"Variables": "Variables"`,
						`"Operation": "Operation"`,
						`"None": "None"`,
						"};",
					},
					"enumsTypesContains": []string{
						"type ValuesOf<T> = T[keyof T]",
						"export declare const DedupeMatchMode: {",
						`readonly Variables: "Variables";`,
						`readonly Operation: "Operation";`,
						`readonly None: "None";`,
						"export type DedupeMatchMode$options = ValuesOf<typeof DedupeMatchMode>",
					},
				},
			},
			{
				Name: "Generates enums.d.ts with TypeScript definitions",
				Input: []string{
					`query GetAppVersion { version }`,
				},
				Extra: map[string]any{
					"enumsTypesContains": []string{
						"type ValuesOf<T> = T[keyof T]",
						"export declare const DedupeMatchMode: {",
						`readonly Variables: "Variables";`,
						`readonly Operation: "Operation";`,
						`readonly None: "None";`,
						"}",
						"export type DedupeMatchMode$options = ValuesOf<typeof DedupeMatchMode>",
					},
					"enumsTypesNotContains": []string{
						"export const",
						"};",
					},
					"indexJsContains": []string{
						"export * from './enums.js'",
					},
					"indexDtsContains": []string{
						"export * from './enums.js'",
					},
				},
			},
		},
		Schema: `
			type Query {
				allUsers: [User!]!
				usersByCursor: UserConnection!
				bookings: [Booking!]!
				places: [Place!]!
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
				userType: UserType!
			}

			type Place {
				id: ID!
				name: String!
				address: String!
			}

			type Booking {
				id: ID!
				status: BookingStatus!
				place: Place!
				user: User!
			}

			enum UserType {
				GUEST
				HOST
				ADMIN
			}

			enum BookingStatus {
				CANCELLED
				CAPTURING_PAYMENT
				CONFIRMED
				CREATING_ASSETS
				PAYMENT_PENDING
			}

			enum GlobalSearchResultType {
				ACCOMMODATION
				RESTAURANT
				USER
			}
    `,
		PerformTest: performDefinitionsTest,
	})
}

func performDefinitionsTest(t *testing.T, p *plugin.HoudiniCore, test tests.Test[config.PluginConfig]) {
	err := runFullGeneration(context.Background(), p)
	if err != nil {
		t.Logf("runFullGeneration error: %v", err)
	}
	require.Nil(t, err)

	if runTwice, ok := test.Extra["runGenerationTwice"].(bool); ok && runTwice {
		err = schema.GenerateDefinitionFiles(context.Background(), p.DB, p.Fs, false)
		if err != nil {
			t.Logf("Second generateDefinitions call failed with errors: %v", err)
			t.Fatalf("Second generateDefinitions call should not fail")
		}
	}

	projectConfig, err := p.DB.ProjectConfig(context.Background())
	require.Nil(t, err)

	// test.Extra contains the expected data for each test
	// if the data is nil then we bypass the assertion
	checkFileContains(t, p.Fs, projectConfig.DefinitionsSchemaPath(), test.Extra["schemaContains"])
	checkFileContains(t, p.Fs, projectConfig.DefinitionsDocumentsPath(), test.Extra["documentsContains"])
	checkFileContains(t, p.Fs, projectConfig.DefinitionsEnumRuntime(), test.Extra["enumsContains"])
	checkFileContains(t, p.Fs, projectConfig.DefinitionsEnumTypes(), test.Extra["enumsTypesContains"])
	// only test  that contains enumsTypesNotContains and for other tests it will bypass
	checkFileNotContains(t, p.Fs, projectConfig.DefinitionsEnumTypes(), test.Extra["enumsTypesNotContains"])
	checkFileContains(t, p.Fs, projectConfig.DefinitionsIndexJs(), test.Extra["indexJsContains"])
	checkFileContains(t, p.Fs, projectConfig.DefinitionsIndexDts(), test.Extra["indexDtsContains"])
	checkDirectiveCount(t, p.Fs, projectConfig.DefinitionsSchemaPath(), test.Extra["listDirectiveCount"])
}

func runFullGeneration(ctx context.Context, p *plugin.HoudiniCore) error {
	err := p.AfterExtract(ctx)
	if err != nil {
		return err
	}

	err = p.Validate(ctx)
	if err != nil {
		return err
	}

	err = p.AfterValidate(ctx)
	if err != nil {
		return err
	}

	projectConfig, err := p.DB.ProjectConfig(ctx)
	if err != nil {
		return err
	}

	originalPath := projectConfig.PersistedQueriesPath
	projectConfig.PersistedQueriesPath = "./dummy-queries.json"
	p.DB.SetProjectConfig(projectConfig)

	_, err = p.Generate(ctx)

	projectConfig.PersistedQueriesPath = originalPath
	p.DB.SetProjectConfig(projectConfig)

	return err
}

// test helpers
func checkFileContains(t *testing.T, fs afero.Fs, path string, data any) {
	// if data is nil then we dont do any assertions, basic bypass
	if data == nil {
		return
	}

	checks, ok := data.([]string)
	if !ok {
		return
	}

	content, err := afero.ReadFile(fs, path)
	require.Nil(t, err)
	str := string(content)
	for _, expected := range checks {
		assert.Contains(t, str, expected)
	}
}

func checkFileNotContains(t *testing.T, fs afero.Fs, path string, data any) {
	if data == nil {
		return
	}

	checks, ok := data.([]string)
	if !ok {
		return
	}

	content, err := afero.ReadFile(fs, path)
	require.Nil(t, err)
	str := string(content)
	for _, unexpected := range checks {
		assert.NotContains(t, str, unexpected)
	}
}

func checkDirectiveCount(t *testing.T, fs afero.Fs, path string, data any) {
	if data == nil {
		return
	}

	expectedCount, ok := data.(int)
	if !ok {
		return
	}

	content, err := afero.ReadFile(fs, path)
	require.Nil(t, err)
	str := string(content)
	actualCount := strings.Count(str, "directive @list")
	assert.Equal(t, expectedCount, actualCount, "directive @list should appear exactly once")
}
