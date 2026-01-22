---
name: video-agent-skills
description: Provide candid, practical feedback on a video file using Gemini. Use when the user wants critique or production feedback on a video.
---

# Video Agent Skills

## When to use
Use this skill when the user asks for video critique, production feedback, or scene-by-scene notes on a local video file.

## Requirements
- A Gemini API key must be available as `GEMINI_API_KEY` or `GOOGLE_API_KEY`.
- The video file path must be accessible on disk.

If the key is not set, ask the user to set it (or use `~/.config/video-agent-skills/.env`). Do not proceed without a key.

## Command
Run the bundled binary in this skill folder:

```bash
# {skill_dir} is the absolute path to this skill folder
{skill_dir}/bin/video-agent-skills feedback \
  -video "{video_path}" \
  -prompt "{optional_user_prompt}"
```

Harshness tiers (optional):
```bash
{skill_dir}/bin/video-agent-skills feedback \
  -video "{video_path}" \
  -tone nice
```

For reverse-engineering the likely prompt used to make a video:

```bash
{skill_dir}/bin/video-agent-skills reverse \
  -video "{video_path}" \
  -prompt "{optional_user_prompt}"
```

Notes:
- Omit `-prompt` if the user did not provide custom guidance.
- You can pass `-model` or `-api-key` if the user specifies them.
- Use `-tone` only for `feedback`. For `reverse`, it is ignored.
- On Windows, the binary name ends with `.exe`.

## Output handling
Return the tool output directly to the user. If the tool reports missing API keys or file errors, surface the error and ask the user to resolve it.
