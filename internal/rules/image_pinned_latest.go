package rules

import (
	"fmt"

	"github.com/lowplane/sevro/internal/parser"
)

// imagePinnedLatest fires when an image tag is empty (implicit
// `:latest`) or literally `latest`. Both are reproducibility hazards:
// the same tag can resolve to different bytes between deploys.
type imagePinnedLatest struct{}

func newImagePinnedLatest() Detector { return imagePinnedLatest{} }

func (imagePinnedLatest) ID() string   { return "image-pinned-latest" }
func (imagePinnedLatest) Name() string { return "Image pinned to :latest" }

func (imagePinnedLatest) Run(w parser.Workload) []Finding {
	if !w.Image.Set {
		return nil
	}
	if w.Image.Tag != "" && w.Image.Tag != "latest" {
		return nil
	}
	displayed := w.Image.String()
	if displayed == "" {
		displayed = "(no image tag)"
	}
	tagSuffix := "(no tag)"
	if w.Image.Tag == "latest" {
		tagSuffix = ":latest"
	}
	return []Finding{{
		DetectorID: "image-pinned-latest",
		Workload:   w.Name,
		Title:      "Image not pinned to a stable tag",
		Detail:     fmt.Sprintf("Image %s uses %s. The same tag can resolve to different bytes across deploys; rollback becomes ambiguous. Pin to an immutable digest or version tag.", displayed, tagSuffix),
		Severity:   SeverityMed,
		Confidence: ConfidenceHigh,
	}}
}
