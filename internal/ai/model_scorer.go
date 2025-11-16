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
	modelPath  string
	ffmpeg     *ffmpeg.Executor
	inputShape ort.Shape
	session    *ort.DynamicAdvancedSession
}

// NewCLIPScorer creates a new CLIP-based scorer.
func NewCLIPScorer(logger zerolog.Logger, ffmpegExec *ffmpeg.Executor, modelPath string) (*CLIPScorer, error) {
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("model file not found: %s", modelPath)
	}

	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("failed to initialize ONNX runtime: %w", err)
	}

	// We can omit names here and just rely on order when calling Run().
	// However, the API requires names if we use Run, so we pass the known ones.
	inputNames := []string{"input_ids", "attention_mask", "pixel_values"}
	outputNames := []string{"logits_per_image"} // first output in HF example

	sess, err := ort.NewDynamicAdvancedSession(
		modelPath,
		inputNames,
		outputNames,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CLIP session: %w", err)
	}

	logger.Info().
		Str("model", modelPath).
		Strs("inputs", inputNames).
		Strs("outputs", outputNames).
		Msg("CLIP model loaded")

	return &CLIPScorer{
		logger:     logger.With().Str("scorer", "clip").Logger(),
		modelPath:  modelPath,
		ffmpeg:     ffmpegExec,
		inputShape: ort.NewShape(1, 3, 224, 224),
		session:    sess,
	}, nil
}

// Score runs CLIP on a keyframe and converts logits_per_image to a score.
func (c *CLIPScorer) Score(ctx context.Context, clip *clips.Clip) (float64, error) {
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

	// TEXT -> input_ids, attention_mask
	// For now, use a dummy "viral video" prompt tokenized as some non-zero IDs.
	// Later you can replace with real tokenizer IDs exported from Python.
	const seqLen = 16
	inputIDs := make([]int64, seqLen)
	attnMask := make([]int64, seqLen)
	for i := range inputIDs {
		inputIDs[i] = 1 // fake token id
		attnMask[i] = 1 // mark as valid
	}
	inputIDsShape := ort.NewShape(1, seqLen)
	attnMaskShape := ort.NewShape(1, seqLen)

	inputIDsTensor, err := ort.NewTensor(inputIDsShape, inputIDs)
	if err != nil {
		return 0.0, fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	defer inputIDsTensor.Destroy()

	attnMaskTensor, err := ort.NewTensor(attnMaskShape, attnMask)
	if err != nil {
		return 0.0, fmt.Errorf("failed to create attention_mask tensor: %w", err)
	}
	defer attnMaskTensor.Destroy()

	// OUTPUT -> logits_per_image
	// Check in Netron what shape logits_per_image has; many CLIP exports use [1, N].
	// Start with [1,1]; if Netron shows [1, 2] (for 2 texts), adjust to that.
	logitsShape := ort.NewShape(1, 1)
	logitsTensor, err := ort.NewEmptyTensor[float32](logitsShape)
	if err != nil {
		return 0.0, fmt.Errorf("failed to create logits_per_image tensor: %w", err)
	}
	defer logitsTensor.Destroy()

	// Run inference: order of inputs/outputs must match names we gave the session.
	inputs := []ort.ArbitraryTensor{inputIDsTensor, attnMaskTensor, pixelTensor}
	outputs := []ort.ArbitraryTensor{logitsTensor}
	if err := c.session.Run(inputs, outputs); err != nil {
		return 0.0, fmt.Errorf("CLIP inference failed: %w", err)
	}

	logits := logitsTensor.GetData()
	if len(logits) == 0 {
		return 0.0, fmt.Errorf("unexpected logits_per_image tensor")
	}

	// Use first logit, convert to 0-1 via sigmoid (you could also softmax if multiple texts)
	logit := float64(logits[0])
	score := 1.0 / (1.0 + math.Exp(-logit))

	c.logger.Debug().
		Str("clip", clip.ID).
		Float64("clip_clip_logit", logit).
		Float64("clip_score", score).
		Msg("CLIP scoring complete")

	clip.Metadata["clip_logits_per_image"] = logit
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

// Close releases CLIP session and ONNX env.
func (c *CLIPScorer) Close() error {
	c.logger.Info().Msg("closing CLIP model session")
	if c.session != nil {
		if err := c.session.Destroy(); err != nil {
			return err
		}
	}
	return ort.DestroyEnvironment()
}
