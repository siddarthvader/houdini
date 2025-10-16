import fs from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

// we need to make sure we have executables for houdini
const houdiniBinPath = 'node_modules/.bin/houdini'
const houdiniCmdPath = path.resolve(__dirname, '../../packages/houdini/build/cmd/index.js')

try {
	// Remove existing symlink if it exists
	await fs.rm(houdiniBinPath, { force: true })

	// Create new symlink to the built command
	await fs.symlink(houdiniCmdPath, houdiniBinPath, 'file')
	console.log(`✅ Created binary symlink: ${houdiniBinPath} -> ${houdiniCmdPath}`)
} catch (e) {
	console.warn(`⚠️  Failed to create binary symlink: ${e.message}`)
}

// make sure its executable
try {
	await fs.chmod(houdiniBinPath, 0o755)
	console.log(`✅ Made binary executable: ${houdiniBinPath}`)
} catch (e) {
	console.warn(`⚠️  Failed to make binary executable: ${e.message}`)
}

// create symlinks for houdini plugins to point to their built directories
const plugins = [
	{
		name: 'houdini',
		path: path.resolve(__dirname, '../../packages/houdini/build'),
	},
	{
		name: 'houdini-react',
		path: path.resolve(__dirname, '../../packages/houdini-react/build/houdini-react'),
	},
	{
		name: 'houdini-core',
		path: path.resolve(__dirname, '../../packages/houdini-core/build/houdini-core'),
	},
]

for (const plugin of plugins) {
	try {
		// remove existing symlink/directory if it exists
		await fs.rm(`node_modules/${plugin.name}`, { recursive: true, force: true })

		// create symlink to the built plugin directory
		await fs.symlink(plugin.path, `node_modules/${plugin.name}`, 'dir')
		console.log(`✅ Created symlink: node_modules/${plugin.name} -> ${plugin.path}`)
	} catch (e) {
		console.warn(`Failed to create symlink for ${plugin.name}:`, e.message)
	}
}
