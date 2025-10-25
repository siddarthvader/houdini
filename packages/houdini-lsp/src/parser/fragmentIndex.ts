import { type FragmentDefinitionNode, parse } from 'graphql'
import type { Connection } from 'vscode-languageserver'
import type { FragmentDefinition, FragmentIndex } from '../types'
import { parseArgumentsDirective } from './argumentsParser'

export class FragmentIndexImpl implements FragmentIndex {
	public fragments = new Map<string, FragmentDefinition>()

	constructor(private connection?: Connection) {}

	/**
	 * Parse a document and index all fragments
	 */
	public indexDocument(uri: string, content: string): void {
		// Only index .graphql files
		if (!uri.endsWith('.graphql') && !uri.endsWith('.gql')) {
			this.connection?.console.error(`[FragmentIndex] Skipping non-GraphQL file: ${uri}`)
			return
		}

		try {
			const document = parse(content, { noLocation: false })

			// Only remove and re-add if parsing succeeds
			this.removeByUri(uri)

			// Find all fragment definitions
			let fragmentCount = 0
			for (const definition of document.definitions) {
				if (definition.kind === 'FragmentDefinition') {
					const fragment = this.parseFragment(definition, uri)
					if (fragment) {
						this.set(fragment)
						fragmentCount++
						this.connection?.console.error(`[FragmentIndex] Indexed fragment: ${fragment.name} with ${fragment.arguments.length} arguments`)
					}
				}
			}
			this.connection?.console.error(`[FragmentIndex] Total fragments indexed: ${fragmentCount}`)
			this.connection?.console.error(`[FragmentIndex] All fragments in index: ${Array.from(this.fragments.keys()).join(', ')}`)
		} catch (error) {
			// Parse error - ignore for now, will be caught by diagnostics
			this.connection?.console.error(`[FragmentIndex] Failed to parse document ${uri}: ${error}`)
		}
	}

	/**
	 * Parse a fragment definition node
	 */
	private parseFragment(
		node: FragmentDefinitionNode,
		uri: string,
	): FragmentDefinition | null {
		// Find @arguments directive
		const argumentsDirective = node.directives?.find(
			(d) => d.name.value === 'arguments',
		)

		const args = argumentsDirective
			? parseArgumentsDirective(argumentsDirective)
			: []

		return {
			name: node.name.value,
			typeCondition: node.typeCondition.name.value,
			arguments: args,
			uri,
			location: {
				line: node.loc?.startToken.line ?? 0,
				column: node.loc?.startToken.column ?? 0,
			},
		}
	}

	public set(fragment: FragmentDefinition): void {
		this.fragments.set(fragment.name, fragment)
	}

	public get(name: string): FragmentDefinition | undefined {
		return this.fragments.get(name)
	}

	public removeByUri(uri: string): void {
		for (const [name, fragment] of this.fragments.entries()) {
			if (fragment.uri === uri) {
				this.fragments.delete(name)
			}
		}
	}

	public getAll(): FragmentDefinition[] {
		return Array.from(this.fragments.values())
	}
}
