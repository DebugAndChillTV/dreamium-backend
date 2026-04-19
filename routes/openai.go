package routes

import (
	"dreamium-backend/middleware"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	openai "github.com/sashabaranov/go-openai"
)

var openaiClient *openai.Client

// Model IDs per OpenAI docs (explicit to avoid deprecated library constants).
// gpt-4o-mini: recommended over gpt-3.5-turbo, cheaper and supported on more tiers.
// gpt-4o: current flagship chat model, recommended over gpt-4-turbo.
const (
	modelFast  = "gpt-4o-mini" // dream validation & language detection
	modelSmart = "gpt-4o"      // dream analysis & interpretations
)

// Initialize OpenAI client
func InitOpenAI() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		panic("Missing OPENAI_API_KEY in environment variables")
	}

	openaiClient = openai.NewClient(apiKey)
}

func SaveDream(userID, userDream, language string, keywords []string, mood string) error {
	// Get Supabase client
	client := middleware.GetSupabaseClient()
	if client == nil {
		fmt.Println("Supabase client not available")
		return fmt.Errorf("database not available")
	}

	// Create dream record matching dreams_rows.json format
	dreamRecord := map[string]interface{}{
		"id":         uuid.New().String(),
		"created_at": time.Now().UTC().Format("2006-01-02 15:04:05.000000+00"),
		"dream_tags": keywords,
		"mood":       mood,
		"user_id":    userID,
		"dream":      userDream,
		"language":   language,
	}

	// Insert into dreams table
	_, _, err := client.From("dreams").Insert(dreamRecord, false, "", "*", "").Execute()
	if err != nil {
		fmt.Printf("Error saving dream to database: %v\n", err)
		return err
	}

	fmt.Printf("Dream saved successfully - UserID: %s, Language: %s, Keywords: %v, Mood: %s\n",
		userID, language, keywords, mood)
	return nil
}

func GetDreams(userID string) ([]map[string]interface{}, error) {
	client := middleware.GetSupabaseClient()
	if client == nil {
		return nil, fmt.Errorf("database not available")
	}

	rows, _, err := client.From("dreams").Select("*", "", false).Eq("user_id", userID).Execute()
	if err != nil {
		return nil, fmt.Errorf("error fetching dreams: %v", err)
	}

	var allDreams []map[string]interface{}
	if err := json.Unmarshal(rows, &allDreams); err != nil {
		return nil, fmt.Errorf("error parsing dreams: %v", err)
	}

	// Skip the latest dream and return the next 5
	if len(allDreams) <= 1 {
		return []map[string]interface{}{}, nil
	}

	start := 1 // Skip the most recent dream
	end := start + 5
	if end > len(allDreams) {
		end = len(allDreams)
	}

	return allDreams[start:end], nil
}

// Check if input is a dream & detect language
func IsDreamInput(c *fiber.Ctx) error {
	log.Printf("api: POST /api/isDreamInput received")
	var req struct {
		UserInput string `json:"userInput"`
		UserId    string `json:"userId"`
	}
	if err := c.BodyParser(&req); err != nil {
		log.Printf("api: IsDreamInput body parse error: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	resp, err := openaiClient.CreateChatCompletion(c.Context(), openai.ChatCompletionRequest{
		Model: modelFast,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: `Determine if the user's input is a valid dream description.
						 Then, detect the input language.
						 Extract psychology-related keywords from the dream (emotions, symbols, actions).
						 Determine the general mood of the dream (positive, negative, neutral, anxious, peaceful, etc.).
						 Return JSON like: {"valid": true/false, "language": "English", "keywords": ["fear", "chase", "darkness"], "mood": "anxious"}.`,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: req.UserInput,
			},
		},
		MaxTokens: 200,
	})
	if err != nil {
		log.Printf("api: IsDreamInput OpenAI error: %v", err)
		var apiErr *openai.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPStatusCode == 429 {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "OpenAI quota exceeded. Check that OPENAI_API_KEY in Railway is from the account with credits, and that billing is set up at platform.openai.com.",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if len(resp.Choices) == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "OpenAI returned no completion choices"})
	}

	response := resp.Choices[0].Message.Content

	var dreamData struct {
		UserId   string   `json:"userId"`
		Valid    bool     `json:"valid"`
		Language string   `json:"language"`
		Keywords []string `json:"keywords"`
		Mood     string   `json:"mood"`
	}
	if err := json.Unmarshal([]byte(response), &dreamData); err == nil && dreamData.Valid {
		SaveDream(req.UserId, req.UserInput, dreamData.Language, dreamData.Keywords, dreamData.Mood)
	}

	log.Printf("api: IsDreamInput success")
	return c.JSON(response)
}

