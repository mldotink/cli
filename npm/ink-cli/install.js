const { existsSync } = require("fs");
const { join } = require("path");

const PLATFORMS = {
  "darwin-arm64": "@mldotink/cli-darwin-arm64",
  "darwin-x64": "@mldotink/cli-darwin-x64",
  "linux-x64": "@mldotink/cli-linux-x64",
  "linux-arm64": "@mldotink/cli-linux-arm64",
  "win32-x64": "@mldotink/cli-win32-x64",
};

const platform = `${process.platform}-${process.arch}`;
const pkg = PLATFORMS[platform];

if (!pkg) {
  console.error(`Unsupported platform: ${platform}`);
  console.error(`Supported: ${Object.keys(PLATFORMS).join(", ")}`);
  process.exit(1);
}

try {
  const binPath = require.resolve(`${pkg}/bin/ink`);
  if (!existsSync(binPath)) {
    throw new Error(`Binary not found at ${binPath}`);
  }
} catch (e) {
  console.error(`Failed to find ink binary for ${platform}`);
  console.error(`Package ${pkg} may not be installed.`);
  console.error(
    `Try: npm install ${pkg} or download from https://github.com/mldotink/cli/releases`
  );
  process.exit(1);
}
