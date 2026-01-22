package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/genai"
)

const (
	defaultModel         = "gemini-3-flash-preview"
	defaultTone          = "harsh"
	promptNice           = "You are a senior motion designer and creative director. Be kind and constructive. Focus on clarity, pacing, hierarchy, and polish. Offer encouragement and actionable improvements without harsh language. Output format:\n1) Quick read (2-3 sentences)\n2) Strengths (2 bullets max)\n3) Issues (6 bullets)\n4) Improvements (6 bullets, each with a concrete fix; include approximate timecodes if possible)."
	promptNormal         = "You are a senior motion designer and creative director. Be direct and candid. Critique storytelling, camera movement, visibility/legibility of key elements, timing, and overall professional polish, plus pacing, rhythm, transitions, typography, composition, color, and easing. Balance critique with actionable improvements. Output format:\n1) Quick read (2-3 sentences)\n2) Strengths (2 bullets max)\n3) Issues (6 bullets)\n4) Improvements (6 bullets, each with a concrete fix; include approximate timecodes if possible)."
	promptHarsh          = "You are a senior motion designer and creative director. Be direct and a bit harsh. Critique storytelling, camera movement, visibility/legibility of key elements, timing, and overall professional polish, plus pacing, rhythm, transitions, typography, composition, color, and easing. Call out what looks amateurish or generic. Provide more improvement ideas than praise. Output format:\n1) Quick read (2-3 sentences)\n2) Strengths (2 bullets max)\n3) Issues (6 bullets)\n4) Improvements (6 bullets, each with a concrete fix; include approximate timecodes if possible)."
	promptSuperHarsh     = "You are a senior motion designer and creative director. Be brutally honest and uncompromising. Assume the work must reach top-tier studio standards. Call out every weak choice, cliché, and amateurish execution. Minimize praise, maximize specific, actionable fixes. Output format:\n1) Quick read (2-3 sentences)\n2) Strengths (2 bullets max)\n3) Issues (6 bullets)\n4) Improvements (6 bullets, each with a concrete fix; include approximate timecodes if possible)."
	defaultReversePrompt = "You are reverse-engineering the prompt that likely produced this video. Infer subject, style, camera behavior, motion language, lighting, typography, color palette, aspect ratio, duration, and rendering style. Output only a single reconstructed prompt as one paragraph. No preamble, no bullets, no explanations."
)