// Generate psychological dream analysis
func GenerateDreamAnalysis(c *fiber.Ctx) error {
	var req struct {
		UserDream        string `json:"userDream"`
		DetectedLanguage string `json:"detectedLanguage"`
		UserId           string `json:"userId"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	dreams, err := GetDreams(req.UserId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Extract last 5 dreams' tags and moods for context
	var previousDreamsContext string
	if len(dreams) > 0 {
		limit := 5
		if len(dreams) < 5 {
			limit = len(dreams)
		}

		var allTags []string
		var allMoods []string

		for i := 0; i < limit; i++ {
			if tags, ok := dreams[i]["dream_tags"].([]interface{}); ok {
				for _, tag := range tags {
					if tagStr, ok := tag.(string); ok {
						allTags = append(allTags, tagStr)
					}
				}
			}
			if mood, ok := dreams[i]["mood"].(string); ok && mood != "" {
				allMoods = append(allMoods, mood)
			}
		}

		if len(allTags) > 0 || len(allMoods) > 0 {
			previousDreamsContext = fmt.Sprintf("\n\nUser's last %d dream patterns:\n- Previous dream tags: %v\n- Previous dream moods: %v\n- Look for repetitive themes and emotional patterns and imply them on the analysis if necessary.", limit, allTags, allMoods)
		}
	}

	content := fmt.Sprintf(`You are a dream analyst and psychologist. Analyze the user's dream in %s based on psychological theories such as Freud, Jung, Adler, Medard Boss, Calvin S. Hall, and Rosalind Cartwright.
- Identify subconscious patterns beyond direct keyword matches.
- If any of these themes relate to the new dream, explain their psychological significance.
- Only mention theorists if their perspective is highly relevant; otherwise, provide an insightful interpretation in simple terms.
- Keep the response concise, engaging, and easy to understand (6-7 sentences max).%s`, req.DetectedLanguage, previousDreamsContext)

	resp, err := openaiClient.CreateChatCompletion(c.Context(), openai.ChatCompletionRequest{
		Model: modelSmart,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: content,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: req.UserDream,
			},
		},
		MaxTokens: 600,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Choices[0].Message.Content)
}

// Generate symbolic interpretation
func GenerateSymbolicInterpretation(c *fiber.Ctx) error {
	var req struct {
		UserDream        string `json:"userDream"`
		DetectedLanguage string `json:"detectedLanguage"`
		UserId           string `json:"userId"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	content := fmt.Sprintf(
		`Provide a symbolic interpretation of the user's dream based on common dream meanings and folklore in %s.
                  If relevant, consider mythological or cultural significance.
                  Keep the response concise, engaging, and easy to understand (5-6 sentences max).`,
		req.DetectedLanguage,
	)

	resp, err := openaiClient.CreateChatCompletion(c.Context(), openai.ChatCompletionRequest{
		Model: modelSmart,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: content,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: req.UserDream,
			},
		},
		MaxTokens: 400,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Choices[0].Message.Content)
}

// Generate psychologist interpretation
func GeneratePsychologistInterpretation(c *fiber.Ctx) error {
	var req struct {
		UserDream        string `json:"userDream"`
		DetectedLanguage string `json:"detectedLanguage"`
		Psychologist     string `json:"psychologist"`
		UserId           string `json:"userId"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	dreams, err := GetDreams(req.UserId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Extract last 5 dreams' tags and moods for context
	var previousDreamsContext string
	if len(dreams) > 0 {
		limit := 5
		if len(dreams) < 5 {
			limit = len(dreams)
		}

		var allTags []string
		var allMoods []string

		for i := 0; i < limit; i++ {
			if tags, ok := dreams[i]["dream_tags"].([]interface{}); ok {
				for _, tag := range tags {
					if tagStr, ok := tag.(string); ok {
						allTags = append(allTags, tagStr)
					}
				}
			}
			if mood, ok := dreams[i]["mood"].(string); ok && mood != "" {
				allMoods = append(allMoods, mood)
			}
		}

		if len(allTags) > 0 || len(allMoods) > 0 {
			previousDreamsContext = fmt.Sprintf("\n\nUser's last %d dream patterns:\n- Previous dream tags: %v\n- Previous dream moods: %v\n- Look for repetitive themes and emotional patterns and imply them on the analysis if necessary.", limit, allTags, allMoods)
		}
	}

	content := fmt.Sprintf(
		`You are a dream analyst and psychologist. Analyze the user's dream in %s based on psychological theories from %s.
		- Identify subconscious patterns beyond direct keyword matches.
		- If any of these themes relate to the new dream, explain their psychological significance just reference the %s.
		- Keep the response concise, engaging, and easy to understand (6-7 sentences max).%s`,
		req.DetectedLanguage, req.Psychologist, req.Psychologist, previousDreamsContext,
	)

	resp, err := openaiClient.CreateChatCompletion(c.Context(), openai.ChatCompletionRequest{
		Model: modelSmart,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: content,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: req.UserDream,
			},
		},
		MaxTokens: 600,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Choices[0].Message.Content)
}
