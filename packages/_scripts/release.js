#!/usr/bin/env node

import { execSync } from 'child_process';
import { readFileSync, readdirSync, existsSync } from 'fs';
import { join, basename } from 'path';

const PACKAGES_DIR = 'packages';
const BUILD_DIR = 'build';

function log(message) {
  console.log(`${message}`);
}

function error(message) {
  console.error(`ERROR: ${message}`);
}

function warn(message) {
  console.warn(`WARN: ${message}`);
}

function runCommand(command, options = {}) {
  try {
    log(`Running: ${command}`);
    const result = execSync(command, {
      encoding: 'utf8',
      stdio: 'pipe',
      ...options
    });
    return { success: true, output: result.trim() };
  } catch (err) {
    return {
      success: false,
      error: err.message,
      output: err.stdout?.trim() || '',
      stderr: err.stderr?.trim() || ''
    };
  }
}

function checkPackageExists(name, version) {
  try {
    // Use npm view to check if package exists
    execSync(`npm view ${name}@${version}`, {
      encoding: 'utf8',
      stdio: 'pipe'
    });
    return true
  } catch (err) {
    // Package doesn't exist if npm view fails
    return false
  }
}

function getPackageInfo(packageJsonPath) {
  if (!existsSync(packageJsonPath)) {
    return null;
  }
  
  try {
    const packageJson = JSON.parse(readFileSync(packageJsonPath, 'utf8'));
    return {
      name: packageJson.name,
      version: packageJson.version,
      private: packageJson.private,
      publishConfig: packageJson.publishConfig,
      optionalDependencies: packageJson.optionalDependencies || {}
    };
  } catch (err) {
    error(`Failed to read ${packageJsonPath}: ${err.message}`);
    return null;
  }
}

function getPreReleaseInfo() {
  const preJsonPath = '.changeset/pre.json';
  
  if (!existsSync(preJsonPath)) {
    return null;
  }
  
  try {
    const preJson = JSON.parse(readFileSync(preJsonPath, 'utf8'));
    return {
      mode: preJson.mode,
      tag: preJson.tag,
      initialVersions: preJson.initialVersions
    };
  } catch (err) {
    error(`Failed to read ${preJsonPath}: ${err.message}`);
    return null;
  }
}

function discoverPackages() {
  const packages = [];

  // Discover regular packages
  const packageDirs = readdirSync(PACKAGES_DIR, { withFileTypes: true })
    .filter(dirent => dirent.isDirectory())
    .map(dirent => join(PACKAGES_DIR, dirent.name));

  for (const packageDir of packageDirs) {
    const packageJsonPath = join(packageDir, 'package.json');
    const packageInfo = getPackageInfo(packageJsonPath);

    if (!packageInfo || packageInfo.private) {
      continue;
    }

    // Check if this is a Go package by looking for main.go
    const isGoPackage = existsSync(join(packageDir, 'main.go'));

    if (isGoPackage) {
      // Go-based package with platform builds
      const buildDir = join(packageDir, BUILD_DIR);
      const buildPackages = discoverBuildPackages(buildDir);
      packages.push({
        type: 'go',
        name: packageInfo.name,
        version: packageInfo.version,
        path: packageDir,
        buildDir: buildDir,
        mainPackage: null, // Will be found in buildPackages
        platformPackages: buildPackages.filter(p => !p.isMainPackage),
        allBuildPackages: buildPackages
      });
    } else {
      // Regular Node.js package
      packages.push({
        type: 'node',
        name: packageInfo.name,
        version: packageInfo.version,
        path: packageDir,
        packageInfo
      });
    }
  }

  return packages;
}

function discoverBuildPackages(buildDir) {
  const buildPackages = [];

  if (!existsSync(buildDir)) {
    return buildPackages;
  }

  const buildSubdirs = readdirSync(buildDir, { withFileTypes: true })
    .filter(dirent => dirent.isDirectory())
    .map(dirent => join(buildDir, dirent.name));

  for (const subdir of buildSubdirs) {
    const packageJsonPath = join(subdir, 'package.json');
    const packageInfo = getPackageInfo(packageJsonPath);

    if (packageInfo) {
      // Main package typically doesn't have platform-specific suffixes
      // Platform packages have names like "houdini-core-darwin-arm64"
      const subdirName = basename(subdir);
      const isMainPackage = !subdirName.includes('-darwin-') &&
                           !subdirName.includes('-linux-') &&
                           !subdirName.includes('-windows-') &&
                           !packageInfo.os &&
                           !packageInfo.cpu;

      buildPackages.push({
        name: packageInfo.name,
        version: packageInfo.version,
        path: subdir,
        packageInfo,
        isMainPackage
      });
    }
  }

  return buildPackages;
}

