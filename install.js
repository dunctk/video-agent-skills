#!/usr/bin/env node
const fs = require("fs");
const path = require("path");
const os = require("os");
const https = require("https");
const { execSync } = require("child_process");

const pkg = require("./package.json");
const rootDir = __dirname;
const skillName = "video-agent-skills";
const repo = "dunctk/video-agent-skills";
const version = pkg.version;
const skillSrc = path.join(rootDir, "claude-skill", skillName);
const targetDir = path.join(os.homedir(), ".claude", "skills", skillName);
const targetBinDir = path.join(targetDir, "bin");

function ensureGoAvailable() {
  try {
    execSync("go version", { stdio: "ignore" });
    return true;
  } catch (err) {
    return false;
  }
}

function copySkillMetadata() {
  if (!fs.existsSync(skillSrc)) {
    console.error(`Skill source not found at ${skillSrc}`);
    process.exit(1);
  }

  fs.mkdirSync(targetBinDir, { recursive: true });
  fs.copyFileSync(
    path.join(skillSrc, "SKILL.md"),
    path.join(targetDir, "SKILL.md")
  );
}

function resolveAsset() {
  const platform = process.platform;
  const arch = process.arch;

  const map = {
    darwin: { x64: "darwin_amd64", arm64: "darwin_arm64" },
    linux: { x64: "linux_amd64", arm64: "linux_arm64" },
    win32: { x64: "windows_amd64" },
  };

  if (!map[platform] || !map[platform][arch]) {
    return null;
  }

  const suffix = map[platform][arch];
  const base = `${skillName}_${version}_${suffix}`;
  const ext = platform === "win32" ? "zip" : "tar.gz";
  const asset = `${base}.${ext}`;
  const url = `https://github.com/${repo}/releases/download/v${version}/${asset}`;
  return { asset, url, ext };
}

function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);

    const get = (currentUrl) => {
      https
        .get(currentUrl, (res) => {
          if (
            res.statusCode === 301 ||
            res.statusCode === 302 ||
            res.statusCode === 307 ||
            res.statusCode === 308
          ) {
            if (!res.headers.location) {
              reject(new Error("Redirect without location"));
              return;
            }
            get(res.headers.location);
            return;
          }

          if (res.statusCode !== 200) {
            reject(new Error(`Download failed (${res.statusCode})`));
            return;
          }

          res.pipe(file);
          file.on("finish", () => {
            file.close(resolve);
          });
        })
        .on("error", (err) => {
          fs.unlink(dest, () => reject(err));
        });
    };

    get(url);
  });
}

function extractArchive(archivePath, ext) {
  if (ext === "zip") {
    execSync(
      `powershell -NoProfile -NonInteractive -Command "Expand-Archive -LiteralPath '${archivePath}' -DestinationPath '${targetBinDir}' -Force"`,
      { stdio: "inherit" }
    );
    return;
  }

  execSync(`tar -xzf "${archivePath}" -C "${targetBinDir}"`, {
    stdio: "inherit",
  });
}

async function downloadBinary() {
  const asset = resolveAsset();
  if (!asset) {
    return false;
  }

  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "video-agent-skills-"));
  const archivePath = path.join(tmpDir, asset.asset);

  try {
    await downloadFile(asset.url, archivePath);
    extractArchive(archivePath, asset.ext);

    if (process.platform !== "win32") {
      const binPath = path.join(targetBinDir, skillName);
      if (fs.existsSync(binPath)) {
        fs.chmodSync(binPath, 0o755);
      }
    }

    return true;
  } catch (err) {
    return false;
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }
}

function buildFromSource() {
  const binName = process.platform === "win32" ? `${skillName}.exe` : skillName;
  const binPath = path.join(targetBinDir, binName);

  if (!ensureGoAvailable()) {
    console.error(
      "Go is required to build the video-agent-skills binary during install."
    );
    console.error("Install Go, then re-run: npm install");
    process.exit(1);
  }

  execSync(`go build -o "${binPath}" ./`, {
    cwd: rootDir,
    stdio: "inherit",
  });
}

async function main() {
  copySkillMetadata();

  if (!process.env.VIDEO_AGENT_SKILLS_FORCE_BUILD) {
    const downloaded = await downloadBinary();
    if (downloaded) {
      console.log(`Claude Code skill installed to ${targetDir}`);
      return;
    }
  }

  buildFromSource();
  console.log(`Claude Code skill installed to ${targetDir}`);
}

main().catch((err) => {
  console.error(err.message || err);
  process.exit(1);
});
