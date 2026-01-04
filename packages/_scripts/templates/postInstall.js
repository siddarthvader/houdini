const fs = require('fs')
const path = require('path')
const zlib = require('zlib')
const https = require('https')

// Adjust the version you want to install. You can also make this dynamic.
const BINARY_DISTRIBUTION_VERSION = 'package-version'

// Windows binaries end with .exe so we need to special case them.
const binaryName = process.platform === 'win32' ? 'my-binary.exe' : 'my-binary'

// Determine package name for this platform
const platformSpecificPackageName = `my-package-${process.platform}-${process.arch}`

// Compute the path we want to emit the fallback binary to
const fallbackBinaryPath = path.join(__dirname, binaryName)

function makeRequest(url) {
	return new Promise((resolve, reject) => {
		https
			.get(url, (response) => {
				if (response.statusCode >= 200 && response.statusCode < 300) {
					const chunks = []
					response.on('data', (chunk) => chunks.push(chunk))
					response.on('end', () => {
						resolve(Buffer.concat(chunks))
					})
				} else if (
					response.statusCode >= 300 &&
					response.statusCode < 400 &&
					response.headers.location
				) {
					// Follow redirects
					makeRequest(response.headers.location).then(resolve, reject)
				} else {
					reject(
						new Error(
							`npm responded with status code ${response.statusCode} when downloading the package!`
						)
					)
				}
			})
			.on('error', (error) => {
				reject(error)
			})
	})
}

function extractFileFromTarball(tarballBuffer, filepath) {
	// Tar archives are organized in 512 byte blocks.
	// Blocks can either be header blocks or data blocks.
	// Header blocks contain file names of the archive in the first 100 bytes, terminated by a null byte.
	// The size of a file is contained in bytes 124-135 of a header block and in octal format.
	// The following blocks will be data blocks containing the file.
	let offset = 0
	while (offset < tarballBuffer.length) {
		const header = tarballBuffer.subarray(offset, offset + 512)
		offset += 512

		const fileName = header.toString('utf-8', 0, 100).replace(/\0.*/g, '')
		const fileSize = parseInt(header.toString('utf-8', 124, 136).replace(/\0.*/g, ''), 8)

		if (fileName === filepath) {
			return tarballBuffer.subarray(offset, offset + fileSize)
		}

		// Clamp offset to the uppoer multiple of 512
		offset = (offset + fileSize + 511) & ~511
	}
}

async function downloadBinaryFromNpm() {
	// Download the tarball of the right binary distribution package
	const tarballDownloadBuffer = await makeRequest(
		`https://registry.npmjs.org/${platformSpecificPackageName}/-/${platformSpecificPackageName}-${BINARY_DISTRIBUTION_VERSION}.tgz`
	)

	const tarballBuffer = zlib.unzipSync(tarballDownloadBuffer)

	// Extract binary from package and write to disk
	fs.writeFileSync(
		fallbackBinaryPath,
		extractFileFromTarball(tarballBuffer, `package/bin/${binaryName}`),
		{ mode: 0o755 } // Make binary file executable
	)
}

function isPlatformSpecificPackageInstalled() {
	try {
		// Try to resolve the platform package itself
		require.resolve(`${platformSpecificPackageName}/package.json`)
		return true
	} catch (e) {
		// Also check if it exists as a sibling directory
		const siblingPath = path.join(__dirname, '..', platformSpecificPackageName)
		const siblingBinaryPath = path.join(siblingPath, 'bin', binaryName)
		return fs.existsSync(siblingBinaryPath)
	}
}

if (!platformSpecificPackageName) {
	throw new Error('Platform not supported!')
}

// once we've confirmed the required package is installed we want to overwrite the bin entry of our package.json
// to point to the correct binary for optimal performance (skip Node.js overhead)
function overwriteBinary() {
	const packageJsonPath = path.join(__dirname, 'package.json')
	const packageJson = require(packageJsonPath)

	let binaryPath = null

	try {
		// Method 1: Use require.resolve to find the actual path to the platform-specific package
		// This works with pnpm and other package managers that use symlinks
		const platformPackagePath = require.resolve(`${platformSpecificPackageName}/package.json`)
		const platformPackageDir = path.dirname(platformPackagePath)
		binaryPath = path.join(platformPackageDir, 'bin', binaryName)
	} catch (error) {
		// Method 2: Check if platform package is installed as a sibling directory
		// This works with npm and local installs
		const siblingPath = path.join(__dirname, '..', platformSpecificPackageName)
		const siblingBinaryPath = path.join(siblingPath, 'bin', binaryName)

		if (fs.existsSync(siblingBinaryPath)) {
			binaryPath = siblingBinaryPath
		} else {
			// Use shim as fallback
			return
		}
	}

	if (binaryPath) {
		// Make the path relative to the main package directory
		const relativeBinaryPath = path.relative(__dirname, binaryPath)
		packageJson.bin = relativeBinaryPath

		fs.writeFileSync(packageJsonPath, JSON.stringify(packageJson, null, 2))
	}
}

// Skip downloading the binary if it was already installed via optionalDependencies
if (!isPlatformSpecificPackageInstalled()) {
	downloadBinaryFromNpm().then(overwriteBinary)
} else {
	overwriteBinary()
}
