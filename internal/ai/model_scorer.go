package ai

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/keagan/slopcannon/internal/clips"
	"github.com/keagan/slopcannon/internal/ffmpeg"
	"github.com/nfnt/resize"
	"github.com/rs/zerolog"
	ort "github.com/yalue/onnxruntime_go"
)

// CLIPScorer uses the sayantan47/clip-vit-b32-onnx model.
type CLIPScorer struct {
	logger     zerolog.Logger
	ffmpeg     *ffmpeg.Executor
	inputShape ort.Shape

	encoderSession *ort.DynamicAdvancedSession
	headSession    *ort.DynamicAdvancedSession
}

var onnxInitOnce sync.Once
var onnxInitErr error

func init() {
	// Adjust this path to where brew installed your dylib.
	// You can find it via: `brew info onnxruntime` or `ls /opt/homebrew/lib | grep onnxruntime`.
	ort.SetSharedLibraryPath("/usr/local/lib/libonnxruntime.1.22.2.dylib")
}

// NewCLIPScorer creates a new CLIP-based scorer using image encoder + virality head.
func NewCLIPScorer(
	logger zerolog.Logger,
	ffmpegExec *ffmpeg.Executor,
	encoderModelPath string,
	headModelPath string,
) (*CLIPScorer, error) {
	if _, err := os.Stat(encoderModelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("encoder model file not found: %s", encoderModelPath)
	}
	if _, err := os.Stat(headModelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("head model file not found: %s", headModelPath)
	}

	// Initialize ONNX Runtime only once per process
	onnxInitOnce.Do(func() {
		onnxInitErr = ort.InitializeEnvironment()
	})
	if onnxInitErr != nil {
		return nil, fmt.Errorf("failed to initialize ONNX runtime: %w", onnxInitErr)
	}

	encoderSession, err := ort.NewDynamicAdvancedSession(
		encoderModelPath,
		[]string{"pixel_values"},
		[]string{"image_embeds"},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CLIP image encoder session: %w", err)
	}

	headSession, err := ort.NewDynamicAdvancedSession(
		headModelPath,
		[]string{"image_embeds"},
		[]string{"score_logits"}, // or "score"
		nil,
	)
	if err != nil {
		encoderSession.Destroy()
		return nil, fmt.Errorf("failed to create virality head session: %w", err)
	}

	logger.Info().
		Str("encoder_model", encoderModelPath).
		Str("head_model", headModelPath).
		Msg("CLIP encoder + virality head models loaded")

	return &CLIPScorer{
		logger:         logger.With().Str("scorer", "clip").Logger(),
		ffmpeg:         ffmpegExec,
		inputShape:     ort.NewShape(1, 3, 224, 224),
		encoderSession: encoderSession,
		headSession:    headSession,
	}, nil
}

// Score runs CLIP image encoder + virality head on a keyframe.
func (c *CLIPScorer) Score(ctx context.Context, clip *clips.Clip) (float64, error) {
	// Extract keyframe from middle of clip
	keyframeTime := clip.Start + (clip.Duration / 2)
	keyframePath := filepath.Join(os.TempDir(),
		fmt.Sprintf("clip_keyframe_%s_%d.jpg", clip.ID, time.Now().UnixNano()))
	defer os.Remove(keyframePath)

	if err := c.ffmpeg.ExtractFrame(ctx, clip.SourceURL, keyframeTime, keyframePath); err != nil {
		c.logger.Warn().Err(err).Str("clip", clip.ID).Msg("keyframe extraction failed")
		return 0.0, err
	}

	// IMAGE -> pixel_values
	pixelTensor, err := c.preprocessImage(keyframePath)
	if err != nil {
		return 0.0, fmt.Errorf("image preprocessing failed: %w", err)
	}
	defer pixelTensor.Destroy()

	// 1) Run image encoder: pixel_values -> image_embeds
	// Match this to the actual dimension of your ONNX encoder output.
	const embedDim = 512
	embedShape := ort.NewShape(1, embedDim)
	embedTensor, err := ort.NewEmptyTensor[float32](embedShape)
	if err != nil {
		return 0.0, fmt.Errorf("failed to create image_embeds tensor: %w", err)
	}
	defer embedTensor.Destroy()

	if err := c.encoderSession.Run(
		[]ort.ArbitraryTensor{pixelTensor},
		[]ort.ArbitraryTensor{embedTensor},
	); err != nil {
		return 0.0, fmt.Errorf("CLIP image encoder inference failed: %w", err)
	}

	// 2) Run virality head: image_embeds -> score_logits (or score)
	scoreTensor, err := ort.NewEmptyTensor[float32](ort.NewShape(1, 1))
	if err != nil {
		return 0.0, fmt.Errorf("failed to create score tensor: %w", err)
	}
	defer scoreTensor.Destroy()

	if err := c.headSession.Run(
		[]ort.ArbitraryTensor{embedTensor},
		[]ort.ArbitraryTensor{scoreTensor},
	); err != nil {
		return 0.0, fmt.Errorf("virality head inference failed: %w", err)
	}

	data := scoreTensor.GetData()
	if len(data) != 1 {
		return 0.0, fmt.Errorf("unexpected score tensor size: %d", len(data))
	}

	// If your ONNX head outputs logits, apply sigmoid here.
	logit := float64(data[0])
	score := 1.0 / (1.0 + math.Exp(-logit))

	c.logger.Debug().
		Str("clip", clip.ID).
		Float64("clip_clip_logit", logit).
		Float64("clip_score", score).
		Msg("CLIP virality scoring complete")

	clip.Metadata["clip_score"] = score
	return score, nil
}

// preprocessImage -> pixel_values (float32[1,3,224,224]) with CLIP normalization.
func (c *CLIPScorer) preprocessImage(imagePath string) (ort.ArbitraryTensor, error) {
	f, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	resized := resize.Resize(224, 224, img, resize.Bilinear)

	data := make([]float32, 3*224*224)
	mean := []float32{0.48145466, 0.4578275, 0.40821073}
	std := []float32{0.26862954, 0.26130258, 0.27577711}

	bounds := resized.Bounds()
	idx := 0

	for ch := 0; ch < 3; ch++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				r, g, b, _ := resized.At(x, y).RGBA()
				var v float32
				switch ch {
				case 0:
					v = float32(r>>8) / 255.0
				case 1:
					v = float32(g>>8) / 255.0
				case 2:
					v = float32(b>>8) / 255.0
				}
				data[idx] = (v - mean[ch]) / std[ch]
				idx++
			}
		}
	}

	return ort.NewTensor(c.inputShape, data)
}

// Close releases ONNX sessions and environment.
func (c *CLIPScorer) Close() error {
	c.logger.Info().Msg("closing CLIP encoder + head sessions")
	if c.encoderSession != nil {
		if err := c.encoderSession.Destroy(); err != nil {
			return err
		}
	}
	if c.headSession != nil {
		if err := c.headSession.Destroy(); err != nil {
			return err
		}
	}
	// Do NOT call ort.DestroyEnvironment() here; it is process-wide.
	return nil
}
