package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/genai"
)

const (
	defaultModel  = "gemini-3-flash-preview"
	defaultPrompt = "You are a senior motion designer and creative director. Be direct and a bit harsh. Critique storytelling, camera movement, visibility/legibility of key elements, timing, and overall professional polish, plus pacing, rhythm, transitions, typography, composition, color, and easing. Call out what looks amateurish or generic. Provide more improvement ideas than praise. Output format:\n1) Quick read (2-3 sentences)\n2) Strengths (2 bullets max)\n3) Issues (6 bullets)\n4) Improvements (6 bullets, each with a concrete fix; include approximate timecodes if possible)."
)

func main() {
	var (
		videoPath = flag.String("video", "", "Path to a video file")
		model     = flag.String("model", defaultModel, "Gemini model name")
		prompt    = flag.String("prompt", defaultPrompt, "Prompt to guide feedback")
		apiKey    = flag.String("api-key", "", "Gemini API key (overrides GEMINI_API_KEY/GOOGLE_API_KEY)")
	)
	flag.Parse()

	if *videoPath == "" {
		log.Fatal("-video is required")
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
		log.Fatal("missing API key: set GEMINI_API_KEY or GOOGLE_API_KEY, or pass -api-key")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  key,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}

	mimeType, err := detectVideoMIME(*videoPath)
	if err != nil {
		log.Fatal(err)
	}

	uploaded, err := client.Files.UploadFromPath(
		ctx,
		*videoPath,
		&genai.UploadFileConfig{
			MIMEType: mimeType,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	uploaded, err = waitForActiveFile(ctx, client, uploaded, 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	parts := []*genai.Part{
		genai.NewPartFromURI(uploaded.URI, uploaded.MIMEType),
		genai.NewPartFromText(*prompt),
	}

	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	response, err := client.Models.GenerateContent(ctx, *model, contents, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response.Text())
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
