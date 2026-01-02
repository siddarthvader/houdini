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
    
    const buildDir = join(packageDir, BUILD_DIR);
    const hasBuilds = existsSync(buildDir);
    
    if (hasBuilds) {
      // Go-based package with platform builds
      const buildPackages = discoverBuildPackages(buildDir);
      packages.push({
        type: 'go-package',
        name: packageInfo.name,
        version: packageInfo.version,
        path: packageDir,
        buildDir: buildDir,
        mainPackage: null, // Will be found in buildPackages
        platformPackages: buildPackages.filter(p => p.name !== packageInfo.name),
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
  
  const buildSubdirs = readdirSync(buildDir, { withFileTypes: true })
    .filter(dirent => dirent.isDirectory())
    .map(dirent => join(buildDir, dirent.name));
  
  for (const subdir of buildSubdirs) {
    const packageJsonPath = join(subdir, 'package.json');
    const packageInfo = getPackageInfo(packageJsonPath);
    
    if (packageInfo) {
      buildPackages.push({
        name: packageInfo.name,
        version: packageInfo.version,
        path: subdir,
        packageInfo,
        isMainPackage: !packageInfo.os && !packageInfo.cpu // Main package has no platform restrictions
      });
    }
  }
  
  return buildPackages;
}

async function publishPackage(packagePath, packageName, options = {}) {
  const { isSnapshot = false, snapshotTag = '', preReleaseTag = '', retryOnFailure = true } = options;
  
  log(`Publishing ${packageName} from ${packagePath}...`);
  
  const publishArgs = ['npm', 'publish', '--access', 'public'];
  
  // Determine which tag to use
  if (isSnapshot && snapshotTag) {
    publishArgs.push('--tag', snapshotTag);
    log(`Using snapshot tag: ${snapshotTag}`);
  } else if (preReleaseTag) {
    publishArgs.push('--tag', preReleaseTag);
    log(`Using prerelease tag: ${preReleaseTag}`);
  } else {
    log('Using default tag: latest');
  }
  
  // Add provenance if supported
  if (process.env.NPM_CONFIG_PROVENANCE === 'true') {
    publishArgs.push('--provenance');
  }
  
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

async function main() {
  const args = process.argv.slice(2);
  const isSnapshot = args.includes('--snapshot');
  const snapshotTag = args.find(arg => arg.startsWith('--tag='))?.split('=')[1];

  log('Starting Houdini release process...');

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
  log(`Discovered ${packages.length} packages:`);
  packages.forEach(pkg => {
    if (pkg.type === 'go-package') {
      log(`  - ${pkg.name} (Go package with ${pkg.platformPackages.length} platform builds)`);
    } else {
      log(`  - ${pkg.name} (Node.js package)`);
    }
  });

  if (!isSnapshot) {
    // Try changeset publish first for regular releases
    log('Attempting changeset publish...');
    const changesetResult = runCommand('changeset publish');

    if (changesetResult.success) {
      log('✅ Changeset publish succeeded!');
      return;
    }

    warn('Changeset publish failed, falling back to individual publishing...');
    warn(`Changeset error: ${changesetResult.error}`);
  } else {
    if (isPreRelease) {
      log(`⚠️ Skipping snapshot release - in prerelease mode (tag: ${preReleaseInfo.tag})`);
      log('Snapshot releases are disabled during prerelease mode');
      return;
    }
    log(`📸 Snapshot mode - will publish with tag: ${snapshotTag}`);
  }

  // Publish packages individually
  const publishOptions = {
    isSnapshot,
    snapshotTag,
    preReleaseTag: isPreRelease ? preReleaseInfo.tag : ''
  };

  try {
    const results = await publishAllPackages(packages, publishOptions);

    // Summary
    const successful = results.filter(r => r.success).length;
    const skipped = results.filter(r => r.skipped).length;
    const failed = results.filter(r => !r.success).length;

    log(`\n📊 Publishing Summary:`);
    log(`  ✅ Successful: ${successful}`);
    log(`  ⏭️ Skipped: ${skipped}`);
    log(`  ❌ Failed: ${failed}`);

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
