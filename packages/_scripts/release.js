#!/usr/bin/env node

/**
 * Comprehensive release script for Houdini monorepo
 * Handles both regular packages and Go-based packages with platform builds
 * Integrates with changesets for version management
 */

import { execSync, spawn } from 'child_process';
import { readFileSync, readdirSync, existsSync, statSync } from 'fs';
import { join, basename } from 'path';

const PACKAGES_DIR = 'packages';
const BUILD_DIR = 'build';

function log(message) {
  console.log(`[release] ${message}`);
}

function error(message) {
  console.error(`[release] ERROR: ${message}`);
}

function warn(message) {
  console.warn(`[release] WARN: ${message}`);
}

function runCommand(command, options = {}) {
  const { dryRun = false } = options;

  if (dryRun) {
    log(`[DRY RUN] Would run: ${command}`);
    if (options.cwd) {
      log(`[DRY RUN] In directory: ${options.cwd}`);
    }
    // Simulate success for dry-run mode
    return { success: true, output: '[DRY RUN] Command not executed', dryRun: true };
  }

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
    const mainGoPath = join(packageDir, 'main.go');
    const isGoPackage = existsSync(mainGoPath);

    if (isGoPackage) {
      // Go-based package with platform builds
      const buildDir = join(packageDir, BUILD_DIR);
      const buildPackages = discoverBuildPackages(buildDir);
      packages.push({
        type: 'go-package',
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
        type: 'node-package',
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

async function publishPackage(packagePath, packageName, options = {}) {
  const { isSnapshot = false, snapshotTag = '', preReleaseTag = '', retryOnFailure = true, dryRun = false } = options;

  log(`${dryRun ? '[DRY RUN] ' : ''}Publishing ${packageName} from ${packagePath}...`);

  const publishArgs = ['npm', 'publish', '--access', 'public'];

  // Determine which tag to use
  if (isSnapshot && snapshotTag) {
    publishArgs.push('--tag', snapshotTag);
    log(`${dryRun ? '[DRY RUN] ' : ''}Using snapshot tag: ${snapshotTag}`);
  } else if (preReleaseTag) {
    publishArgs.push('--tag', preReleaseTag);
    log(`${dryRun ? '[DRY RUN] ' : ''}Using prerelease tag: ${preReleaseTag}`);
  } else {
    log(`${dryRun ? '[DRY RUN] ' : ''}Using default tag: latest`);
  }

  // Add provenance if supported
  if (process.env.NPM_CONFIG_PROVENANCE === 'true') {
    publishArgs.push('--provenance');
  }

  // Add dry-run flag for npm
  if (dryRun) {
    publishArgs.push('--dry-run');
  }

  // In dry-run mode, just print what we would do without executing
  if (dryRun) {
    log(`📋 [DRY RUN] Would execute: ${publishArgs.join(' ')}`);
    log(`📁 [DRY RUN] Working directory: ${packagePath}`);

    // Check if package.json exists to simulate basic validation
    const packageJsonPath = join(packagePath, 'package.json');
    if (existsSync(packageJsonPath)) {
      const packageInfo = getPackageInfo(packageJsonPath);
      if (packageInfo) {
        log(`📦 [DRY RUN] Package: ${packageInfo.name}@${packageInfo.version}`);
        log(`🏷️ [DRY RUN] Would publish as: ${packageInfo.name}@${isSnapshot && snapshotTag ? snapshotTag : preReleaseTag || 'latest'}`);
        log(`✅ [DRY RUN] Package validation passed - would publish successfully`);
        return { success: true, dryRun: true };
      } else {
        warn(`⚠️ [DRY RUN] Invalid package.json - would fail to publish`);
        return { success: false, dryRun: true, error: 'Invalid package.json' };
      }
    } else {
      warn(`⚠️ [DRY RUN] No package.json found - would fail to publish`);
      return { success: false, dryRun: true, error: 'No package.json found' };
    }
  }

  // Real publishing logic (when not in dry-run mode)
  const result = runCommand(publishArgs.join(' '), { cwd: packagePath });

  if (result.success) {
    log(`✅ Successfully published ${packageName}`);
    return { success: true };
  }

  // Handle common errors
  if (result.stderr.includes('You cannot publish over the previously published versions') ||
      result.stderr.includes('already exists')) {
    log(`ℹ️ ${packageName} already published`);
    return { success: true, skipped: true };
  }

  if (result.stderr.includes('404') && result.stderr.includes('Not found') && retryOnFailure) {
    warn(`Package ${packageName} not found, might be a new package. Retrying...`);
    // For new packages, sometimes we need to retry
    await new Promise(resolve => setTimeout(resolve, 2000));
    return publishPackage(packagePath, packageName, { ...options, retryOnFailure: false });
  }

  error(`Failed to publish ${packageName}:`);
  error(`Command: ${publishArgs.join(' ')}`);
  error(`Error: ${result.error}`);
  error(`STDERR: ${result.stderr}`);

  return { success: false, error: result.error };
}

async function publishGoPackage(goPackage, options = {}) {
  log(`Publishing Go package: ${goPackage.name}`);
  
  const results = [];
  
  // Publish platform packages first
  for (const platformPkg of goPackage.platformPackages) {
    const result = await publishPackage(platformPkg.path, platformPkg.name, options);
    results.push({ package: platformPkg.name, ...result });
  }
  
  // Find and publish main package last (it depends on platform packages)
  const mainPackage = goPackage.allBuildPackages.find(p => p.isMainPackage);
  if (mainPackage) {
    const result = await publishPackage(mainPackage.path, mainPackage.name, options);
    results.push({ package: mainPackage.name, ...result });
  }
  
  return results;
}

async function publishAllPackages(packages, options = {}) {
  const allResults = [];
  
  for (const pkg of packages) {
    if (pkg.type === 'go-package') {
      const results = await publishGoPackage(pkg, options);
      allResults.push(...results);
    } else {
      const result = await publishPackage(pkg.path, pkg.name, options);
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
  --dry-run              Test run - shows what would happen without executing commands
  --help                 Show this help message

Examples:
  node packages/_scripts/release.js                           # Regular release
  node packages/_scripts/release.js --dry-run                 # Test regular release
  node packages/_scripts/release.js --snapshot --tag=test     # Snapshot release
  node packages/_scripts/release.js --snapshot --dry-run      # Test snapshot release

NPM Scripts:
  pnpm run release                    # Regular release
  pnpm run release:dry-run            # Test regular release
  pnpm run release:snapshot           # Snapshot release
  pnpm run release:snapshot:dry-run   # Test snapshot release
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
  const dryRun = args.includes('--dry-run');

  log(`Starting Houdini release process${dryRun ? ' (DRY RUN MODE)' : ''}...`);

  if (dryRun) {
    log('🧪 DRY RUN MODE: No packages will actually be published');
    log('🧪 This will show you what would happen without making changes');
  }

  // Check for prerelease mode
  const preReleaseInfo = getPreReleaseInfo();
  const isPreRelease = preReleaseInfo !== null;

  if (isPreRelease) {
    log(`🚧 Prerelease mode detected - tag: ${preReleaseInfo.tag}, mode: ${preReleaseInfo.mode}`);
  } else {
    log('📦 Regular release mode');
  }

  // Discover all packages
  const packages = discoverPackages();
  log(`${dryRun ? '[DRY RUN] ' : ''}Discovered ${packages.length} packages:`);
  packages.forEach(pkg => {
    if (pkg.type === 'go-package') {
      log(`  - ${pkg.name} (Go package with ${pkg.platformPackages.length} platform builds)`);
      if (dryRun) {
        pkg.allBuildPackages.forEach(buildPkg => {
          log(`    └─ ${buildPkg.name}@${buildPkg.version} ${buildPkg.isMainPackage ? '(main)' : '(platform)'}`);
        });
      }
    } else {
      log(`  - ${pkg.name} (Node.js package)`);
      if (dryRun) {
        log(`    └─ ${pkg.name}@${pkg.version}`);
      }
    }
  });

  if (!isSnapshot) {
    log(`${dryRun ? '[DRY RUN] ' : ''}Regular release mode - using custom publishing logic`);
    log(`${dryRun ? '[DRY RUN] ' : ''}Changeset handles version management, custom script handles publishing`);
  } else {
    if (isPreRelease) {
      log(`⚠️ ${dryRun ? '[DRY RUN] ' : ''}Skipping snapshot release - in prerelease mode (tag: ${preReleaseInfo.tag})`);
      log(`${dryRun ? '[DRY RUN] ' : ''}Snapshot releases are disabled during prerelease mode`);
      return;
    }
    log(`📸 ${dryRun ? '[DRY RUN] ' : ''}Snapshot mode - will publish with tag: ${snapshotTag}`);
  }

  // Publish packages individually
  const publishOptions = {
    isSnapshot,
    snapshotTag,
    preReleaseTag: isPreRelease ? preReleaseInfo.tag : '',
    dryRun
  };

  try {
    const results = await publishAllPackages(packages, publishOptions);

    // Summary
    const successful = results.filter(r => r.success).length;
    const skipped = results.filter(r => r.skipped).length;
    const failed = results.filter(r => !r.success).length;
    const dryRunResults = results.filter(r => r.dryRun).length;

    log(`\n📊 ${dryRun ? 'Dry Run ' : ''}Publishing Summary:`);
    log(`  ✅ ${dryRun ? 'Would succeed' : 'Successful'}: ${successful}`);
    log(`  ⏭️ Skipped: ${skipped}`);
    log(`  ❌ ${dryRun ? 'Would fail' : 'Failed'}: ${failed}`);

    if (dryRun) {
      log(`\n🧪 DRY RUN COMPLETE - No packages were actually published`);
      log(`🧪 ${successful} packages would be published successfully`);
      if (failed > 0) {
        log(`🧪 ${failed} packages would fail to publish`);
      }
    } else {
      if (failed > 0) {
        error('Some packages failed to publish');
        process.exit(1);
      }
      log('🎉 All packages published successfully!');
    }

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
