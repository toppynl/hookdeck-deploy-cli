#!/usr/bin/env node
const { execFileSync } = require('child_process');
const { existsSync } = require('fs');
const path = require('path');

const platform = process.platform;
const arch = process.arch;

const archMap = {
  'x64': 'amd64',
  'arm64': 'arm64',
  'ia32': '386'
};
const goArch = archMap[arch] || arch;

const binaryDir = `${platform}-${goArch}`;
const binaryName = platform === 'win32' ? 'hookdeck-deploy-cli.exe' : 'hookdeck-deploy-cli';
const binaryPath = path.join(__dirname, '..', 'binaries', binaryDir, binaryName);

if (!existsSync(binaryPath)) {
  console.error(`Error: Unsupported platform: ${platform}-${arch}`);
  console.error(`Expected binary at: ${binaryPath}`);
  console.error(`Please report this issue at https://github.com/toppynl/hookdeck-deploy-cli/issues`);
  process.exit(1);
}

try {
  execFileSync(binaryPath, process.argv.slice(2), { stdio: 'inherit' });
} catch (error) {
  process.exit(error.status ?? 1);
}
