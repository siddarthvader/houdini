#!/usr/bin/env node

// Simple shim that can be replaced with the actual binary for optimal performance
// This follows esbuild's approach: the postInstall script will replace this entire file
// with the native binary when possible, eliminating Node.js startup overhead

const fs = require('fs');
const path = require('path');
const { execFileSync } = require('child_process');

// Manual binary path override support
const MANUAL_BINARY_PATH = process.env.HOUDINI_BINARY_PATH || process.env.MY_PACKAGE_BINARY_PATH;

function getBinaryPath() {
	// Check for manual override first
	if (MANUAL_BINARY_PATH && fs.existsSync(MANUAL_BINARY_PATH)) {
		return MANUAL_BINARY_PATH;
	}

	// Platform-specific package lookup
	const BINARY_DISTRIBUTION_PACKAGES = {
		'linux-x64': 'my-package-linux-x64',
		'linux-arm64': 'my-package-linux-arm64',
		'win32-x64': 'my-package-win32-x64',
		'win32-arm64': 'my-package-win32-arm64',
		'darwin-x64': 'my-package-darwin-x64',
		'darwin-arm64': 'my-package-darwin-arm64',
	}

	const binaryName = process.platform === 'win32' ? 'my-binary.exe' : 'my-binary'
	const platformSpecificPackageName = BINARY_DISTRIBUTION_PACKAGES[`${process.platform}-${process.arch}`]

	if (!platformSpecificPackageName) {
		// Fallback to downloaded binary if platform not supported
		return path.join(__dirname, binaryName)
	}

	try {
		// Method 1: Use require.resolve to find the platform-specific package
		const platformPackagePath = require.resolve(`${platformSpecificPackageName}/package.json`)
		const platformPackageDir = path.dirname(platformPackagePath)
		return path.join(platformPackageDir, 'bin', binaryName)
	} catch (error) {
		// Method 2: Check sibling directory (npm structure)
		const siblingPath = path.join(__dirname, '..', platformSpecificPackageName)
		const siblingBinaryPath = path.join(siblingPath, 'bin', binaryName)

		if (fs.existsSync(siblingBinaryPath)) {
			return siblingBinaryPath
		}

		// Method 3: Check pnpm structure
		const pnpmMatch = __dirname.match(/(.+\/node_modules\/)\.pnpm\/([^\/]+)\/node_modules\//)
		if (pnpmMatch) {
			const [, nodeModulesRoot] = pnpmMatch
			const pnpmDir = path.join(nodeModulesRoot, '.pnpm')

			try {
				const pnpmEntries = fs.readdirSync(pnpmDir)
				// Get the expected version from the main package
				const packageJSON = require(path.join(__dirname, '..', 'package.json'))
				const expectedVersion = packageJSON.version
				const expectedPnpmEntry = `${platformSpecificPackageName}@${expectedVersion}`
				const platformEntry = pnpmEntries.find(entry => entry === expectedPnpmEntry)

				if (platformEntry) {
					const pnpmBinaryPath = path.join(pnpmDir, platformEntry, 'node_modules', platformSpecificPackageName, 'bin', binaryName)
					if (fs.existsSync(pnpmBinaryPath)) {
						return pnpmBinaryPath
					}
				}
			} catch (err) {
				// Ignore pnpm detection errors
			}
		}

		// Method 4: Fallback to downloaded binary in main package
		return path.join(__dirname, binaryName)
	}
}

// Execute the binary directly (this entire file may be replaced with the actual binary)
try {
	execFileSync(getBinaryPath(), process.argv.slice(2), { stdio: 'inherit' });
} catch (error) {
	// If execFileSync fails, exit with the same code
	process.exit(error.status || 1);
}
