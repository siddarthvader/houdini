import type { DirectiveNode, ObjectValueNode } from 'graphql'

import type { FragmentArgument } from '../types'

/**
 * Parse @arguments directive to extract fragment arguments
 *
 * Input: @arguments(width: { type: "Int!", default: 50 }, rounded: { type: "Boolean" })
 * Output: [
 *   { name: "width", type: "Int", required: true, default: 50 },
 *   { name: "rounded", type: "Boolean", required: false }
 * ]
 */
export function parseArgumentsDirective(directive: DirectiveNode): FragmentArgument[] {
	console.log('[ArgumentsParser] Parsing @arguments directive');
	const args: FragmentArgument[] = []

	if (!directive.arguments) {
		console.log('[ArgumentsParser] No arguments in directive');
		return args
	}

	console.log(`[ArgumentsParser] Found ${directive.arguments.length} directive arguments`);

	for (const arg of directive.arguments) {
		// arg.name.value = "width"
		// arg.value = { kind: "ObjectValue", fields: [...] }

		if (arg.value.kind !== 'ObjectValue') {
			console.log(`[ArgumentsParser] Skipping non-object argument: ${arg.name.value}`);
			continue
		}

		const config = parseArgumentConfig(arg.value)

		args.push({
			name: arg.name.value,
			type: config.type.replace('!', ''),
			required: config.type.endsWith('!'),
			default: config.default,
			location: {
				line: arg.loc?.startToken.line ?? 0,
				column: arg.loc?.startToken.column ?? 0,
			},
		})

		console.log(`[ArgumentsParser] Parsed argument: ${arg.name.value} (${config.type}${config.default !== undefined ? `, default: ${config.default}` : ''})`);
	}

	console.log(`[ArgumentsParser] Total parsed: ${args.length} arguments`);
	return args
}

/**
 * Parse the argument config object: { type: "Int!", default: 50 }
 */
function parseArgumentConfig(objectValue: ObjectValueNode): {
	type: string
	default?: any
} {
	let type = ''
	let defaultValue: any = undefined

	for (const field of objectValue.fields) {
		if (field.name.value === 'type') {
			// Extract type string
			if (field.value.kind === 'StringValue') {
				type = field.value.value
			}
		} else if (field.name.value === 'default') {
			// Extract default value
			defaultValue = extractValue(field.value)
		}
	}

	return { type, default: defaultValue }
}

/**
 * Extract primitive value from GraphQL AST node
 */
function extractValue(valueNode: any): any {
	switch (valueNode.kind) {
		case 'IntValue':
			return parseInt(valueNode.value, 10)
		case 'FloatValue':
			return parseFloat(valueNode.value)
		case 'StringValue':
			return valueNode.value
		case 'BooleanValue':
			return valueNode.value
		case 'NullValue':
			return null
		default:
			return undefined
	}
}
