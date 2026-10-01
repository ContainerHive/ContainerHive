package rendering

import (
	"log/slog"
	"maps"
	"slices"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// baseAliasCandidates returns the base tag names for an image, excluding
// variant suffixes, as alias candidates in deterministic sorted order (see
// AliasCandidatesFor).
func baseAliasCandidates(imageDef *model.Image) []AliasCandidate {
	candidates := make([]AliasCandidate, 0, len(imageDef.Tags))
	for _, tagName := range slices.Sorted(maps.Keys(imageDef.Tags)) {
		candidates = append(candidates, AliasCandidate{
			Name:         tagName,
			IsPrerelease: imageDef.Tags[tagName].IsPrerelease,
		})
	}
	return candidates
}

// ResolveAllAliases computes every alias of a single image definition,
// mapping alias name to the exact tag it points at: the semantic version
// series aliases from ResolveImageAliases plus the latest_alias for the base
// tags and for each variant. It is the single source of truth for both
// registry retagging and the CI template context.
//
// A latest_alias that cannot be resolved is handled per its on_missing
// setting: "silent" and "warning" skip it, anything else returns the error.
func ResolveAllAliases(imageDef *model.Image) (map[string]string, error) {
	aliases := ResolveImageAliases(imageDef)

	if imageDef.LatestAlias == nil {
		return aliases, nil
	}

	latestTarget, err := ResolveLatestAliasFor(baseAliasCandidates(imageDef), imageDef.LatestAlias.Tag)
	if err != nil {
		switch imageDef.LatestAlias.OnMissing {
		case "silent":
			// do nothing
		case "warning":
			slog.Warn("Latest alias resolution failed", "error", err)
		default: // "error" or unset
			return nil, err
		}
	} else {
		aliases[imageDef.LatestAlias.Tag] = latestTarget
	}

	for _, variantName := range slices.Sorted(maps.Keys(imageDef.Variants)) {
		variantDef := imageDef.Variants[variantName]
		variantTags := make([]AliasCandidate, 0, len(imageDef.Tags))
		for _, tagName := range slices.Sorted(maps.Keys(imageDef.Tags)) {
			variantTags = append(variantTags, AliasCandidate{
				Name:         tagName + variantDef.TagSuffix,
				IsPrerelease: imageDef.Tags[tagName].IsPrerelease,
			})
		}
		variantTarget, err := ResolveLatestAliasFor(variantTags, imageDef.LatestAlias.Tag)
		if err != nil {
			switch imageDef.LatestAlias.OnMissing {
			case "silent":
				// do nothing
			case "warning":
				slog.Warn("Latest alias resolution failed for variant", "variant", variantDef.Name, "error", err)
			default: // "error" or unset
				return nil, err
			}
		} else {
			aliases[imageDef.LatestAlias.Tag+variantDef.TagSuffix] = variantTarget
		}
	}

	return aliases, nil
}