async function publishPackage(packagePath, packageName, packageVersion, options = {}) {
  const { isSnapshot = false, snapshotTag = '', preReleaseTag = '', retryOnFailure = true } = options;

  // Check if package already exists
  const packageCheck = checkPackageExists(packageName, packageVersion);
  if (packageCheck) {
    log(`📦 Package ${packageName}@${packageVersion} already exists`);
    return { success: true, skipped: true };
  }

  log(`🚀 Publishing ${packageName}@${packageVersion} from ${packagePath}...`);

  const publishArgs = ['pnpm', 'publish', '--access', 'public'];

  // Determine which tag to use
  if (isSnapshot && snapshotTag) {
    publishArgs.push('--tag', snapshotTag);
  } else if (preReleaseTag) {
    publishArgs.push('--tag', preReleaseTag);
  }

  // Add provenance if supported
  if (process.env.NPM_CONFIG_PROVENANCE === 'true') {
    publishArgs.push('--provenance');
  }

  // Skip git checks in CI environments to avoid "unclean working tree" errors
  if (process.env.CI) {
    publishArgs.push('--no-git-checks');
  }

  const result = runCommand(publishArgs.join(' '), { cwd: packagePath });

  if (result.success) {
    log(`✅ Successfully published ${packageName}@${packageVersion}`);
    return { success: true };
  }

  // Enhanced error logging
  error(`Failed to publish ${packageName}:`);
  error(`Command: ${publishArgs.join(' ')}`);
  error(`Working directory: ${packagePath}`);
  error(`Exit code/Error: ${result.error}`);
  if (result.output) {
    error(`STDOUT: ${result.output}`);
  }
  if (result.stderr) {
    error(`STDERR: ${result.stderr}`);
  }

  // Handle common errors - check for various "already published" error formats
  const isAlreadyPublished = result.stderr && (
    result.stderr.includes('You cannot publish over the previously published versions') ||
    result.stderr.includes('already exists') ||
    result.stderr.includes('You cannot publish over') ||
    (result.stderr.includes('403 Forbidden') && result.stderr.includes('publish over'))
  );

  if (isAlreadyPublished) {
    return { success: true, skipped: true };
  }

  if (result.stderr && result.stderr.includes('404') && result.stderr.includes('Not found') && retryOnFailure) {
    warn(`Package ${packageName} not found, might be a new package. Retrying...`);
    // For new packages, sometimes we need to retry
    await new Promise(resolve => setTimeout(resolve, 2000));
    return publishPackage(packagePath, packageName, packageVersion, { ...options, retryOnFailure: false });
  }

  // Check for authentication issues
  if (result.stderr && (result.stderr.includes('401') || result.stderr.includes('403') || result.stderr.includes('authentication'))) {
    error('❌ Authentication failed! Check NPM_TOKEN or OIDC configuration.');
  }

  return { success: false, error: result.error };
}

async function publishGoPackage(mod, options = {}) {
  const results = [];

  // Publish platform packages first
  for (const platformPkg of mod.platformPackages) {
    const result = await publishPackage(platformPkg.path, platformPkg.name, platformPkg.version, options);
    results.push({ package: platformPkg.name, ...result });
  }

  // Find and publish main package last (it depends on platform packages)
  const mainPackage = mod.allBuildPackages.find(p => p.isMainPackage);
  if (mainPackage) {
    const result = await publishPackage(mainPackage.path, mainPackage.name, mainPackage.version, options);
    results.push({ package: mainPackage.name, ...result });
  }

  return results;
}

async function publishAllPackages(packages, options = {}) {
  const allResults = [];
  
  for (const pkg of packages) {
    if (pkg.type === 'go') {
      const results = await publishGoPackage(pkg, options);
      allResults.push(...results);
    } else {
      const result = await publishPackage(pkg.path, pkg.name, pkg.version, options);
      allResults.push({ package: pkg.name, ...result });
    }
  }
  
  return allResults;
}

function showHelp() {
  console.log(`
Houdini Release Script

Usage:
  node packages/_scripts/release.js [options]

Options:
  --snapshot              Publish snapshot release
  --tag=<tag>            Specify tag for snapshot release (e.g., --tag=commit-abc123)
  --help                 Show this help message

Examples:
  node packages/_scripts/release.js                           # Regular release
  node packages/_scripts/release.js --snapshot --tag=test     # Snapshot release

NPM Scripts:
  pnpm run release                    # Regular release
  pnpm run release:snapshot           # Snapshot release
`);
}

async function main() {
  const args = process.argv.slice(2);

  if (args.includes('--help') || args.includes('-h')) {
    showHelp();
    return;
  }

  const isSnapshot = args.includes('--snapshot');
  const snapshotTag = args.find(arg => arg.startsWith('--tag='))?.split('=')[1];

  log('Starting Houdini release process...');

  // Check for prerelease mode
  const preReleaseInfo = getPreReleaseInfo();
  const isPreRelease = preReleaseInfo !== null;

  if (isPreRelease) {
    log(`🚧 Prerelease mode detected - tag: ${preReleaseInfo.tag}`);
  } else {
    log('📦 Standard release mode');
  }

  // Discover all packages
  const packages = discoverPackages()

  log("🔍 Discovered packages to publish:");
  packages.forEach(pkg => {
    if (pkg.type === 'go') {
      log(`- ${pkg.name} (Go package with ${pkg.platformPackages.length} platform builds)`);
      for (const buildPkg of pkg.platformPackages) {
        log(` └─ ${buildPkg.name}@${buildPkg.version}`);
      }
    } else {
      log(`- ${pkg.name} (Node.js package)`);
    }
  });

  console.log("\nPublishing packages...")

  try {
    // Publish packages individually
    const results = await publishAllPackages(packages, {
      isSnapshot,
      snapshotTag,
      preReleaseTag: isPreRelease ? preReleaseInfo.tag : ''
    });

    // Summary
    const published = results.filter(r => r.success && !r.skipped).length;
    const skipped = results.filter(r => r.skipped).length;
    const failed = results.filter(r => !r.success).length;

    log(`\n📊 Publishing Summary:`);
    log(`  ✅ Published: ${published}`);
    log(`  ⏭  Skipped: ${skipped}`);
    log(`  ❌ Failed: ${failed}\n`);

    if (failed > 0) {
      error('Some packages failed to publish');
      process.exit(1);
    }

    log('🎉 All packages published successfully!');

  } catch (err) {
    error(`Publishing failed: ${err.message}`);
    process.exit(1);
  }
}

// Run the script
main().catch(err => {
  error(`Script failed: ${err.message}`);
  console.error(err.stack);
  process.exit(1);
});