func main() {
	if len(os.Args) < 2 || isHelpArg(os.Args[1]) {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "feedback":
		if err := runVideoCommand("feedback", os.Args[2:], promptHarsh); err != nil {
			log.Fatal(err)
		}
	case "reverse":
		if err := runVideoCommand("reverse", os.Args[2:], defaultReversePrompt); err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func isHelpArg(arg string) bool {
	switch arg {
	case "-h", "--help", "help":
		return true
	default:
		return false
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage:
  video-agent-skills feedback -video <path> [options]
  video-agent-skills reverse  -video <path> [options]

Options:
  -video string     Path to a video file (required)
  -model string     Gemini model name (default: %s)
  -prompt string    Prompt override (feedback or reverse)
  -tone string      Feedback harshness: nice|normal|harsh|super-harsh (default: %s)
  -api-key string   Gemini API key (overrides GEMINI_API_KEY/GOOGLE_API_KEY)
  -h, --help        Show help

Examples:
  video-agent-skills feedback -video ./examples/RefreshAgent-Demo-30s.mp4
  video-agent-skills reverse -video ./examples/RefreshAgent-Demo-30s.mp4
`, defaultModel, defaultTone)
}

func printCommandUsage(command string) {
	switch command {
	case "reverse":
		fmt.Fprintf(os.Stderr, `Usage:
  video-agent-skills reverse -video <path> [options]

Options:
  -video string     Path to a video file (required)
  -model string     Gemini model name (default: %s)
  -prompt string    Prompt override for reverse engineering
  -api-key string   Gemini API key (overrides GEMINI_API_KEY/GOOGLE_API_KEY)
  -h, --help        Show help
`, defaultModel)
	default:
		fmt.Fprintf(os.Stderr, `Usage:
  video-agent-skills feedback -video <path> [options]

Options:
  -video string     Path to a video file (required)
  -model string     Gemini model name (default: %s)
  -prompt string    Prompt to guide feedback
  -tone string      Feedback harshness: nice|normal|harsh|super-harsh (default: %s)
  -api-key string   Gemini API key (overrides GEMINI_API_KEY/GOOGLE_API_KEY)
  -h, --help        Show help
`, defaultModel, defaultTone)
	}
}

func runVideoCommand(command string, args []string, promptDefault string) error {
	var (
		fs        = flag.NewFlagSet(command, flag.ContinueOnError)
		videoPath = fs.String("video", "", "Path to a video file")
		model     = fs.String("model", defaultModel, "Gemini model name")
		prompt    = fs.String("prompt", promptDefault, "Prompt to guide output")
		tone      = fs.String("tone", defaultTone, "Feedback harshness: nice|normal|harsh|super-harsh")
		apiKey    = fs.String("api-key", "", "Gemini API key (overrides GEMINI_API_KEY/GOOGLE_API_KEY)")
		help      = fs.Bool("help", false, "Show help")
		h         = fs.Bool("h", false, "Show help")
	)
	fs.Usage = func() {
		printCommandUsage(command)
	}
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.Usage()
			return nil
		}
		return err
	}
	if *help || *h {
		fs.Usage()
		return nil
	}

	if *videoPath == "" {
		return errors.New("-video is required")
	}

	promptValue := *prompt
	promptSet := false
	toneSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "prompt" {
			promptSet = true
		}
		if f.Name == "tone" {
			toneSet = true
		}
	})

	if !promptSet && command == "feedback" {
		selection, err := feedbackPromptForTone(*tone)
		if err != nil {
			return err
		}
		promptValue = selection
	} else if command != "feedback" && toneSet {
		log.Printf("warning: -tone is ignored for %s", command)
	}

	if err := loadEnvFile(defaultConfigEnvPath()); err != nil {
		log.Printf("warning: %v", err)
	}

	key := strings.TrimSpace(*apiKey)
	if key == "" {
		key = strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	}
	if key == "" {
		key = strings.TrimSpace(os.Getenv("GOOGLE_API_KEY"))
	}
	if key == "" {
		return errors.New("missing API key: set GEMINI_API_KEY or GOOGLE_API_KEY, or pass -api-key")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  key,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return err
	}

	mimeType, err := detectVideoMIME(*videoPath)
	if err != nil {
		return err
	}

	uploaded, err := client.Files.UploadFromPath(
		ctx,
		*videoPath,
		&genai.UploadFileConfig{
			MIMEType: mimeType,
		},
	)
	if err != nil {
		return err
	}

	uploaded, err = waitForActiveFile(ctx, client, uploaded, 5*time.Second)
	if err != nil {
		return err
	}

	parts := []*genai.Part{
		genai.NewPartFromURI(uploaded.URI, uploaded.MIMEType),
		genai.NewPartFromText(promptValue),
	}

	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	response, err := client.Models.GenerateContent(ctx, *model, contents, nil)
	if err != nil {
		return err
	}

	fmt.Println(response.Text())
	return nil
}

func feedbackPromptForTone(tone string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(tone)) {
	case "nice":
		return promptNice, nil
	case "normal":
		return promptNormal, nil
	case "harsh":
		return promptHarsh, nil
	case "super-harsh", "superharsh":
		return promptSuperHarsh, nil
	default:
		return "", fmt.Errorf("unknown tone %q (expected nice|normal|harsh|super-harsh)", tone)
	}
}

func detectVideoMIME(path string) (string, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if stat.IsDir() {
		return "", errors.New("video path is a directory")
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "video/mp4", nil
	}

	m := mime.TypeByExtension(ext)
	if m == "" {
		return "video/mp4", nil
	}
	if strings.Contains(m, ";") {
		m = strings.SplitN(m, ";", 2)[0]
	}
	return m, nil
}

func waitForActiveFile(ctx context.Context, client *genai.Client, file *genai.File, pollInterval time.Duration) (*genai.File, error) {
	for file.State == genai.FileStateUnspecified || file.State != genai.FileStateActive {
		log.Printf("Processing video... (state=%v)", file.State)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}

		var err error
		file, err = client.Files.Get(ctx, file.Name, nil)
		if err != nil {
			return nil, err
		}
	}

	return file, nil
}

func defaultConfigEnvPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "video-agent-skills", ".env")
}

func loadEnvFile(path string) error {
	if path == "" {
		return nil
	}
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if stat.IsDir() {
		return fmt.Errorf("env path is a directory: %s", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}
		val = strings.Trim(val, "\"'")
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
