#!/usr/bin/env node
const fs = require("fs");
const path = require("path");
const os = require("os");
const { execSync } = require("child_process");

const rootDir = __dirname;
const skillName = "video-agent-skills";
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

function buildBinary() {
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

copySkillMetadata();
buildBinary();

console.log(`Claude Code skill installed to ${targetDir}`);
